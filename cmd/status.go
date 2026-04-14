package cmd

import (
	"fmt"
	"strings"

	"github.com/lowcarbdev/s34/internal/modem"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show modem status information",
	Long:  `Displays modem status. Sub-commands provide focused views; running 'status' alone shows connection summary.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStatus("connection")
	},
}

var statusConnectionCmd = &cobra.Command{
	Use:   "connection",
	Short: "Connection info (lock status, IP, etc.)",
	RunE:  func(cmd *cobra.Command, args []string) error { return runStatus("connection") },
}

var statusStartupCmd = &cobra.Command{
	Use:   "startup",
	Short: "Startup sequence / boot status",
	RunE:  func(cmd *cobra.Command, args []string) error { return runStatus("startup") },
}

var statusSoftwareCmd = &cobra.Command{
	Use:   "software",
	Short: "Firmware and software versions",
	RunE:  func(cmd *cobra.Command, args []string) error { return runStatus("software") },
}

var statusDownstreamCmd = &cobra.Command{
	Use:   "downstream",
	Short: "Downstream channel bonding info",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newAuthClient()
		if err != nil {
			return err
		}
		channels, err := client.StatusDownstream()
		if err != nil {
			return err
		}
		printDownstream(channels)
		return nil
	},
}

var statusUpstreamCmd = &cobra.Command{
	Use:   "upstream",
	Short: "Upstream channel bonding info",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newAuthClient()
		if err != nil {
			return err
		}
		channels, err := client.StatusUpstream()
		if err != nil {
			return err
		}
		printUpstream(channels)
		return nil
	},
}

func runStatus(kind string) error {
	client, err := newAuthClient()
	if err != nil {
		return err
	}

	var data map[string]any
	switch kind {
	case "connection":
		data, err = client.StatusConnection()
	case "startup":
		data, err = client.StatusStartup()
	case "software":
		data, err = client.StatusSoftware()
	default:
		return fmt.Errorf("unknown status kind: %s", kind)
	}
	if err != nil {
		return err
	}
	return printResult(data)
}

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show the modem event log",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newAuthClient()
		if err != nil {
			return err
		}

		data, err := client.EventLog()
		if err != nil {
			return err
		}

		if flagJSON {
			return printResult(data)
		}

		resp, _ := data["GetCustomerStatusLogResponse"].(map[string]any)
		blob, _ := resp["CustomerStatusLogList"].(string)
		if blob == "" {
			fmt.Println("(no log entries)")
			return nil
		}

		// Each entry: index^datetime^^severity^description, separated by "}-{"
		for _, entry := range strings.Split(blob, "}-{") {
			fields := strings.SplitN(entry, "^", 5)
			if len(fields) < 5 {
				continue
			}
			datetime := strings.TrimSpace(fields[1])
			severity := strings.TrimSpace(fields[3])
			description := strings.TrimSpace(fields[4])
			fmt.Printf("%s  %-10s  %s\n", datetime, severity, description)
		}
		return nil
	},
}

// newAuthClient creates a client and logs in using the global flags/env.
func newAuthClient() (*modem.Client, error) {
	user, pass, err := credentials()
	if err != nil {
		return nil, err
	}
	client := modem.NewClient(flagURL)
	if err := client.Login(user, pass); err != nil {
		return nil, err
	}
	return client, nil
}

func printDownstream(channels []modem.DownstreamChannel) {
	fmt.Printf("%-4s  %-8s  %-10s  %-4s  %-12s  %-8s  %-8s  %-10s  %-12s\n",
		"Ch", "Status", "Modulation", "ID", "Freq (Hz)", "Pwr (dBmV)", "SNR (dB)", "Corrected", "Uncorrected")
	fmt.Println(strings.Repeat("-", 90))
	for _, ch := range channels {
		fmt.Printf("%-4s  %-8s  %-10s  %-4s  %-12s  %-10s  %-8s  %-10s  %-12s\n",
			ch.Channel, ch.Status, ch.Modulation, ch.ID,
			ch.FreqHz, ch.PowerDBmV, ch.SNRDB, ch.Corrected, ch.Uncorrected)
	}
}

func printUpstream(channels []modem.UpstreamChannel) {
	fmt.Printf("%-4s  %-8s  %-8s  %-4s  %-14s  %-12s  %-10s\n",
		"Ch", "Status", "Type", "ID", "Sym Rate (Ksym)", "Freq (Hz)", "Pwr (dBmV)")
	fmt.Println(strings.Repeat("-", 72))
	for _, ch := range channels {
		fmt.Printf("%-4s  %-8s  %-8s  %-4s  %-14s  %-12s  %-10s\n",
			ch.Channel, ch.Status, ch.Type, ch.ID,
			ch.SymRateKsym, ch.FreqHz, ch.PowerDBmV)
	}
}

func init() {
	statusCmd.AddCommand(statusConnectionCmd, statusDownstreamCmd, statusUpstreamCmd,
		statusStartupCmd, statusSoftwareCmd)
	rootCmd.AddCommand(statusCmd, logCmd)
}
