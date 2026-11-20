package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

const (
	// transcribeJobTimeout bounds one transcription: an Apify run (up to
	// scraperMediaTimeout, 5 min) plus an ASR call (up to 5 min, cold start
	// included) and a minute for the database writes.
	transcribeJobTimeout = 11 * time.Minute

	// transcribeMarkFailedTimeout bounds the write that marks a video as failed.
	transcribeMarkFailedTimeout = 10 * time.Second

	// transcribeDispatchTimeout bounds one run that enqueues the transcription
	// jobs.
	transcribeDispatchTimeout = 5 * time.Minute
)

// transcribeRetryDelays is the wait before the next attempt.
var transcribeRetryDelays = []time.Duration{2 * time.Minute, 10 * time.Minute}

// TranscribeDiscoveredVideoWorker transcribes one screened-in video.
type TranscribeDiscoveredVideoWorker struct {
	river.WorkerDefaults[queue.TranscribeDiscoveredVideoJob]
	transcribeService *service.TranscribeService
}

// NewTranscribeDiscoveredVideoWorker creates a new TranscribeDiscoveredVideoWorker.
func NewTranscribeDiscoveredVideoWorker(
	transcribeService *service.TranscribeService,
) *TranscribeDiscoveredVideoWorker {
	return &TranscribeDiscoveredVideoWorker{transcribeService: transcribeService}
}

func (w *TranscribeDiscoveredVideoWorker) Timeout(
	_ *river.Job[queue.TranscribeDiscoveredVideoJob],
) time.Duration {
	return transcribeJobTimeout
}

func (w *TranscribeDiscoveredVideoWorker) NextRetry(
	job *river.Job[queue.TranscribeDiscoveredVideoJob],
) time.Time {
	i := min(max(job.Attempt-1, 0), len(transcribeRetryDelays)-1)
	return time.Now().Add(transcribeRetryDelays[i])
}

// Work transcribes the video. When the last attempt fails, or when the video
// has no media at all, the video is marked as failed so it is not retried.
func (w *TranscribeDiscoveredVideoWorker) Work(
	ctx context.Context,
	job *river.Job[queue.TranscribeDiscoveredVideoJob],
) error {
	videoID, err := uuid.Parse(job.Args.VideoID)
	if err != nil {
		// A malformed id never becomes valid, so do not retry.
		return river.JobCancel(fmt.Errorf("invalid video id %q: %w", job.Args.VideoID, err))
	}
	if err := w.transcribeService.TranscribeVideo(ctx, videoID); err != nil {
		return w.handleTranscribeError(ctx, job, videoID, err)
	}
	return nil
}

// handleTranscribeError returns the error of a failed attempt. On the last
// attempt, or when retrying cannot help, it also marks the video as failed,
// unless the server is shutting down.
func (w *TranscribeDiscoveredVideoWorker) handleTranscribeError(
	ctx context.Context,
	job *river.Job[queue.TranscribeDiscoveredVideoJob],
	videoID uuid.UUID,
	transcribeErr error,
) error {
	wrapped := fmt.Errorf("failed to process transcription of video %s: %w", videoID, transcribeErr)

	// Without media every retry would spend an Apify run for nothing.
	permanent := errors.Is(transcribeErr, data.ErrMediaUnavailable)
	lastAttempt := job.Attempt >= job.MaxAttempts
	shuttingDown := errors.Is(ctx.Err(), context.Canceled)
	if (!permanent && !lastAttempt) || shuttingDown {
		return wrapped
	}

	markCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), transcribeMarkFailedTimeout)
	defer cancel()

	if err := w.transcribeService.MarkTranscribeFailed(
		markCtx,
		videoID,
		transcribeErr,
	); err != nil {
		return errors.Join(wrapped, err)
	}
	log.Printf("[TranscribeDiscoveredVideoWorker] video %s marked as failed after %d attempts: %v",
		videoID, job.Attempt, transcribeErr)

	if permanent {
		return river.JobCancel(wrapped)
	}
	return wrapped
}

// TranscribeDiscoveredVideosWorker enqueues a transcription job for every
// screened-in video.
type TranscribeDiscoveredVideosWorker struct {
	river.WorkerDefaults[queue.TranscribeDiscoveredVideosJob]
	transcribeService *service.TranscribeService
}

// NewTranscribeDiscoveredVideosWorker creates a new TranscribeDiscoveredVideosWorker.
func NewTranscribeDiscoveredVideosWorker(
	transcribeService *service.TranscribeService,
) *TranscribeDiscoveredVideosWorker {
	return &TranscribeDiscoveredVideosWorker{transcribeService: transcribeService}
}

func (w *TranscribeDiscoveredVideosWorker) Timeout(
	_ *river.Job[queue.TranscribeDiscoveredVideosJob],
) time.Duration {
	return transcribeDispatchTimeout
}

// Work enqueues a transcription job for every video that is still 'screened_in'.
func (w *TranscribeDiscoveredVideosWorker) Work(
	ctx context.Context,
	_ *river.Job[queue.TranscribeDiscoveredVideosJob],
) error {
	count, err := w.transcribeService.EnqueueScreenedInVideos(ctx)
	log.Printf("[TranscribeDiscoveredVideosWorker] enqueued=%d", count)
	if err != nil {
		return fmt.Errorf("failed to enqueue video transcription: %w", err)
	}
	return nil
}
