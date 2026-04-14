package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show device identity (model, MAC, serial, firmware, uptime)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newAuthClient()
		if err != nil {
			return err
		}

		info, err := client.GetDeviceInfo()
		if err != nil {
			return err
		}

		if flagJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(info)
		}

		fmt.Printf("Model:       %s\n", info.ModelName)
		fmt.Printf("Serial:      %s\n", info.SerialNumber)
		fmt.Printf("MAC:         %s\n", info.MACAddress)
		fmt.Printf("Firmware:    %s\n", info.FirmwareVersion)
		fmt.Printf("Hardware:    %s\n", info.HardwareVersion)
		fmt.Printf("Internet:    %s\n", info.InternetConnection)
		fmt.Printf("Network:     %s\n", info.NetworkAccess)
		fmt.Printf("Uptime:      %s\n", info.Uptime)
		fmt.Printf("System time: %s\n", info.SystemTime)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
