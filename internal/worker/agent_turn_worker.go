package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

// agentTurnJobTimeout is the maximum time allowed for a single agent turn.
const agentTurnJobTimeout = 3 * time.Minute

// workerComponentAgentTurn is this worker's name for notifyBotFailed's log prefix.
const workerComponentAgentTurn = "AgentTurnWorker"

// AgentTurnWorker processes a single agent conversation turn and sends
// the final reply using the standard bot callback.
type AgentTurnWorker struct {
	river.WorkerDefaults[queue.AgentTurnJob]
	agentService *service.AgentService
	botClient    *external.BotClient
}

// NewAgentTurnWorker creates a new AgentTurnWorker.
func NewAgentTurnWorker(
	agentService *service.AgentService,
	botClient *external.BotClient,
) *AgentTurnWorker {
	return &AgentTurnWorker{agentService: agentService, botClient: botClient}
}

func (w *AgentTurnWorker) Timeout(_ *river.Job[queue.AgentTurnJob]) time.Duration {
	return agentTurnJobTimeout
}

// Work runs the agent turn and notifies the bot with the formatted result.
func (w *AgentTurnWorker) Work(ctx context.Context, job *river.Job[queue.AgentTurnJob]) error {
	log.Printf(
		"[AgentTurnWorker] Starting agent turn for jid=%s ExtractionID=%s",
		job.Args.Jid, job.Args.ExtractionID,
	)

	answer, err := w.agentService.RunTurn(ctx, job.Args)
	if err != nil {
		err = fmt.Errorf("agent turn failed: %w", err)
		notifyBotFailed(ctx, w.botClient, workerComponentAgentTurn, job.Args.ExtractionID, err)
		return err
	}

	log.Printf(
		"[AgentTurnWorker] Successfully processed agent turn for jid=%s ExtractionID=%s",
		job.Args.Jid, job.Args.ExtractionID,
	)

	if notifyErr := w.botClient.NotifyExtractDone(
		ctx,
		job.Args.ExtractionID,
		"completed",
		"",
		&external.ExtractCallbackData{Answer: answer},
	); notifyErr != nil {
		log.Printf("[AgentTurnWorker] Failed to notify bot (answer will be lost): %v", notifyErr)
	}

	return nil
}
