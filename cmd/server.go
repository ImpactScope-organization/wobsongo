package cmd

import (
	"os"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/impactscope-organization/wobsongo/internal/worker"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
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

		// Initialize database connection pool.
		pool, err := pgxpool.New(cmd.Context(), config.PostgresURI)
		if err != nil {
			cmd.PrintErrf("Failed to connect to database: %s\n", err.Error())
			os.Exit(1)
			return
		}
		defer pool.Close()

		apifyDispatcher := external.NewDispatcher(
			config.ApifyToken,
			config.ApifyTikTokActorID,
			config.ApifyIGActorID,
		)

		// The media provider is constructed here (not inside buildApp) so it
		// can be shared with River workers, which must be registered before
		// the river.Client (and therefore before buildApp) exists.
		if err := internal.IsS3OK(config.S3Config); err != nil {
			cmd.PrintErrf("Config error: %s\n", err.Error())
			os.Exit(1)
			return
		}
		mediaProvider, err := repo.NewS3Provider(cmd.Context(), config.S3Config)
		if err != nil {
			cmd.PrintErrf("Failed to initialize S3 media provider: %s\n", err.Error())
			os.Exit(1)
			return
		}
		mediaService := service.NewMediaService(mediaProvider)
		doclingClient := external.NewDoclingClient(config.DoclingBaseURL)

		// register workers with River
		workers := river.NewWorkers()

		// register ExtractMediaWorker with River
		mediaWorker := worker.NewExtractMediaWorker(apifyDispatcher)
		river.AddWorker(workers, mediaWorker)

		// register ParseDocumentWorker with River
		parseDocumentWorker := worker.NewParseDocumentWorker(doclingClient, mediaService)
		river.AddWorker(workers, parseDocumentWorker)

		// Initialize River client with the database pool and registered workers.
		riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
			Queues: map[string]river.QueueConfig{
				river.QueueDefault: {MaxWorkers: 10},
			},
			Workers: workers,
		})
		if err != nil {
			cmd.PrintErrf("Failed to initialize job queue: %s\n", err.Error())
			os.Exit(1)
			return
		}

		if err := riverClient.Start(cmd.Context()); err != nil {
			cmd.PrintErrf("Failed to start job queue: %s\n", err.Error())
			os.Exit(1)
			return
		}
		defer func() {
			if err := riverClient.Stop(cmd.Context()); err != nil {
				cmd.PrintErrf("Failed to stop River client: %v", err)
				os.Exit(1)
				return
			}
		}()

		// Build and start HTTP API server.
		app := buildApp(config, pool, riverClient, mediaProvider)

		cmd.Printf("Starting the server at %s\n", config.APIHost)
		if err := app.Start(); err != nil {
			cmd.PrintErrf("cannot start the server: %s", err.Error())
			os.Exit(1)
			return
		}
	},
}
