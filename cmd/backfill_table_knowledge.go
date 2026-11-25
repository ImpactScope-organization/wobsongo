package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/spf13/cobra"
)

var (
	backfillTablesApply bool
	backfillTablesLimit int
)

// errTableChunksMismatch is returned when the table chunks already stored
// for a document can't be lined up one-to-one with the tables in its raw
// Docling output, so no chunk is touched rather than risk writing a table
// onto the wrong chunk.
var errTableChunksMismatch = errors.New("stored table chunks do not match raw docling tables")

// backfillTablesCmd fills in the text and Markdown of table chunks stored
// before table support existed. Those chunks were saved with empty text, so
// they were never embedded, translated or mined for knowledge.
var backfillTablesCmd = &cobra.Command{
	Use:   "backfill-tables",
	Short: "Fill in text and Markdown for table chunks ingested before table support",
	Long: "Finds documents that have table chunks with empty text and no Markdown,\n" +
		"re-reads each document's raw Docling output from S3 (Docling itself is\n" +
		"never called again), and writes the table text and Markdown onto the\n" +
		"stored chunks. Then it enqueues embedding, knowledge extraction and\n" +
		"translation for the document; those jobs only pick up chunks that still\n" +
		"need work and run inside `wob serve`.\n\n" +
		"Chunks are matched to Docling tables by order and page. A document whose\n" +
		"table count or pages don't line up (or whose raw output is missing in S3)\n" +
		"is skipped and reported, and nothing is written for it. Only chunks that\n" +
		"are still empty are updated, so re-running is safe.\n\n" +
		"Without --apply, this is a dry run: it reads the raw output and reports\n" +
		"what would change, but writes and enqueues nothing.",
	Run: runBackfillTables,
}

func init() {
	backfillTablesCmd.Flags().
		BoolVar(&backfillTablesApply, "apply", false, "Actually update chunks and enqueue jobs (default is a dry run)")
	backfillTablesCmd.Flags().
		IntVar(&backfillTablesLimit, "limit", 1000, "Max documents per run")
}

type registrationOnlyEmbedChunksWorker struct {
	river.WorkerDefaults[queue.EmbedChunksDTO]
}

func (*registrationOnlyEmbedChunksWorker) Work(
	context.Context,
	*river.Job[queue.EmbedChunksDTO],
) error {
	return nil
}

type registrationOnlyExtractKnowledgeWorker struct {
	river.WorkerDefaults[queue.ExtractKnowledgeDTO]
}

func (*registrationOnlyExtractKnowledgeWorker) Work(
	context.Context,
	*river.Job[queue.ExtractKnowledgeDTO],
) error {
	return nil
}

// tableChunkUpdate is the text and Markdown to write onto one stored chunk.
type tableChunkUpdate struct {
	ChunkID       uuid.UUID
	Text          string
	TableMarkdown string
}

type tableBackfillPlan struct {
	DocumentID uuid.UUID
	Updates    []tableChunkUpdate
}

func runBackfillTables(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	config := internal.NewConfig(EnvFile)

	if backfillTablesLimit <= 0 {
		cmd.PrintErrln("--limit must be > 0")
		os.Exit(1)
		return
	}

	pool, err := pgxpool.New(ctx, config.PostgresURI)
	if err != nil {
		cmd.PrintErrf("Failed to connect to database: %s\n", err.Error())
		os.Exit(1)
		return
	}
	defer pool.Close()

	rawStore, err := repo.NewS3Provider(ctx, config.S3Config)
	if err != nil {
		cmd.PrintErrf("Failed to initialize S3 provider: %s\n", err.Error())
		os.Exit(1)
		return
	}

	queries := db.New(pool)

	documentIDs, err := queries.ListDocumentIDsNeedingTableBackfill(ctx)
	if err != nil {
		cmd.PrintErrf("Failed to list documents needing table backfill: %s\n", err.Error())
		os.Exit(1)
		return
	}

	if len(documentIDs) == 0 {
		cmd.Println("No documents have table chunks needing backfill.")
		return
	}
	if len(documentIDs) > backfillTablesLimit {
		documentIDs = documentIDs[:backfillTablesLimit]
	}

	plans := make([]tableBackfillPlan, 0, len(documentIDs))
	skipped := 0
	for _, id := range documentIDs {
		plan, err := planTableBackfill(ctx, queries, rawStore, id)
		if err != nil {
			cmd.PrintErrf("Skipping document %s: %s\n", id, err.Error())
			skipped++
			continue
		}
		if len(plan.Updates) == 0 {
			continue
		}
		plans = append(plans, *plan)
	}

	chunkCount := 0
	for _, plan := range plans {
		chunkCount += len(plan.Updates)
	}

	if len(plans) == 0 {
		cmd.Printf("Nothing to backfill (%d document(s) skipped).\n", skipped)
		return
	}

	if !backfillTablesApply {
		cmd.Printf(
			"Dry run: would update %d table chunk(s) in %d document(s), %d skipped.\n",
			chunkCount, len(plans), skipped,
		)
		for _, plan := range plans {
			cmd.Printf("  %s: %d table chunk(s)\n", plan.DocumentID, len(plan.Updates))
		}
		cmd.Println("Re-run with --apply to write the changes and enqueue the jobs.")
		return
	}

	workers := river.NewWorkers()
	river.AddWorker(workers, &registrationOnlyEmbedChunksWorker{})
	river.AddWorker(workers, &registrationOnlyExtractKnowledgeWorker{})
	river.AddWorker(workers, &registrationOnlyTranslateChunksWorker{})
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}},
		Workers: workers,
	})
	if err != nil {
		cmd.PrintErrf("Failed to initialize job queue client: %s\n", err.Error())
		os.Exit(1)
		return
	}

	applied := 0
	for _, plan := range plans {
		if err := applyTableBackfill(ctx, pool, riverClient, plan); err != nil {
			cmd.PrintErrf(
				"Failed to backfill document %s, it was left unchanged: %s\n",
				plan.DocumentID,
				err.Error(),
			)
			skipped++
			continue
		}
		applied++
	}

	cmd.Printf(
		"Updated table chunks in %d document(s), %d skipped. `wob serve` will embed, translate and extract knowledge from them.\n",
		applied,
		skipped,
	)
}

// planTableBackfill reads a document's raw Docling output from S3 and works
// out which of its stored table chunks should be updated.
func planTableBackfill(
	ctx context.Context,
	queries *db.Queries,
	rawStore data.RawObjectStore,
	documentID uuid.UUID,
) (*tableBackfillPlan, error) {
	rc, err := rawStore.GetObject(ctx, tableBackfillRawOutputKey(documentID))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch raw docling output: %w", err)
	}
	raw, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read raw docling output: %w", err)
	}

	parsed, err := external.ParseRaw(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse raw docling output: %w", err)
	}

	stored, err := queries.ListDocumentChunksByDocumentID(ctx, documentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list stored chunks: %w", err)
	}

	updates, err := matchTableChunks(stored, parsed.Chunks)
	if err != nil {
		return nil, err
	}

	return &tableBackfillPlan{DocumentID: documentID, Updates: updates}, nil
}

// matchTableChunks pairs the stored table chunks (in reading order) with the
// table chunks of the raw Docling output, which come out in the same order
// because Docling lists every table once, in order. It refuses to pair
// anything unless the counts and pages agree, and only returns chunks that
// are still empty and have a non-empty table to write.
func matchTableChunks(
	stored []db.DocumentChunk,
	parsed []model.ParsedChunk,
) ([]tableChunkUpdate, error) {
	storedTables := make([]db.DocumentChunk, 0, len(stored))
	for i := range stored {
		if stored[i].LayoutType == string(model.LayoutTypeTable) {
			storedTables = append(storedTables, stored[i])
		}
	}

	parsedTables := make([]model.ParsedChunk, 0, len(parsed))
	for i := range parsed {
		if parsed[i].LayoutType == model.LayoutTypeTable {
			parsedTables = append(parsedTables, parsed[i])
		}
	}

	if len(storedTables) != len(parsedTables) {
		return nil, fmt.Errorf(
			"%w: %d stored vs %d parsed",
			errTableChunksMismatch, len(storedTables), len(parsedTables),
		)
	}

	updates := make([]tableChunkUpdate, 0, len(storedTables))
	for i := range storedTables {
		if int(storedTables[i].Page) != parsedTables[i].Page {
			return nil, fmt.Errorf(
				"%w: table %d is on page %d vs %d",
				errTableChunksMismatch, i, storedTables[i].Page, parsedTables[i].Page,
			)
		}

		alreadyFilled := storedTables[i].Text != "" || storedTables[i].TableMarkdown != ""
		emptyTable := parsedTables[i].Text == ""
		if alreadyFilled || emptyTable {
			continue
		}

		updates = append(updates, tableChunkUpdate{
			ChunkID:       storedTables[i].ID,
			Text:          parsedTables[i].Text,
			TableMarkdown: parsedTables[i].TableMarkdown,
		})
	}

	return updates, nil
}

// applyTableBackfill writes one document's table chunks and enqueues its
// follow-up jobs in a single Postgres transaction, so a document is either
// fully updated and queued, or left untouched.
func applyTableBackfill(
	ctx context.Context,
	pool *pgxpool.Pool,
	riverClient *river.Client[pgx.Tx],
	plan tableBackfillPlan,
) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Rollback is a no-op once Commit has succeeded.
	defer func() { _ = tx.Rollback(ctx) }()

	queries := db.New(pool).WithTx(tx)
	now := time.Now()

	for _, u := range plan.Updates {
		if err := queries.UpdateChunkTable(ctx, db.UpdateChunkTableParams{
			ID:            u.ChunkID,
			Text:          u.Text,
			TableMarkdown: u.TableMarkdown,
			UpdatedAt:     now,
		}); err != nil {
			return fmt.Errorf("failed to update chunk %s: %w", u.ChunkID, err)
		}
	}

	if _, err := riverClient.InsertTx(ctx, tx, queue.EmbedChunksDTO{
		DocumentID: plan.DocumentID,
	}, nil); err != nil {
		return fmt.Errorf("failed to enqueue chunk embedding: %w", err)
	}
	if _, err := riverClient.InsertTx(ctx, tx, queue.ExtractKnowledgeDTO{
		DocumentID: plan.DocumentID,
	}, nil); err != nil {
		return fmt.Errorf("failed to enqueue knowledge extraction: %w", err)
	}
	if _, err := riverClient.InsertTx(ctx, tx, queue.TranslateChunksDTO{
		DocumentID: plan.DocumentID,
	}, nil); err != nil {
		return fmt.Errorf("failed to enqueue chunk translation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// tableBackfillRawOutputKey mirrors worker.rawOutputKey, the S3 key
// ParseDocumentWorker stores a document's raw Docling response under.
func tableBackfillRawOutputKey(documentID uuid.UUID) string {
	return fmt.Sprintf("parsed_output/%s.json", documentID)
}
