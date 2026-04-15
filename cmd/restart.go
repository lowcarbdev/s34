package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

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

		deadline := time.Now().Add(5 * time.Minute)
		for {
			client := modem.NewClient(flagURL)

			fmt.Fprintf(os.Stderr, "Logging in as %s...\n", user)
			if err := client.Login(user, pass); err != nil {
				if errors.Is(err, modem.ErrReload) && time.Now().Before(deadline) {
					fmt.Fprintln(os.Stderr, "Modem is starting up, retrying in 10s...")
					time.Sleep(10 * time.Second)
					continue
				}
				return err
			}

			fmt.Fprintln(os.Stderr, "Sending restart command...")
			if err := client.Restart(); err != nil {
				return err
			}

			fmt.Println("Modem is restarting. It will be unreachable for approximately 2 minutes.")
			return nil
		}
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
