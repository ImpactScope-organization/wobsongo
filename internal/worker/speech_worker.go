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

const (
	// speechJobTimeout bounds one synthesis attempt (translation + TTS).
	speechJobTimeout = 4 * time.Minute

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

	// A job that waited too long in the queue is useless: wa-bot has given up or
	// is about to. Skip it without touching the GPU.
	deadline := job.CreatedAt.Add(speechJobTimeout)
	if !time.Now().Before(deadline) {
		err := fmt.Errorf(
			"speech job expired after waiting %s in the queue",
			time.Since(job.CreatedAt).Round(time.Second),
		)
		w.reportFailure(ctx, job.Args.JobID, err)
		return river.JobCancel(err)
	}
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	audio, err := w.speechService.Synthesize(ctx, job.Args.Text, job.Args.Language)
	if err != nil {
		err = fmt.Errorf("speech synthesis failed: %w", err)
		w.reportFailure(ctx, job.Args.JobID, err)
		return err
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

// reportFailure logs a failed job and tells the bot, so it can send the text
// answer right away instead of waiting for its own timer.
func (w *SpeechWorker) reportFailure(ctx context.Context, jobID string, cause error) {
	log.Printf("[SpeechWorker] failure for JobID=%s: %v", jobID, cause)
	if err := w.notifyBot(ctx, jobID, ttsStatusFailed, nil); err != nil {
		log.Printf("[SpeechWorker] Failed to notify bot of failure: %v", err)
	}
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
