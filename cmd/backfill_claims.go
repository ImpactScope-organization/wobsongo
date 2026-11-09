package cmd

import (
	"context"
	"os"
	"time"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
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

	queries := db.New(pool)

	// Claims need a topic: with none configured every check would succeed
	// without storing anything, wasting LLM calls.
	activeTopics, err := queries.CountActiveTopics(ctx)
	if err != nil {
		cmd.PrintErrf("Failed to count active topics: %s\n", err.Error())
		os.Exit(1)
		return
	}
	if activeTopics == 0 {
		cmd.PrintErrln("No active topics configured. Add at least one topic, then re-run.")
		os.Exit(1)
		return
	}

	since := time.Unix(0, 0).UTC()
	maxRows := backfillClaimsLimit

	messages, err := queries.ListUserMessagesWithoutClaims(
		ctx,
		db.ListUserMessagesWithoutClaimsParams{Since: since, MaxRows: maxRows},
	)
	if err != nil {
		cmd.PrintErrf("Failed to list messages: %s\n", err.Error())
		os.Exit(1)
		return
	}
	transcripts, err := queries.ListVideoTranscriptsWithoutClaims(
		ctx,
		db.ListVideoTranscriptsWithoutClaimsParams{Since: since, MaxRows: maxRows},
	)
	if err != nil {
		cmd.PrintErrf("Failed to list video transcripts: %s\n", err.Error())
		os.Exit(1)
		return
	}

	if len(messages) == 0 && len(transcripts) == 0 {
		cmd.Println("Nothing to backfill.")
		return
	}
	if !backfillClaimsApply {
		cmd.Printf(
			"Dry run (%d active topic(s)): would enqueue %d text message(s) and "+
				"%d video transcript(s). Re-run with --apply.\n",
			activeTopics,
			len(messages),
			len(transcripts),
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

	opts := &river.InsertOpts{
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

	for _, m := range messages {
		if _, err := riverClient.Insert(ctx, queue.ClaimCheckJob{
			Text:       m.Content,
			MessageID:  m.ID.String(),
			Platform:   model.ClaimPlatformWhatsApp,
			Format:     string(model.ClaimFormatText),
			OccurredAt: m.CreatedAt,
		}, opts); err != nil {
			cmd.PrintErrf("Failed to enqueue message %s: %s\n", m.ID, err.Error())
			os.Exit(1)
			return
		}
	}

	for _, t := range transcripts {
		if _, err := riverClient.Insert(ctx, queue.ClaimCheckJob{
			Text:       t.Content,
			MessageID:  t.MessageID.String(),
			Platform:   model.ClaimPlatformTikTok,
			Format:     string(model.ClaimFormatVideo),
			OccurredAt: t.CreatedAt,
		}, opts); err != nil {
			cmd.PrintErrf(
				"Failed to enqueue transcript of message %s: %s\n",
				t.MessageID,
				err.Error(),
			)
			os.Exit(1)
			return
		}
	}

	cmd.Printf(
		"Enqueued %d text message(s) and %d video transcript(s). `wob serve` will process them.\n",
		len(messages), len(transcripts),
	)
}