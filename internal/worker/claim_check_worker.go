package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

const claimCheckJobTimeout = 3 * time.Minute

// workerComponentClaimCheck is this worker's name for notifyBotFailed's log prefix.
const workerComponentClaimCheck = "ClaimCheckWorker"

// ClaimCheckWorker runs the full claim-check pipeline (analyze → retrieve →
// judge) for a piece of text.
type ClaimCheckWorker struct {
	river.WorkerDefaults[queue.ClaimCheckJob]
	claimService        *service.ClaimService
	botClient           *external.BotClient
	conversationService *service.ConversationService
}

// NewClaimCheckWorker creates a new ClaimCheckWorker.
func NewClaimCheckWorker(
	claimService *service.ClaimService,
	botClient *external.BotClient,
	conversationService *service.ConversationService,
) *ClaimCheckWorker {
	return &ClaimCheckWorker{
		claimService:        claimService,
		botClient:           botClient,
		conversationService: conversationService,
	}
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

	internal.NotifyBotProgress(ctx, w.botClient, workerComponentClaimCheck, job.Args.ExtractionID,
		"🔍 Checking the claim...")

	result, err := w.claimService.CheckClaim(ctx, &dto.CheckClaimDTO{Text: job.Args.Text})
	if err != nil {
		err = fmt.Errorf("claim check failed: %w", err)
		internal.NotifyBotFailed(
			ctx,
			w.botClient,
			workerComponentClaimCheck,
			job.Args.ExtractionID,
			err,
		)
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

	if job.Args.Jid != "" {
		if err := w.conversationService.AppendMessage(
			ctx, job.Args.Jid, model.ConversationRoleAssistant, message,
		); err != nil {
			log.Printf(
				"[ClaimCheckWorker] failed to log claim result to conversation history: %v",
				err,
			)
		}
	}

	return nil
}
