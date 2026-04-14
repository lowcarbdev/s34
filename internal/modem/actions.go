package modem

import (
	"fmt"
	"strings"
)

// Restart reboots the modem. The session must be authenticated first.
func (c *Client) Restart() error {
	resp, err := c.Do("SetStatusSecuritySettings", map[string]any{
		"SetStatusSecuritySettings": map[string]any{
			"MotoReboot": "reboot",
		},
	})
	if err != nil {
		return err
	}
	if result := actionResult(resp, "SetStatusSecuritySettings"); result != "OK" {
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

// StatusConnection returns the modem connection info (upstream/downstream lock, etc.).
func (c *Client) StatusConnection() (map[string]any, error) {
	return c.Do("GetCustomerStatusConnectionInfo", map[string]any{
		"GetCustomerStatusConnectionInfo": "",
	})
}

// StatusDownstream returns downstream channel bonding information.
func (c *Client) StatusDownstream() (map[string]any, error) {
	return c.Do("GetCustomerStatusDownstreamChannelInfo", map[string]any{
		"GetCustomerStatusDownstreamChannelInfo": "",
	})
}

// StatusUpstream returns upstream channel bonding information.
func (c *Client) StatusUpstream() (map[string]any, error) {
	return c.Do("GetCustomerStatusUpstreamChannelInfo", map[string]any{
		"GetCustomerStatusUpstreamChannelInfo": "",
	})
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
