package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

// screenJobTimeout bounds one screening run.
const screenJobTimeout = 30 * time.Minute

// ScreenVideosWorker screens every discovered video for relevance.
type ScreenVideosWorker struct {
	river.WorkerDefaults[queue.ScreenVideosJob]
	screenService *service.ScreenService
}

// NewScreenVideosWorker creates a new ScreenVideosWorker.
func NewScreenVideosWorker(screenService *service.ScreenService) *ScreenVideosWorker {
	return &ScreenVideosWorker{screenService: screenService}
}

func (w *ScreenVideosWorker) Timeout(_ *river.Job[queue.ScreenVideosJob]) time.Duration {
	return screenJobTimeout
}

// Work screens every video that is still 'discovered'.
func (w *ScreenVideosWorker) Work(
	ctx context.Context,
	_ *river.Job[queue.ScreenVideosJob],
) error {
	summary, err := w.screenService.ScreenDiscoveredVideos(ctx)
	log.Printf(
		"[ScreenVideosWorker] screened_in=%d screened_out=%d failed=%d",
		summary.ScreenedIn, summary.ScreenedOut, summary.Failed,
	)
	if err != nil {
		return fmt.Errorf("failed to screen videos: %w", err)
	}
	return nil
}
