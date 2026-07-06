package worker

import (
	"context"
	"fmt"
	"log"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/riverqueue/river"
)

// ExtractMediaWorker is a River worker that handles media extraction jobs.
type ExtractMediaWorker struct {
	// Embedding River's default worker behavior for the specific DTO.
	river.WorkerDefaults[queue.ExtractMediaDTO]
	// Extractor is the interface that defines how to trigger media extraction.
	Extractor data.MediaExtractor
}

// NewExtractMediaWorker is a constructor for ExtractMediaWorker.
func NewExtractMediaWorker(extractor data.MediaExtractor) *ExtractMediaWorker {
	return &ExtractMediaWorker{
		Extractor: extractor,
	}
}

// Work is the main method that gets called when a job is dequeued.
func (w *ExtractMediaWorker) Work(
	ctx context.Context,
	job *river.Job[queue.ExtractMediaDTO],
) error {
	log.Printf("[ExtractMediaWorker] Processing job %d: extracting media for target URL %s",
		job.ID, job.Args.TargetURL)

	// Constructing the DTO for the media extraction request based on the queue payload.
	req := dto.ExtractionRequest{
		TargetURL:  job.Args.TargetURL,
		WebhookURL: job.Args.WebhookURL,
	}

	// Calling the external media extractor to trigger the extraction process.
	err := w.Extractor.TriggerAudioExtraction(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to trigger audio extraction via Apify: %w", err)
	}

	log.Printf(
		"[ExtractMediaWorker] Job %d completed: media extraction triggered successfully",
		job.ID,
	)

	return nil
}
