package worker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

const (
	// Statuses understood by wa-bot's /callback/tts-done endpoint.
	ttsStatusCompleted = "completed"
	ttsStatusFailed    = "failed"

	// ttsPresignTTLSeconds is how long Twilio/Telegram have to fetch the reply
	// audio. They fetch it right after the callback, so this is generous.
	ttsPresignTTLSeconds int64 = 900
)

// SpeechWorker turns a bot reply into a voice message, stores the audio in S3
// and tells wa-bot where to fetch it through the TTS callback.
type SpeechWorker struct {
	river.WorkerDefaults[queue.SpeechJob]
	speechService *service.SpeechService
	botClient     *external.BotClient
	store         data.RawObjectStore
	presigner     data.ObjectPresigner
}

// NewSpeechWorker creates a new SpeechWorker.
func NewSpeechWorker(
	speechService *service.SpeechService,
	botClient *external.BotClient,
	store data.RawObjectStore,
	presigner data.ObjectPresigner,
) *SpeechWorker {
	return &SpeechWorker{
		speechService: speechService,
		botClient:     botClient,
		store:         store,
		presigner:     presigner,
	}
}

func (w *SpeechWorker) Timeout(_ *river.Job[queue.SpeechJob]) time.Duration {
	return queue.SpeechJobTTL
}

// Work synthesizes the audio and notifies the bot with the result.
func (w *SpeechWorker) Work(ctx context.Context, job *river.Job[queue.SpeechJob]) error {
	log.Printf("[SpeechWorker] Starting speech synthesis for JobID=%s", job.Args.JobID)

	// A job that waited too long in the queue is useless: wa-bot has given up or
	// is about to. Skip it without touching the GPU.
	deadline := job.CreatedAt.Add(queue.SpeechJobTTL)
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

	key := ttsObjectKey(job.Args.JobID)
	if err := w.store.PutObject(
		ctx,
		key,
		bytes.NewReader(audio),
		int64(len(audio)),
		"audio/ogg",
	); err != nil {
		err = fmt.Errorf("failed to store synthesized audio: %w", err)
		w.reportFailure(ctx, job.Args.JobID, err)
		return err
	}
	audioURL, err := w.presigner.GetPresignedGETURL(ctx, key, ttsPresignTTLSeconds)
	if err != nil {
		err = fmt.Errorf("failed to presign synthesized audio: %w", err)
		w.reportFailure(ctx, job.Args.JobID, err)
		return err
	}

	if notifyErr := w.notifyBot(
		ctx,
		job.Args.JobID,
		ttsStatusCompleted,
		audioURL,
	); notifyErr != nil {
		log.Printf("[SpeechWorker] Failed to notify bot (audio will be lost): %v", notifyErr)
	}

	return nil
}

// reportFailure logs a failed job and tells the bot, so it can send the text
// answer right away instead of waiting for its own timer.
func (w *SpeechWorker) reportFailure(ctx context.Context, jobID string, cause error) {
	log.Printf("[SpeechWorker] failure for JobID=%s: %v", jobID, cause)
	if err := w.notifyBot(ctx, jobID, ttsStatusFailed, ""); err != nil {
		log.Printf("[SpeechWorker] Failed to notify bot of failure: %v", err)
	}
}

// notifyBot reports the job result to the bot.
func (w *SpeechWorker) notifyBot(
	ctx context.Context,
	jobID, status, audioURL string,
) error {
	notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), queue.SpeechNotifyTimeout)
	defer cancel()

	return w.botClient.NotifyTTSDone(notifyCtx, jobID, status, audioURL)
}

// ttsObjectKey returns the S3 key of a synthesized reply.
func ttsObjectKey(jobID string) string {
	sum := sha256.Sum256([]byte(jobID))
	return data.ObjectPrefixForIntent(
		data.TTSAudioUploadIntent,
	) + hex.EncodeToString(
		sum[:],
	) + ".ogg"
}
