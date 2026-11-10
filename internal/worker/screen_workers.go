package worker

import (
	"context"
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
)

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

// Work screens the video.
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
		return fmt.Errorf("failed to screen video %s: %w", videoID, err)
	}
	return nil
}
