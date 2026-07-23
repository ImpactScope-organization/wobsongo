package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)


const claimCheckJobTimeout = 3 * time.Minute

// ClaimCheckWorker runs the full claim-check pipeline (analyze → retrieve →
// judge) for a piece of text.
type ClaimCheckWorker struct {
	river.WorkerDefaults[queue.ClaimCheckJob]
	claimService *service.ClaimService
	botClient    *external.BotClient
}

// NewClaimCheckWorker creates a new ClaimCheckWorker.
func NewClaimCheckWorker(
	claimService *service.ClaimService,
	botClient *external.BotClient,
) *ClaimCheckWorker {
	return &ClaimCheckWorker{claimService: claimService, botClient: botClient}
}

func (w *ClaimCheckWorker) Timeout(_ *river.Job[queue.ClaimCheckJob]) time.Duration {
	return claimCheckJobTimeout
}

// Work runs the claim check and notifies the bot with the formatted result.
func (w *ClaimCheckWorker) Work(ctx context.Context, job *river.Job[queue.ClaimCheckJob]) error {
	log.Printf(
		"[ClaimCheckWorker] Starting claim check for ExtractionID: %s",
		job.Args.ExtractionID,
	)

	result, err := w.claimService.CheckClaim(ctx, &dto.CheckClaimDTO{Text: job.Args.Text})
	if err != nil {
		err = fmt.Errorf("claim check failed: %w", err)
		w.notifyFailed(ctx, job.Args.ExtractionID, err)
		return err
	}

	message := result.FormattedMessage
	if !result.InScope {
		message = result.RefusalReason
	}

	log.Printf("[ClaimCheckWorker] Successfully processed ExtractionID %s", job.Args.ExtractionID)

	if notifyErr := w.botClient.NotifyExtractDone(
		ctx,
		job.Args.ExtractionID,
		"completed",
		"",
		&external.ExtractCallbackData{Answer: message},
	); notifyErr != nil {
		log.Printf("[ClaimCheckWorker] Failed to notify bot (answer will be lost): %v", notifyErr)
	}

	return nil
}

// notifyFailed notifies the bot that the claim-check job has failed.
func (w *ClaimCheckWorker) notifyFailed(ctx context.Context, extractionID string, cause error) {
	if extractionID == "" {
		return
	}
	if err := w.botClient.NotifyExtractDone(
		ctx,
		extractionID,
		"failed",
		cause.Error(),
		nil,
	); err != nil {
		log.Printf("[ClaimCheckWorker] failed to notify bot (failed case): %v", err)
	}
}
