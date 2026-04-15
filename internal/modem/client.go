// Package modem provides a client for interacting with the ARRIS SURFboard S34
// cable modem via its HNAP JSON API over HTTPS.
package modem

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://192.168.100.1"
	hnapPath       = "/HNAP1/"
	hnapNS         = "http://purenetworks.com/HNAP1/"
)

// ErrReload is returned by Login when the modem responds with RELOAD,
// indicating it is still starting up and the caller should retry.
var ErrReload = fmt.Errorf("modem is starting up")

// Client holds the HTTP client and session state for the modem.
type Client struct {
	http       *http.Client
	tlsCfg     *tls.Config
	host       string // host:port, derived from baseURL
	baseURL    string
	privateKey string // SHA-256 hex uppercase; used as HMAC key for HNAP_AUTH
	uid        string // session cookie value
}

// NewClient creates a new modem client. TLS verification is skipped because
// the modem uses a self-signed certificate issued by an internal ARRIS CA.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	// Derive host:port for raw TLS connections.
	host := strings.TrimPrefix(baseURL, "https://")
	host = strings.TrimPrefix(host, "http://")
	if !strings.Contains(host, ":") {
		if strings.HasPrefix(baseURL, "https://") {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	tlsCfg := &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	return &Client{
		baseURL: baseURL,
		host:    host,
		tlsCfg:  tlsCfg,
		http: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsCfg,
			},
			// Do not follow redirects — unauthenticated requests redirect to
			// /Login.html; surface that as an error instead.
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Login authenticates against the modem using the two-phase HNAP
// challenge/response scheme.
//
// Algorithm (observed from browser traffic):
//
//	Phase 1 — request challenge:
//	  POST {"Login":{"Action":"request","Username":...,"LoginPassword":"","Captcha":""}}
//	  Response: {"Challenge":"...", "Cookie":"...", "PublicKey":"...", "LoginResult":"OK"}
//
//	Key derivation:
//	  PrivateKey    = UPPER(HMAC-SHA-256(key=PublicKey+Password, msg=Challenge))
//	  LoginPassword = UPPER(HMAC-SHA-256(key=PrivateKey,         msg=Challenge))
//
//	Phase 2 — authenticate:
//	  Cookie header: uid=<Cookie>; PrivateKey=<PrivateKey>
//	  HNAP_AUTH header: UPPER(HMAC-SHA-256(key=PrivateKey, msg=ts+"<SOAPAction>")) + " " + ts
//	  POST {"Login":{"Action":"login","Username":...,"LoginPassword":<LoginPassword>,
//	                 "Captcha":"","PrivateLogin":"LoginPassword"}}
func (c *Client) Login(username, password string) error {
	// Phase 1: request challenge
	phase1, err := c.hnapPost("Login", map[string]any{
		"Login": map[string]any{
			"Action":        "request",
			"Username":      username,
			"LoginPassword": "",
			"Captcha":       "",
		},
	}, "", "")
	if err != nil {
		return fmt.Errorf("login phase 1: %w", err)
	}

	lr, ok := phase1["LoginResponse"].(map[string]any)
	if !ok {
		return fmt.Errorf("login phase 1: unexpected response shape")
	}
	switch result, _ := lr["LoginResult"].(string); result {
	case "LOCKUP":
		return fmt.Errorf("account locked: too many failed attempts — wait a few minutes and try again")
	case "RELOAD":
		return ErrReload
	case "OK":
		// continue
	default:
		return fmt.Errorf("login phase 1 failed: %s", result)
	}

	challenge, _ := lr["Challenge"].(string)
	publicKey, _ := lr["PublicKey"].(string)
	cookie, _ := lr["Cookie"].(string)
	if challenge == "" || publicKey == "" || cookie == "" {
		return fmt.Errorf("login phase 1: missing challenge fields in response")
	}

	// Derive keys.
	//   PrivateKey    = UPPER(HMAC-SHA-256(key=PublicKey+Password, msg=Challenge))
	//   LoginPassword = UPPER(HMAC-SHA-256(key=PrivateKey,         msg=Challenge))
	privateKey := hmacSHA256Upper(publicKey+password, challenge)
	loginPassword := hmacSHA256Upper(privateKey, challenge)

	// Phase 2: authenticate
	cookieHdr := fmt.Sprintf("uid=%s; PrivateKey=%s", cookie, privateKey)
	phase2, err := c.hnapPost("Login", map[string]any{
		"Login": map[string]any{
			"Action":        "login",
			"Username":      username,
			"LoginPassword": loginPassword,
			"Captcha":       "",
			"PrivateLogin":  "LoginPassword",
		},
	}, cookieHdr, privateKey)
	if err != nil {
		return fmt.Errorf("login phase 2: %w", err)
	}

	lr2, ok := phase2["LoginResponse"].(map[string]any)
	if !ok {
		return fmt.Errorf("login phase 2: unexpected response shape")
	}
	switch result, _ := lr2["LoginResult"].(string); result {
	case "LOCKUP":
		return fmt.Errorf("account locked: too many failed attempts — wait a few minutes and try again")
	case "OK":
		// authenticated
	default:
		return fmt.Errorf("authentication failed (wrong password?): %s", result)
	}

	c.privateKey = privateKey
	c.uid = cookie
	return nil
}

// Do executes an authenticated HNAP action and returns the parsed JSON response.
func (c *Client) Do(action string, body map[string]any) (map[string]any, error) {
	if c.privateKey == "" {
		return nil, fmt.Errorf("not authenticated — call Login first")
	}
	cookieHdr := fmt.Sprintf("uid=%s; PrivateKey=%s", c.uid, c.privateKey)
	return c.hnapPost(action, body, cookieHdr, c.privateKey)
}

// DoRaw executes an authenticated HNAP action using a raw TLS connection,
// bypassing Go's HTTP header parser. This is required for the downstream
// channel endpoint, which injects channel data into an HTTP header line,
// producing a response that Go's net/http rejects as malformed.
func (c *Client) DoRaw(action string, body map[string]any) (map[string]any, error) {
	if c.privateKey == "" {
		return nil, fmt.Errorf("not authenticated — call Login first")
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	soapAction := fmt.Sprintf(`"%s%s"`, hnapNS, action)
	cookieHdr := fmt.Sprintf("uid=%s; PrivateKey=%s", c.uid, c.privateKey)
	auth := hnapAuth(c.privateKey, soapAction)

	req := fmt.Sprintf(
		"POST %s HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"SOAPAction: %s\r\n"+
			"HNAP_AUTH: %s\r\n"+
			"Cookie: %s\r\n"+
			"Content-Length: %d\r\n"+
			"Connection: close\r\n"+
			"\r\n"+
			"%s",
		hnapPath,
		strings.Split(c.host, ":")[0],
		soapAction,
		auth,
		cookieHdr,
		len(payload),
		payload,
	)

	conn, err := tls.Dial("tcp", c.host, c.tlsCfg)
	if err != nil {
		return nil, fmt.Errorf("TLS dial: %w", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(30 * time.Second)) //nolint:errcheck

	if _, err := io.WriteString(conn, req); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	// Read the raw response, skipping headers line by line until the blank
	// separator, then collect the body. The modem corrupts one header line
	// with channel data, so we cannot use net/http to parse this response.
	br := bufio.NewReader(conn)

	// Read status line
	statusLine, err := br.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read status line: %w", err)
	}
	if !strings.Contains(statusLine, "200") {
		return nil, fmt.Errorf("modem returned non-200: %s", strings.TrimSpace(statusLine))
	}

	// Skip all header lines until blank line
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("read headers: %w", err)
		}
		if strings.TrimSpace(line) == "" {
			break
		}
	}

	// Read body
	rawBody, err := io.ReadAll(br)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var result map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(rawBody), &result); err != nil {
		return nil, fmt.Errorf("unexpected modem response: %w (body: %s)", err, rawBody)
	}
	return result, nil
}

// hnapPost sends a POST to /HNAP1/ for the given action.
//
//   - cookie is the raw Cookie header value (empty = omit header).
//   - privateKey, when non-empty, causes an HNAP_AUTH header to be computed and sent.
//
// The modem only parses the body as JSON when Content-Type is absent;
// sending application/json yields 404, so we intentionally omit it.
func (c *Client) hnapPost(action string, body map[string]any, cookie, privateKey string) (map[string]any, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+hnapPath, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	soapAction := fmt.Sprintf(`"%s%s"`, hnapNS, action)
	req.Header.Set("SOAPAction", soapAction)

	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}

	if privateKey != "" {
		req.Header.Set("HNAP_AUTH", hnapAuth(privateKey, soapAction))
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from modem", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("unexpected modem response: %w (body: %s)", err, raw)
	}
	return result, nil
}

// hnapAuth computes the HNAP_AUTH header value:
//
//	UPPER(HMAC-SHA-256(key=privateKey, msg=timestampMs+soapAction)) + " " + timestampMs
//
// where soapAction is the quoted URI string, e.g. `"http://purenetworks.com/HNAP1/Login"`.
func hnapAuth(privateKey, soapAction string) string {
	ts := fmt.Sprintf("%d", time.Now().UnixMilli())
	return hmacSHA256Upper(privateKey, ts+soapAction) + " " + ts
}

// hmacSHA256Upper returns UPPER(HEX(HMAC-SHA-256(key=key, msg=msg))).
func hmacSHA256Upper(key, msg string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(msg))
	return strings.ToUpper(fmt.Sprintf("%x", mac.Sum(nil)))
}
