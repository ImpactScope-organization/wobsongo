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

// ScreenMultipleVideoWorker enqueues a screening job for every discovered video.
type ScreenMultipleVideoWorker struct {
	river.WorkerDefaults[queue.ScreenMultipleVideoJob]
	screenService *service.ScreenService
}

// NewScreenMultipleVideoWorker creates a new ScreenMultipleVideoWorker.
func NewScreenMultipleVideoWorker(screenService *service.ScreenService) *ScreenMultipleVideoWorker {
	return &ScreenMultipleVideoWorker{screenService: screenService}
}

func (w *ScreenMultipleVideoWorker) Timeout(
	_ *river.Job[queue.ScreenMultipleVideoJob],
) time.Duration {
	return screenDispatchTimeout
}

// Work enqueues a screening job for every video that is still 'discovered'.
func (w *ScreenMultipleVideoWorker) Work(
	ctx context.Context,
	_ *river.Job[queue.ScreenMultipleVideoJob],
) error {
	count, err := w.screenService.EnqueueDiscoveredVideos(ctx)
	log.Printf("[ScreenMultipleVideosWorker] enqueued=%d", count)
	if err != nil {
		return fmt.Errorf("failed to enqueue video screening: %w", err)
	}
	return nil
}

// ScreenSingleVideoWorker screens one video.
type ScreenSingleVideoWorker struct {
	river.WorkerDefaults[queue.ScreenSingleVideoJob]
	screenService *service.ScreenService
}

// NewScreenSingleVideoWorker creates a new ScreenSingleVideoWorker.
func NewScreenSingleVideoWorker(screenService *service.ScreenService) *ScreenSingleVideoWorker {
	return &ScreenSingleVideoWorker{screenService: screenService}
}

func (w *ScreenSingleVideoWorker) Timeout(_ *river.Job[queue.ScreenSingleVideoJob]) time.Duration {
	return screenVideoJobTimeout
}

// NextRetry waits longer than River's default, so a short outage of the
// analyzer does not use up every attempt.
func (w *ScreenSingleVideoWorker) NextRetry(job *river.Job[queue.ScreenSingleVideoJob]) time.Time {
	i := min(max(job.Attempt-1, 0), len(screenRetryDelays)-1)
	return time.Now().Add(screenRetryDelays[i])
}

// Work screens the video. When the last attempt fails, the video is marked as
// failed so it is not screened again.
func (w *ScreenSingleVideoWorker) Work(
	ctx context.Context,
	job *river.Job[queue.ScreenSingleVideoJob],
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
func (w *ScreenSingleVideoWorker) handleScreenError(
	ctx context.Context,
	job *river.Job[queue.ScreenSingleVideoJob],
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
	log.Printf("[ScreenSingleVideoWorker] video %s marked as failed after %d attempts: %v",
		videoID, job.Attempt, screenErr)
	return wrapped
}
