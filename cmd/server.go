package cmd

import (
	"os"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/impactscope-organization/wobsongo/internal/worker"
	"github.com/jackc/pgx/v5"
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

		if err := internal.IsVLMOK(config.VLMConfig); err != nil {
			cmd.PrintErrf("Config error: %s\n", err.Error())
			os.Exit(1)
			return
		}
		vlmClient := external.NewVLMClient(
			config.VLMConfig.BaseURL,
			config.VLMConfig.Model,
			config.VLMConfig.APIKey,
		)

		// riverClient is assigned below, after workers (which need to be
		// registered via river.AddWorker before river.NewClient produces the
		// client) are constructed. ChunkRepo/RiverJobEnqueuer only resolve it
		// lazily, at Enqueue-call time — always well after riverClient.Start()
		// — so this ordering is safe. See their constructors' doc comments.
		var riverClient *river.Client[pgx.Tx]
		riverClientFn := func() *river.Client[pgx.Tx] { return riverClient }

		chunkRepo := repo.NewDocumentChunkRepo(db.New(pool), pool, riverClientFn)
		jobEnqueuer := repo.NewRiverJobEnqueuer(pool, riverClientFn)

		// Same reasoning as chunkRepo's nil case used to be: this document
		// repo instance is only used by the worker to backfill PageCount/Title
		// after parsing (GetByID+Update) — it never calls Enqueue. The
		// HTTP-facing document repo (with a real riverClient) is built
		// separately, inside buildApp.
		workerDocumentRepo := repo.NewDocumentRepo(db.New(pool), pool, nil)
		documentService := service.NewDocumentService(workerDocumentRepo)

		// register workers with River
		workers := river.NewWorkers()

		// register ExtractMediaWorker with River
		mediaWorker := worker.NewExtractMediaWorker(apifyDispatcher)
		river.AddWorker(workers, mediaWorker)

		// register ParseDocumentWorker with River
		parseDocumentWorker := worker.NewParseDocumentWorker(
			doclingClient,
			mediaService,
			mediaProvider,
			jobEnqueuer,
		)
		river.AddWorker(workers, parseDocumentWorker)

		// register ProcessParsedDocumentWorker with River
		processParsedDocumentWorker := worker.NewProcessParsedDocumentWorker(
			mediaProvider,
			documentService,
			chunkRepo,
		)
		river.AddWorker(workers, processParsedDocumentWorker)

		// register CaptionImageChunksWorker with River
		captionImageChunksWorker := worker.NewCaptionImageChunksWorker(
			mediaProvider,
			chunkRepo,
			documentService,
			vlmClient,
		)
		river.AddWorker(workers, captionImageChunksWorker)

		// Initialize River client with the database pool and registered workers.
		riverClient, err = river.NewClient(riverpgxv5.New(pool), &river.Config{
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
