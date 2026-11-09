package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"github.com/spf13/cobra"
)

var (
	backfillClaimsApply bool
	backfillClaimsLimit int32
)

// errNoActiveTopics is returned when no topic is configured..
var errNoActiveTopics = errors.New(
	"no active topics configured, add at least one topic then re-run",
)

var backfillClaimsCmd = &cobra.Command{
	Use:   "backfill-claims",
	Short: "Enqueue claim checks for old user messages that never became claims",
	Long: "Finds user messages without a claim, of any age, and enqueues one ClaimCheckJob\n" +
		"each, on the claim_backfill queue. The jobs are processed by `wob serve`, which\n" +
		"then also clusters the new claims into narratives. Nothing is deleted or\n" +
		"overwritten and no reply is sent to any user.\n" +
		"Requires at least one active topic: without topics no claim can be stored.\n" +
		"Without --apply this is a dry run.",
	Run: runBackfillClaims,
}

func init() {
	backfillClaimsCmd.Flags().
		BoolVar(&backfillClaimsApply, "apply", false, "Actually enqueue (default: dry run)")
	backfillClaimsCmd.Flags().
		Int32Var(&backfillClaimsLimit, "limit", 1000, "Max messages per run")
}

// registrationOnlyClaimCheckWorker lets this CLI process Insert the job.
// Work is never called because the client is never started.
type registrationOnlyClaimCheckWorker struct {
	river.WorkerDefaults[queue.ClaimCheckJob]
}

func (*registrationOnlyClaimCheckWorker) Work(
	context.Context,
	*river.Job[queue.ClaimCheckJob],
) error {
	return nil
}

// claimBackfillCandidates is everything a backfill run would enqueue.
type claimBackfillCandidates struct {
	activeTopics int64
	messages     []db.ListUserMessagesWithoutClaimsRow
	transcripts  []db.ListVideoTranscriptsWithoutClaimsRow
}

func runBackfillClaims(cmd *cobra.Command, _ []string) {
	ctx := cmd.Context()
	config := internal.NewConfig(EnvFile)

	if backfillClaimsLimit <= 0 {
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

	candidates, err := loadClaimBackfillCandidates(ctx, db.New(pool), backfillClaimsLimit)
	if err != nil {
		cmd.PrintErrf("%s\n", err.Error())
		os.Exit(1)
		return
	}

	if len(candidates.messages) == 0 && len(candidates.transcripts) == 0 {
		cmd.Println("Nothing to backfill.")
		return
	}
	if !backfillClaimsApply {
		cmd.Printf(
			"Dry run (%d active topic(s)): would enqueue %d text message(s) and "+
				"%d video transcript(s). Re-run with --apply.\n",
			candidates.activeTopics,
			len(candidates.messages),
			len(candidates.transcripts),
		)
		return
	}

	workers := river.NewWorkers()
	river.AddWorker(workers, &registrationOnlyClaimCheckWorker{})
	riverClient, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}},
		Workers: workers,
	})
	if err != nil {
		cmd.PrintErrf("Failed to initialize job queue client: %s\n", err.Error())
		os.Exit(1)
		return
	}

	if err := enqueueClaimBackfill(
		ctx, pool, riverClient, claimBackfillInsertOpts(), candidates,
	); err != nil {
		cmd.PrintErrf("Failed to enqueue claim backfill, nothing was enqueued: %s\n", err.Error())
		os.Exit(1)
		return
	}

	cmd.Printf(
		"Enqueued %d text message(s) and %d video transcript(s). `wob serve` will process them.\n",
		len(candidates.messages), len(candidates.transcripts),
	)
}

// loadClaimBackfillCandidates checks that at least one topic is active, then
// lists the text messages and video transcripts that have no claim yet.
func loadClaimBackfillCandidates(
	ctx context.Context,
	queries *db.Queries,
	maxRows int32,
) (*claimBackfillCandidates, error) {
	activeTopics, err := queries.CountActiveTopics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count active topics: %w", err)
	}
	if activeTopics == 0 {
		return nil, errNoActiveTopics
	}

	since := time.Unix(0, 0).UTC()

	messages, err := queries.ListUserMessagesWithoutClaims(
		ctx,
		db.ListUserMessagesWithoutClaimsParams{Since: since, MaxRows: maxRows},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list messages: %w", err)
	}
	transcripts, err := queries.ListVideoTranscriptsWithoutClaims(
		ctx,
		db.ListVideoTranscriptsWithoutClaimsParams{Since: since, MaxRows: maxRows},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list video transcripts: %w", err)
	}

	return &claimBackfillCandidates{
		activeTopics: activeTopics,
		messages:     messages,
		transcripts:  transcripts,
	}, nil
}

// claimBackfillInsertOpts routes the jobs to the claim_backfill queue. Completed
// is listed in ByState so a job that already ran for a message (even one that
// produced no claim, like a greeting) is not enqueued again.
func claimBackfillInsertOpts() *river.InsertOpts {
	return &river.InsertOpts{
		Queue:       queue.QueueClaimBackfill,
		MaxAttempts: 3,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
			ByState: []rivertype.JobState{
				rivertype.JobStateAvailable,
				rivertype.JobStateCompleted,
				rivertype.JobStatePending,
				rivertype.JobStateRetryable,
				rivertype.JobStateRunning,
				rivertype.JobStateScheduled,
			},
		},
	}
}

// enqueueClaimBackfill inserts every claim-check job in one Postgres transaction.
func enqueueClaimBackfill(
	ctx context.Context,
	pool *pgxpool.Pool,
	riverClient *river.Client[pgx.Tx],
	opts *river.InsertOpts,
	candidates *claimBackfillCandidates,
) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Rollback is a no-op once Commit has succeeded.
	defer func() { _ = tx.Rollback(ctx) }()

	for _, m := range candidates.messages {
		if _, err := riverClient.InsertTx(ctx, tx, queue.ClaimCheckJob{
			Text:       m.Content,
			MessageID:  m.ID.String(),
			Platform:   model.ClaimPlatformWhatsApp,
			Format:     string(model.ClaimFormatText),
			OccurredAt: m.CreatedAt,
		}, opts); err != nil {
			return fmt.Errorf("failed to enqueue message %s: %w", m.ID, err)
		}
	}

	for _, t := range candidates.transcripts {
		if _, err := riverClient.InsertTx(ctx, tx, queue.ClaimCheckJob{
			Text:       t.Content,
			MessageID:  t.MessageID.String(),
			Platform:   model.ClaimPlatformTikTok,
			Format:     string(model.ClaimFormatVideo),
			OccurredAt: t.CreatedAt,
		}, opts); err != nil {
			return fmt.Errorf("failed to enqueue transcript of message %s: %w", t.MessageID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}