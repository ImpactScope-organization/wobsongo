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
	"github.com/spf13/cobra"
)

var (
	backfillClaimsApply bool
	backfillClaimsDays  int
	backfillClaimsLimit int
)

var backfillClaimsCmd = &cobra.Command{
	Use:   "backfill-claims",
	Short: "Enqueue claim checks for old user messages that never became claims",
	Long: "Finds user messages without a claim and enqueues one ClaimCheckJob each, on the\n" +
		"claim_backfill queue. The jobs are processed by `wob serve`, which then also\n" +
		"clusters the new claims into narratives. Nothing is deleted or overwritten and\n" +
		"no reply is sent to any user.\n" +
		"Without --apply this is a dry run.",
	Run: runBackfillClaims,
}

func init() {
	backfillClaimsCmd.Flags().
		BoolVar(&backfillClaimsApply, "apply", false, "Actually enqueue (default: dry run)")
	backfillClaimsCmd.Flags().
		IntVar(&backfillClaimsDays, "days", 30, "Only messages from the last N days")
	backfillClaimsCmd.Flags().
		IntVar(&backfillClaimsLimit, "limit", 1000, "Max messages per run")
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

	pool, err := pgxpool.New(ctx, config.PostgresURI)
	if err != nil {
		cmd.PrintErrf("Failed to connect to database: %s\n", err.Error())
		os.Exit(1)
		return
	}
	defer pool.Close()

	queries := db.New(pool)
	since := time.Now().AddDate(0, 0, -backfillClaimsDays)
	maxRows := int32(backfillClaimsLimit)

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
			"Dry run: would enqueue %d text message(s) and %d video transcript(s). Re-run with --apply.\n",
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
		UniqueOpts:  river.UniqueOpts{ByArgs: true},
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
