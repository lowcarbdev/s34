package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var channelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "Show downstream and upstream channel bonding summary",
	RunE: func(cmd *cobra.Command, args []string) error {
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

		fmt.Printf("Downstream (%d channels)\n", len(ds))
		fmt.Println(strings.Repeat("-", 90))
		printDownstream(ds)

		fmt.Println()

		fmt.Printf("Upstream (%d channels)\n", len(us))
		fmt.Println(strings.Repeat("-", 72))
		printUpstream(us)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(channelsCmd)
}
