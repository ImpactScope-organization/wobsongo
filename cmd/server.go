package cmd

import (
	"os"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the HTTP API server",
	Run: func(cmd *cobra.Command, _ []string) {
		config := internal.NewConfig(EnvFile)
		if err := config.IsOK(); err != nil {
			cmd.PrintErrf("Config error: %s\n", err.Error())
			os.Exit(1)
			return
		}

		// Build and start HTTP API server.
		app := buildApp(
			config,
		)

		cmd.Printf("Starting the Wobsongo server at %s\n", config.APIHost)
		if err := app.Start(); err != nil {
			cmd.PrintErrf("Cannot start the server: %s", err.Error())
			os.Exit(1)
			return
		}
	},
}
