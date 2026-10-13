package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

const (
	// speechJobTimeout bounds one synthesis attempt (translation + TTS).
	speechJobTimeout = 10 * time.Minute

	// speechNotifyTimeout bounds the callback that delivers the result to the bot.
	speechNotifyTimeout = 30 * time.Second

	// Statuses understood by wa-bot's /callback/tts-done endpoint.
	ttsStatusCompleted = "completed"
	ttsStatusFailed    = "failed"
)

// SpeechWorker turns a bot reply into a voice message and delivers the audio
// to wa-bot through the TTS callback.
type SpeechWorker struct {
	river.WorkerDefaults[queue.SpeechJob]
	speechService *service.SpeechService
	botClient     *external.BotClient
}

// NewSpeechWorker creates a new SpeechWorker.
func NewSpeechWorker(
	speechService *service.SpeechService,
	botClient *external.BotClient,
) *SpeechWorker {
	return &SpeechWorker{speechService: speechService, botClient: botClient}
}

func (w *SpeechWorker) Timeout(_ *river.Job[queue.SpeechJob]) time.Duration {
	return speechJobTimeout
}

// Work synthesizes the audio and notifies the bot with the result.
func (w *SpeechWorker) Work(ctx context.Context, job *river.Job[queue.SpeechJob]) error {
	log.Printf("[SpeechWorker] Starting speech synthesis for JobID=%s", job.Args.JobID)

	audio, err := w.speechService.Synthesize(ctx, job.Args.Text, job.Args.Language)
	if err != nil {
		return w.handleFailure(ctx, job, err)
	}

	log.Printf(
		"[SpeechWorker] Successfully synthesized %d bytes for JobID=%s",
		len(audio), job.Args.JobID,
	)

	if notifyErr := w.notifyBot(ctx, job.Args.JobID, ttsStatusCompleted, audio); notifyErr != nil {
		log.Printf("[SpeechWorker] Failed to notify bot (audio will be lost): %v", notifyErr)
	}

	return nil
}

// handleFailure decides what to do with a failed attempt.
func (w *SpeechWorker) handleFailure(
	ctx context.Context,
	job *river.Job[queue.SpeechJob],
	cause error,
) error {
	err := fmt.Errorf("speech synthesis failed: %w", cause)

	permanent := errors.Is(cause, service.ErrSpeechNotConfigured) ||
		errors.Is(cause, service.ErrSpeechEmptyText)
	lastAttempt := job.Attempt >= job.MaxAttempts

	if !permanent && !lastAttempt {
		log.Printf(
			"[SpeechWorker] attempt %d/%d failed for JobID=%s, will retry: %v",
			job.Attempt, job.MaxAttempts, job.Args.JobID, err,
		)
		return err
	}

	log.Printf("[SpeechWorker] failure for JobID=%s: %v", job.Args.JobID, err)
	if notifyErr := w.notifyBot(ctx, job.Args.JobID, ttsStatusFailed, nil); notifyErr != nil {
		log.Printf("[SpeechWorker] Failed to notify bot of failure: %v", notifyErr)
	}

	if permanent {
		// Retrying cannot fix a missing configuration or an empty text.
		return river.JobCancel(err)
	}
	return err
}

// notifyBot reports the job result to the bot.
func (w *SpeechWorker) notifyBot(
	ctx context.Context,
	jobID, status string,
	audio []byte,
) error {
	notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), speechNotifyTimeout)
	defer cancel()

	return w.botClient.NotifyTTSDone(notifyCtx, jobID, status, audio)
}
