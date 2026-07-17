// cmd/bot.go
package cmd

import (
	"os"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/spf13/cobra"
)

var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "Manage the WhatsApp bot lifecycle (start, stop, status)",
}

var botStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the WhatsApp bot connection",
	Run: func(cmd *cobra.Command, _ []string) {
		client := newBotClientFromConfig()
		status, err := client.Start(cmd.Context())
		printBotResult(cmd, status, err)
	},
}

var botStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the WhatsApp bot connection",
	Run: func(cmd *cobra.Command, _ []string) {
		purge, _ := cmd.Flags().GetBool("purge")
		client := newBotClientFromConfig()
		status, err := client.Stop(cmd.Context(), purge)
		printBotResult(cmd, status, err)
	},
}

var botStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the current WhatsApp bot connection status",
	Run: func(cmd *cobra.Command, _ []string) {
		client := newBotClientFromConfig()
		status, err := client.Status(cmd.Context())
		printBotResult(cmd, status, err)
	},
}

func newBotClientFromConfig() *external.BotClient {
	config := internal.NewConfig(EnvFile)
	return external.NewBotClient(config.BotBaseURL, config.BotCallbackPSK, config.BotExtractPSK)
}

func printBotResult(cmd *cobra.Command, status *external.BotStatus, err error) {
	if err != nil {
		cmd.PrintErrf("Error: %s\n", err.Error())
		os.Exit(1)
		return
	}
	cmd.Printf("status: %s\n", status.Status)
	if status.QR != "" {
		cmd.Printf("qr: %s\n(Scan this QR code to connect WhatsApp.)\n", status.QR)
	}
}

func init() {
	botStopCmd.Flags().
		Bool("purge", false, "Purge WhatsApp session data on stop (requires QR re-scan next time)")
	botCmd.AddCommand(botStartCmd, botStopCmd, botStatusCmd)
}
