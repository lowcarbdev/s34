package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	flagURL      string
	flagUsername string
	flagPassword string
	flagJSON     bool
)

var rootCmd = &cobra.Command{
	Use:   "s34",
	Short: "CLI for the ARRIS SURFboard S34 cable modem",
	Long: `s34 interacts with the ARRIS SURFboard S34 cable modem's HNAP JSON API.

Credentials can be supplied via flags or environment variables:
  S34_USERNAME  (default: admin)
  S34_PASSWORD

The modem uses a self-signed TLS certificate; verification is intentionally skipped.`,
}

// Execute is the entry point called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagURL, "url", "https://192.168.100.1", "Modem base URL")
	rootCmd.PersistentFlags().StringVarP(&flagUsername, "username", "u", "", "Admin username (env: S34_USERNAME, default: admin)")
	rootCmd.PersistentFlags().StringVarP(&flagPassword, "password", "p", "", "Admin password (env: S34_PASSWORD)")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output raw JSON (where applicable)")
}

// credentials resolves username and password from flags then environment.
func credentials() (string, string, error) {
	user := flagUsername
	if user == "" {
		user = os.Getenv("S34_USERNAME")
	}
	if user == "" {
		user = "admin"
	}

	pass := flagPassword
	if pass == "" {
		pass = os.Getenv("S34_PASSWORD")
	}
	if pass == "" {
		return "", "", fmt.Errorf("password required: use --password / -p or set S34_PASSWORD")
	}
	return user, pass, nil
}

// printResult prints a response map as pretty JSON or as indented key=value pairs.
func printResult(data map[string]any) error {
	if flagJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}
	printMap(data, 0)
	return nil
}

func printMap(v any, depth int) {
	indent := strings.Repeat("  ", depth)
	switch val := v.(type) {
	case map[string]any:
		for k, v2 := range val {
			switch inner := v2.(type) {
			case map[string]any:
				fmt.Printf("%s%s:\n", indent, k)
				printMap(inner, depth+1)
			case []any:
				fmt.Printf("%s%s:\n", indent, k)
				for i, item := range inner {
					fmt.Printf("%s  [%d]\n", indent, i)
					printMap(item, depth+2)
				}
			default:
				fmt.Printf("%s%s: %v\n", indent, k, v2)
			}
		}
	default:
		fmt.Printf("%s%v\n", indent, v)
	}
}
