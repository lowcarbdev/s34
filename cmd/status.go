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

var statusDownstreamCmd = &cobra.Command{
	Use:   "downstream",
	Short: "Downstream channel bonding info",
	RunE:  func(cmd *cobra.Command, args []string) error { return runStatus("downstream") },
}

var statusUpstreamCmd = &cobra.Command{
	Use:   "upstream",
	Short: "Upstream channel bonding info",
	RunE:  func(cmd *cobra.Command, args []string) error { return runStatus("upstream") },
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

func runStatus(kind string) error {
	user, pass, err := credentials()
	if err != nil {
		return err
	}

	client := modem.NewClient(flagURL)
	if err := client.Login(user, pass); err != nil {
		return err
	}

	var data map[string]any
	switch kind {
	case "connection":
		data, err = client.StatusConnection()
	case "downstream":
		data, err = client.StatusDownstream()
	case "upstream":
		data, err = client.StatusUpstream()
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
		user, pass, err := credentials()
		if err != nil {
			return err
		}

		client := modem.NewClient(flagURL)
		if err := client.Login(user, pass); err != nil {
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

		// Each entry is separated by "}-{"; fields within an entry by "^".
		// Field order: index ^ datetime ^ (empty) ^ severity ^ description
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

func init() {
	statusCmd.AddCommand(statusConnectionCmd, statusDownstreamCmd, statusUpstreamCmd,
		statusStartupCmd, statusSoftwareCmd)
	rootCmd.AddCommand(statusCmd, logCmd)
}
