package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/lowcarbdev/s34/internal/modem"
	"github.com/spf13/cobra"
)

var hnapCmd = &cobra.Command{
	Use:   "hnap <Action> [json-body]",
	Short: "Send a raw HNAP action to the modem",
	Long: `Send an arbitrary authenticated HNAP action and print the raw JSON response.

The Action argument is the bare action name (e.g. GetMotoStatusLog).
An optional second argument supplies the JSON request body; if omitted the
action is sent with an empty string value, which works for most read-only
GetMoto* actions.

Example:
  s34 hnap GetMotoStatusLog
  s34 hnap SetStatusSecuritySettings '{"SetStatusSecuritySettings":{"MotoReboot":"reboot"}}'`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		user, pass, err := credentials()
		if err != nil {
			return err
		}

		action := args[0]

		var body map[string]any
		if len(args) == 2 {
			if err := json.Unmarshal([]byte(args[1]), &body); err != nil {
				return fmt.Errorf("invalid JSON body: %w", err)
			}
		} else {
			body = map[string]any{action: ""}
		}

		client := modem.NewClient(flagURL)
		if err := client.Login(user, pass); err != nil {
			return err
		}

		resp, err := client.Do(action, body)
		if err != nil {
			return err
		}

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resp)
	},
}

func init() {
	rootCmd.AddCommand(hnapCmd)
}
