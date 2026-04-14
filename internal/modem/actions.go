package modem

import (
	"fmt"
	"strings"
)

// DownstreamChannel holds parsed info for one downstream bonded channel.
type DownstreamChannel struct {
	Channel     string
	Status      string
	Modulation  string
	ID          string
	FreqHz      string
	PowerDBmV   string
	SNRDB       string
	Corrected   string
	Uncorrected string
}

// UpstreamChannel holds parsed info for one upstream bonded channel.
type UpstreamChannel struct {
	Channel     string
	Status      string
	Type        string
	ID          string
	SymRateKsym string
	FreqHz      string
	PowerDBmV   string
}

// DeviceInfo holds the consolidated device identity fields from GetMultipleHNAPs.
type DeviceInfo struct {
	ModelName          string
	SerialNumber       string
	MACAddress         string
	FirmwareVersion    string
	HardwareVersion    string
	InternetConnection string
	NetworkAccess      string
	Uptime             string
	SystemTime         string
}

// GetDeviceInfo fetches device identity fields in a single GetMultipleHNAPs request.
func (c *Client) GetDeviceInfo() (*DeviceInfo, error) {
	resp, err := c.Do("GetMultipleHNAPs", map[string]any{
		"GetMultipleHNAPs": map[string]any{
			"GetInternetConnectionStatus":     "",
			"GetArrisRegisterInfo":            "",
			"GetCustomerStatusSoftware":       "",
			"GetCustomerStatusConnectionInfo": "",
		},
	})
	if err != nil {
		return nil, err
	}

	outer, _ := resp["GetMultipleHNAPsResponse"].(map[string]any)

	str := func(section, key string) string {
		if s, ok := outer[section].(map[string]any); ok {
			v, _ := s[key].(string)
			return v
		}
		return ""
	}

	return &DeviceInfo{
		ModelName:          str("GetArrisRegisterInfoResponse", "ModelName"),
		SerialNumber:       str("GetArrisRegisterInfoResponse", "SerialNumber"),
		MACAddress:         str("GetArrisRegisterInfoResponse", "MacAddress"),
		FirmwareVersion:    str("GetCustomerStatusSoftwareResponse", "StatusSoftwareSfVer"),
		HardwareVersion:    str("GetCustomerStatusSoftwareResponse", "StatusSoftwareHdVer"),
		InternetConnection: str("GetInternetConnectionStatusResponse", "InternetConnection"),
		NetworkAccess:      str("GetCustomerStatusConnectionInfoResponse", "CustomerConnNetworkAccess"),
		Uptime:             str("GetCustomerStatusSoftwareResponse", "CustomerConnSystemUpTime"),
		SystemTime:         str("GetCustomerStatusConnectionInfoResponse", "CustomerCurSystemTime"),
	}, nil
}

// Restart reboots the modem. The session must be authenticated first.
func (c *Client) Restart() error {
	// Step 1: read current configuration so we can echo it back.
	// The modem requires the current EEE and LED values to be included in the
	// reboot request — it won't accept a bare reboot action without them.
	cfg, err := c.Do("GetMultipleHNAPs", map[string]any{
		"GetMultipleHNAPs": map[string]any{
			"GetArrisConfigurationInfo": "",
		},
	})
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	outer, _ := cfg["GetMultipleHNAPsResponse"].(map[string]any)
	info, _ := outer["GetArrisConfigurationInfoResponse"].(map[string]any)
	eee, _ := info["ethSWEthEEE"].(string)
	led, _ := info["LedStatus"].(string)
	if eee == "" {
		eee = "0"
	}
	if led == "" {
		led = "1"
	}

	// Step 2: send reboot with the current config values echoed back.
	resp, err := c.Do("SetArrisConfigurationInfo", map[string]any{
		"SetArrisConfigurationInfo": map[string]any{
			"Action":       "reboot",
			"SetEEEEnable": eee,
			"LED_Status":   led,
		},
	})
	if err != nil {
		return err
	}
	if result := actionResult(resp, "SetArrisConfigurationInfo"); result != "OK" {
		return fmt.Errorf("restart failed: %s", result)
	}
	return nil
}

// StatusStartup returns the modem startup sequence status.
func (c *Client) StatusStartup() (map[string]any, error) {
	return c.Do("GetCustomerStatusStartupSequence", map[string]any{
		"GetCustomerStatusStartupSequence": "",
	})
}

// StatusConnection returns the modem connection info.
func (c *Client) StatusConnection() (map[string]any, error) {
	return c.Do("GetCustomerStatusConnectionInfo", map[string]any{
		"GetCustomerStatusConnectionInfo": "",
	})
}

// StatusDownstream returns parsed downstream channel bonding information.
// It uses a raw TLS connection to work around a modem firmware bug that
// injects channel data into an HTTP response header.
func (c *Client) StatusDownstream() ([]DownstreamChannel, error) {
	resp, err := c.DoRaw("GetCustomerStatusDownstreamChannelInfo", map[string]any{
		"GetCustomerStatusDownstreamChannelInfo": "",
	})
	if err != nil {
		return nil, err
	}

	inner, _ := resp["GetCustomerStatusDownstreamChannelInfoResponse"].(map[string]any)
	blob, _ := inner["CustomerConnDownstreamChannel"].(string)
	if blob == "" {
		return nil, fmt.Errorf("empty downstream channel data")
	}

	// Format: ch^status^modulation^id^freqHz^power^snr^corrected^uncorrected|+|...
	var channels []DownstreamChannel
	for _, entry := range strings.Split(blob, "|+|") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		f := strings.Split(entry, "^")
		if len(f) < 9 {
			continue
		}
		channels = append(channels, DownstreamChannel{
			Channel:     strings.TrimSpace(f[0]),
			Status:      strings.TrimSpace(f[1]),
			Modulation:  strings.TrimSpace(f[2]),
			ID:          strings.TrimSpace(f[3]),
			FreqHz:      strings.TrimSpace(f[4]),
			PowerDBmV:   strings.TrimSpace(f[5]),
			SNRDB:       strings.TrimSpace(f[6]),
			Corrected:   strings.TrimSpace(f[7]),
			Uncorrected: strings.TrimSpace(f[8]),
		})
	}
	return channels, nil
}

// StatusUpstream returns parsed upstream channel bonding information.
func (c *Client) StatusUpstream() ([]UpstreamChannel, error) {
	resp, err := c.Do("GetCustomerStatusUpstreamChannelInfo", map[string]any{
		"GetCustomerStatusUpstreamChannelInfo": "",
	})
	if err != nil {
		return nil, err
	}

	inner, _ := resp["GetCustomerStatusUpstreamChannelInfoResponse"].(map[string]any)
	blob, _ := inner["CustomerConnUpstreamChannel"].(string)
	if blob == "" {
		return nil, fmt.Errorf("empty upstream channel data")
	}

	// Format: ch^status^type^id^symRateKsym^freqHz^power|+|...
	var channels []UpstreamChannel
	for _, entry := range strings.Split(blob, "|+|") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		f := strings.Split(entry, "^")
		if len(f) < 7 {
			continue
		}
		channels = append(channels, UpstreamChannel{
			Channel:     strings.TrimSpace(f[0]),
			Status:      strings.TrimSpace(f[1]),
			Type:        strings.TrimSpace(f[2]),
			ID:          strings.TrimSpace(f[3]),
			SymRateKsym: strings.TrimSpace(f[4]),
			FreqHz:      strings.TrimSpace(f[5]),
			PowerDBmV:   strings.TrimSpace(f[6]),
		})
	}
	return channels, nil
}

// EventLog returns the modem event log.
func (c *Client) EventLog() (map[string]any, error) {
	return c.Do("GetCustomerStatusLog", map[string]any{
		"GetCustomerStatusLog": "",
	})
}

// StatusSoftware returns firmware/software version information.
func (c *Client) StatusSoftware() (map[string]any, error) {
	return c.Do("GetCustomerStatusSoftware", map[string]any{
		"GetCustomerStatusSoftware": "",
	})
}

// actionResult extracts the <Action>Result field from an HNAP response map.
func actionResult(resp map[string]any, action string) string {
	key := action + "Response"
	if inner, ok := resp[key].(map[string]any); ok {
		if r, ok := inner[action+"Result"].(string); ok {
			return strings.TrimSpace(r)
		}
	}
	return ""
}
