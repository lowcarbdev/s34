package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var watchInterval time.Duration

var watchCmd = &cobra.Command{
	Use:   "watch <command>",
	Short: "Repeatedly run a command, refreshing the output",
	Long: `Repeatedly run a subcommand at a fixed interval, clearing the screen between runs.

Supported commands: channels, status connection, status startup, status software,
                    status downstream, status upstream, log

Examples:
  s34 watch channels
  s34 watch --interval 10s status connection`,
	Args:      cobra.MinimumNArgs(1),
	ValidArgs: []string{"channels", "status", "log"},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Pre-flight: validate the subcommand before starting the loop.
		if err := runWatch(args); err != nil {
			return err
		}

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		ticker := time.NewTicker(watchInterval)
		defer ticker.Stop()

		for {
			select {
			case <-sig:
				return nil
			case <-ticker.C:
				clearScreen()
				if err := runWatch(args); err != nil {
					// Print the error but keep running — transient network
					// hiccups shouldn't kill the watch loop.
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
				}
			}
		}
	},
}

// runWatch dispatches to the appropriate command handler based on args.
func runWatch(args []string) error {
	stamp := time.Now().Format("2006-01-02 15:04:05")

	switch args[0] {
	case "channels":
		client, err := newAuthClient()
		if err != nil {
			return err
		}
		ds, err := client.StatusDownstream()
		if err != nil {
			return fmt.Errorf("downstream: %w", err)
		}
		us, err := client.StatusUpstream()
		if err != nil {
			return fmt.Errorf("upstream: %w", err)
		}
		fmt.Printf("channels  —  %s  (every %s, Ctrl-C to stop)\n\n", stamp, watchInterval)
		fmt.Printf("Downstream (%d channels)\n", len(ds))
		fmt.Println(strings.Repeat("-", 90))
		printDownstream(ds)
		fmt.Println()
		fmt.Printf("Upstream (%d channels)\n", len(us))
		fmt.Println(strings.Repeat("-", 72))
		printUpstream(us)

	case "status":
		if len(args) < 2 {
			return fmt.Errorf("watch status requires a subcommand: connection, startup, software, downstream, upstream")
		}
		fmt.Printf("status %s  —  %s  (every %s, Ctrl-C to stop)\n\n", args[1], stamp, watchInterval)
		switch args[1] {
		case "downstream":
			client, err := newAuthClient()
			if err != nil {
				return err
			}
			ds, err := client.StatusDownstream()
			if err != nil {
				return err
			}
			printDownstream(ds)
		case "upstream":
			client, err := newAuthClient()
			if err != nil {
				return err
			}
			us, err := client.StatusUpstream()
			if err != nil {
				return err
			}
			printUpstream(us)
		case "connection", "startup", "software":
			return runStatus(args[1])
		default:
			return fmt.Errorf("unknown status subcommand %q", args[1])
		}

	case "log":
		fmt.Printf("log  —  %s  (every %s, Ctrl-C to stop)\n\n", stamp, watchInterval)
		client, err := newAuthClient()
		if err != nil {
			return err
		}
		data, err := client.EventLog()
		if err != nil {
			return err
		}
		resp, _ := data["GetCustomerStatusLogResponse"].(map[string]any)
		blob, _ := resp["CustomerStatusLogList"].(string)
		for _, entry := range strings.Split(blob, "}-{") {
			fields := strings.SplitN(entry, "^", 5)
			if len(fields) < 5 {
				continue
			}
			fmt.Printf("%s  %-10s  %s\n",
				strings.TrimSpace(fields[1]),
				strings.TrimSpace(fields[3]),
				strings.TrimSpace(fields[4]))
		}

	default:
		return fmt.Errorf("watch does not support %q; try: channels, status <sub>, log", args[0])
	}
	return nil
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func init() {
	watchCmd.Flags().DurationVarP(&watchInterval, "interval", "i", 30*time.Second, "Refresh interval (e.g. 10s, 1m)")
	rootCmd.AddCommand(watchCmd)
}
