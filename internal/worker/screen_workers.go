package worker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

const (
	// screenDispatchTimeout bounds one run that enqueues the screening jobs.
	screenDispatchTimeout = 5 * time.Minute

	// screenVideoJobTimeout bounds the screening of one video.
	screenVideoJobTimeout = 2 * time.Minute

	// screenMarkFailedTimeout bounds the write that marks a video as failed.
	screenMarkFailedTimeout = 10 * time.Second
)

// screenRetryDelays is the wait before the next attempt. Short delays would
// use every attempt up during one LLM outage.
var screenRetryDelays = []time.Duration{5 * time.Minute, 15 * time.Minute}

// ScreenVideosWorker enqueues a screening job for every discovered video.
type ScreenVideosWorker struct {
	river.WorkerDefaults[queue.ScreenVideosJob]
	screenService *service.ScreenService
}

// NewScreenVideosWorker creates a new ScreenVideosWorker.
func NewScreenVideosWorker(screenService *service.ScreenService) *ScreenVideosWorker {
	return &ScreenVideosWorker{screenService: screenService}
}

func (w *ScreenVideosWorker) Timeout(_ *river.Job[queue.ScreenVideosJob]) time.Duration {
	return screenDispatchTimeout
}

// Work enqueues a screening job for every video that is still 'discovered'.
func (w *ScreenVideosWorker) Work(
	ctx context.Context,
	_ *river.Job[queue.ScreenVideosJob],
) error {
	count, err := w.screenService.EnqueueDiscoveredVideos(ctx)
	log.Printf("[ScreenVideosWorker] enqueued=%d", count)
	if err != nil {
		return fmt.Errorf("failed to enqueue video screening: %w", err)
	}
	return nil
}

// ScreenVideoWorker screens one video.
type ScreenVideoWorker struct {
	river.WorkerDefaults[queue.ScreenVideoJob]
	screenService *service.ScreenService
}

// NewScreenVideoWorker creates a new ScreenVideoWorker.
func NewScreenVideoWorker(screenService *service.ScreenService) *ScreenVideoWorker {
	return &ScreenVideoWorker{screenService: screenService}
}

func (w *ScreenVideoWorker) Timeout(_ *river.Job[queue.ScreenVideoJob]) time.Duration {
	return screenVideoJobTimeout
}

// NextRetry waits longer than River's default, so a short outage of the
// analyzer does not use up every attempt.
func (w *ScreenVideoWorker) NextRetry(job *river.Job[queue.ScreenVideoJob]) time.Time {
	i := min(max(job.Attempt-1, 0), len(screenRetryDelays)-1)
	return time.Now().Add(screenRetryDelays[i])
}

// Work screens the video. When the last attempt fails, the video is marked as
// failed so it is not screened again.
func (w *ScreenVideoWorker) Work(
	ctx context.Context,
	job *river.Job[queue.ScreenVideoJob],
) error {
	videoID, err := uuid.Parse(job.Args.VideoID)
	if err != nil {
		// A malformed id never becomes valid, so do not retry.
		return river.JobCancel(fmt.Errorf("invalid video id %q: %w", job.Args.VideoID, err))
	}
	if err := w.screenService.ScreenVideo(ctx, videoID); err != nil {
		return w.handleScreenError(ctx, job, videoID, err)
	}
	return nil
}

// handleScreenError returns the error of a failed attempt. On the last attempt
// it also marks the video as failed, unless the server is shutting down.
func (w *ScreenVideoWorker) handleScreenError(
	ctx context.Context,
	job *river.Job[queue.ScreenVideoJob],
	videoID uuid.UUID,
	screenErr error,
) error {
	wrapped := fmt.Errorf("failed to screen video %s: %w", videoID, screenErr)

	lastAttempt := job.Attempt >= job.MaxAttempts
	shuttingDown := errors.Is(ctx.Err(), context.Canceled)
	if !lastAttempt || shuttingDown {
		return wrapped
	}

	markCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), screenMarkFailedTimeout)
	defer cancel()

	if err := w.screenService.MarkScreenFailed(markCtx, videoID, screenErr); err != nil {
		return errors.Join(wrapped, err)
	}
	log.Printf("[ScreenVideoWorker] video %s marked as failed after %d attempts: %v",
		videoID, job.Attempt, screenErr)
	return wrapped
}
