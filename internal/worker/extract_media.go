package worker

import (
	"context"
	"fmt"
	"log"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/riverqueue/river"
)

// workerComponentExtractMedia is this worker's name for notifyBotFailed's log prefix.
const workerComponentExtractMedia = "ExtractMediaWorker"

// ExtractMediaWorker is a River worker that handles media extraction jobs.
type ExtractMediaWorker struct {
	// Embedding River's default worker behavior for the specific DTO.
	river.WorkerDefaults[queue.ExtractMediaDTO]
	// Extractor is the interface that defines how to trigger media extraction.
	Extractor data.MediaExtractor
	botClient *external.BotClient
}

// NewExtractMediaWorker is a constructor for ExtractMediaWorker.
func NewExtractMediaWorker(
	extractor data.MediaExtractor,
	botClient *external.BotClient,
) *ExtractMediaWorker {
	return &ExtractMediaWorker{
		Extractor: extractor,
		botClient: botClient,
	}
}

// Work is the main method that gets called when a job is dequeued.
func (w *ExtractMediaWorker) Work(
	ctx context.Context,
	job *river.Job[queue.ExtractMediaDTO],
) error {
	log.Printf("[ExtractMediaWorker] Processing job %d: extracting media for target URL %s",
		job.ID, job.Args.TargetURL)

	internal.NotifyBotProgress(ctx, w.botClient, workerComponentExtractMedia, job.Args.ExtractionID,
		internal.MsgCheckingVideo)

	// Constructing the DTO for the media extraction request based on the queue payload.
	req := dto.ExtractionRequest{
		TargetURL:  job.Args.TargetURL,
		WebhookURL: job.Args.WebhookURL,
	}

	// Calling the external media extractor to trigger the extraction process.
	if err := w.Extractor.TriggerAudioExtraction(ctx, req); err != nil {
		err = fmt.Errorf("failed to trigger audio extraction via Apify: %w", err)
		internal.NotifyBotFailed(
			ctx,
			w.botClient,
			workerComponentExtractMedia,
			job.Args.ExtractionID,
			err,
			"Une erreur est survenue pendant le traitement de la vidéo. Réessaie plus tard.",
		)
		return err
	}

	log.Printf(
		"[ExtractMediaWorker] Job %d completed: media extraction triggered successfully",
		job.ID,
	)

	return nil
}
