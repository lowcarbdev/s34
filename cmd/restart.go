package cmd

import (
	"fmt"
	"os"

	"github.com/lowcarbdev/s34/internal/modem"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart (reboot) the modem",
	Long:  `Authenticates with the modem and sends a reboot command. The modem will be unreachable for ~2 minutes while it restarts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		user, pass, err := credentials()
		if err != nil {
			return err
		}

		client := modem.NewClient(flagURL)

		fmt.Fprintf(os.Stderr, "Logging in as %s...\n", user)
		if err := client.Login(user, pass); err != nil {
			return err
		}

		fmt.Fprintln(os.Stderr, "Sending restart command...")
		if err := client.Restart(); err != nil {
			return err
		}

		fmt.Println("Modem is restarting. It will be unreachable for approximately 2 minutes.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
