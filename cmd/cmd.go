package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// EnvFile is the path to the environment variable file, passed via CLI flag.
// By default, it is empty, meaning no env file is used.
var EnvFile string

var rootCmd = &cobra.Command{
	Use:   "wob",
	Short: "Wobsongo API CLI",
}

func Execute() {
	rootCmd.PersistentFlags().StringVarP(
		&EnvFile,
		"env",
		"e",
		"",
		"Environment variable file",
	)

	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(migrateUpCmd)
	rootCmd.AddCommand(migrateDownCmd)
	rootCmd.AddCommand(resetCmd)
	rootCmd.AddCommand(documentCmd)
	rootCmd.AddCommand(doclingCmd)
	rootCmd.AddCommand(healthcheckCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
