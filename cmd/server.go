package cmd

import (
	"os"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/impactscope-organization/wobsongo/internal/worker"
	"github.com/jackc/pgx/v5"
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

		// Initialize database connection pool. Uses repo.NewPgxPool (not
		// pgxpool.New) so pgvector types are registered on every connection —
		// required for document_chunks.embedding to (de)serialize correctly.
		pool, err := repo.NewPgxPool(cmd.Context(), config.PostgresURI)
		if err != nil {
			cmd.PrintErrf("Failed to connect to database: %s\n", err.Error())
			os.Exit(1)
			return
		}
		defer pool.Close()

		apifyDispatcher := external.NewDispatcher(
			config.ApifyConfig.Token,
			config.ApifyConfig.TikTokActorID,
			config.ApifyConfig.IGActorID,
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

		if err := internal.IsEmbeddingOK(config.EmbeddingConfig); err != nil {
			cmd.PrintErrf("Config error: %s\n", err.Error())
			os.Exit(1)
			return
		}
		embeddingClient := newEmbeddingClient(config.EmbeddingConfig)

		if err := internal.IsExtractionOK(config.ExtractionConfig); err != nil {
			cmd.PrintErrf("Config error: %s\n", err.Error())
			os.Exit(1)
			return
		}
		extractionClient := external.NewExtractionClient(
			config.ExtractionConfig.BaseURL,
			config.ExtractionConfig.Model,
			config.ExtractionConfig.APIKey,
		)

		if err := internal.IsClaimCheckOK(config.ClaimCheckConfig); err != nil {
			cmd.PrintErrf("Config error: %s\n", err.Error())
			os.Exit(1)
			return
		}
		claimAnalyzerClient := external.NewClaimAnalyzerClient(
			config.ClaimCheckConfig.BaseURL,
			config.ClaimCheckConfig.Model,
			config.ClaimCheckConfig.APIKey,
		)
		judgeClient := external.NewJudgeClient(
			config.ClaimCheckConfig.BaseURL,
			config.ClaimCheckConfig.Model,
			config.ClaimCheckConfig.APIKey,
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

		atomicKnowledgeRepo := repo.NewAtomicKnowledgeRepo(db.New(pool), pool)

		workerVideoRepo := repo.NewVideoRepo(db.New(pool), pool, nil)
		workerVideoService := service.NewVideoService(workerVideoRepo)
		// register workers with River
		workers := river.NewWorkers()

		// register ExtractMediaWorker with River
		mediaWorker := worker.NewExtractMediaWorker(apifyDispatcher)
		river.AddWorker(workers, mediaWorker)
		botClient := external.NewBotClient(config.BotBaseURL, config.BotCallbackPSK)

		transcriptionWorker := worker.NewTranscriptionWorker(
			workerVideoService,
			config.ASRConfig.Endpoint,
			config.ASRConfig.Model,
			config.ASRConfig.SourceLang,
			botClient,
		)
		river.AddWorker(workers, transcriptionWorker)

		transcriptionWorker := worker.NewTranscriptionWorker(
			workerVideoService,
			config.ASRConfig.Endpoint,
		)
		river.AddWorker(workers, transcriptionWorker)

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

		// register EmbedChunksWorker with River
		embedChunksWorker := worker.NewEmbedChunksWorker(chunkRepo, embeddingClient)
		river.AddWorker(workers, embedChunksWorker)

		// register ExtractKnowledgeWorker with River
		extractKnowledgeWorker := worker.NewExtractKnowledgeWorker(
			chunkRepo,
			atomicKnowledgeRepo,
			documentService,
			extractionClient,
			config.ExtractionConfig.Concurrency,
		)
		river.AddWorker(workers, extractKnowledgeWorker)

		// register EmbedKnowledgeWorker with River
		embedKnowledgeWorker := worker.NewEmbedKnowledgeWorker(atomicKnowledgeRepo, embeddingClient)
		river.AddWorker(workers, embedKnowledgeWorker)

		// Initialize River client with the database pool and registered workers.
		// Document ingestion and media processing get separate queues (see
		// each job DTO's InsertOpts() in internal/queue) so a long-running
		// document import — extraction alone can run for hours on a large
		// document, see internal/worker/extract_knowledge.go — can't starve
		// video processing of worker slots, or vice versa.
		riverClient, err = river.NewClient(riverpgxv5.New(pool), &river.Config{
			Queues: map[string]river.QueueConfig{
				queue.QueueDocumentIngestion: {MaxWorkers: 10},
				queue.QueueMediaProcessing:   {MaxWorkers: 10},
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
		app := buildApp(config, pool, riverClient, mediaProvider, buildAppClaimCheckDeps{
			chunkRepo:     chunkRepo,
			knowledgeRepo: atomicKnowledgeRepo,
			embedder:      embeddingClient,
			claimAnalyzer: claimAnalyzerClient,
			claimJudge:    judgeClient,
		})

		cmd.Printf("Starting the server at %s\n", config.APIHost)
		if err := app.Start(); err != nil {
			cmd.PrintErrf("cannot start the server: %s", err.Error())
			os.Exit(1)
			return
		}
	},
}

// newEmbeddingClient constructs the data.Embedder implementation matching
// cfg.Provider — shared between cmd/server.go and cmd/healthcheck.go so both
// exercise the exact same wire shape. Assumes internal.IsEmbeddingOK(cfg)
// has already been checked (Provider is guaranteed recognized by that gate).
func newEmbeddingClient(cfg *internal.EmbeddingConfig) data.Embedder {
	if cfg.Provider == internal.EmbeddingProviderModalBGE {
		return external.NewModalBGEClient(cfg.BaseURL, cfg.APIKey)
	}
	return external.NewEmbeddingClient(cfg.BaseURL, cfg.Model, cfg.APIKey)
}
