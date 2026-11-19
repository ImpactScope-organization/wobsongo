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
	// analyzeJobTimeout bounds the analysis of one video: claim extraction, then
	// retrieval and judging of every sub-claim.
	analyzeJobTimeout = 5 * time.Minute

	// analyzeMarkFailedTimeout bounds the write that marks a video as failed.
	analyzeMarkFailedTimeout = 10 * time.Second

	// analyzeDispatchTimeout bounds one run that enqueues the analysis jobs.
	analyzeDispatchTimeout = 5 * time.Minute
)

// analyzeRetryDelays is the wait before the next attempt. Short delays would
// use every attempt up during one LLM outage.
var analyzeRetryDelays = []time.Duration{5 * time.Minute, 15 * time.Minute}

// AnalyzeVideoWorker analyzes one transcribed video.
type AnalyzeVideoWorker struct {
	river.WorkerDefaults[queue.AnalyzeVideoJob]
	analyzeService *service.AnalyzeService
}

// NewAnalyzeVideoWorker creates a new AnalyzeVideoWorker.
func NewAnalyzeVideoWorker(analyzeService *service.AnalyzeService) *AnalyzeVideoWorker {
	return &AnalyzeVideoWorker{analyzeService: analyzeService}
}

func (w *AnalyzeVideoWorker) Timeout(_ *river.Job[queue.AnalyzeVideoJob]) time.Duration {
	return analyzeJobTimeout
}

func (w *AnalyzeVideoWorker) NextRetry(job *river.Job[queue.AnalyzeVideoJob]) time.Time {
	i := min(max(job.Attempt-1, 0), len(analyzeRetryDelays)-1)
	return time.Now().Add(analyzeRetryDelays[i])
}

// Work analyzes the video. When the last attempt fails, the video is marked as
// failed so it is not analyzed again.
func (w *AnalyzeVideoWorker) Work(
	ctx context.Context,
	job *river.Job[queue.AnalyzeVideoJob],
) error {
	videoID, err := uuid.Parse(job.Args.VideoID)
	if err != nil {
		// A malformed id never becomes valid, so do not retry.
		return river.JobCancel(fmt.Errorf("invalid video id %q: %w", job.Args.VideoID, err))
	}
	if err := w.analyzeService.AnalyzeVideo(ctx, videoID); err != nil {
		return w.handleAnalyzeError(ctx, job, videoID, err)
	}
	return nil
}

// handleAnalyzeError returns the error of a failed attempt. On the last
// attempt it also marks the video as failed, unless the server is shutting down.
func (w *AnalyzeVideoWorker) handleAnalyzeError(
	ctx context.Context,
	job *river.Job[queue.AnalyzeVideoJob],
	videoID uuid.UUID,
	analyzeErr error,
) error {
	wrapped := fmt.Errorf("failed to process analysis of video %s: %w", videoID, analyzeErr)

	lastAttempt := job.Attempt >= job.MaxAttempts
	shuttingDown := errors.Is(ctx.Err(), context.Canceled)
	if !lastAttempt || shuttingDown {
		return wrapped
	}

	markCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), analyzeMarkFailedTimeout)
	defer cancel()

	if err := w.analyzeService.MarkAnalyzeFailed(markCtx, videoID, analyzeErr); err != nil {
		return errors.Join(wrapped, err)
	}
	log.Printf("[AnalyzeVideoWorker] video %s marked as failed after %d attempts: %v",
		videoID, job.Attempt, analyzeErr)
	return wrapped
}

// AnalyzeVideosWorker enqueues an analysis job for every transcribed video.
type AnalyzeVideosWorker struct {
	river.WorkerDefaults[queue.AnalyzeVideosJob]
	analyzeService *service.AnalyzeService
}

// NewAnalyzeVideosWorker creates a new AnalyzeVideosWorker.
func NewAnalyzeVideosWorker(analyzeService *service.AnalyzeService) *AnalyzeVideosWorker {
	return &AnalyzeVideosWorker{analyzeService: analyzeService}
}

func (w *AnalyzeVideosWorker) Timeout(_ *river.Job[queue.AnalyzeVideosJob]) time.Duration {
	return analyzeDispatchTimeout
}

// Work enqueues an analysis job for every video that is still 'transcribed'.
func (w *AnalyzeVideosWorker) Work(
	ctx context.Context,
	_ *river.Job[queue.AnalyzeVideosJob],
) error {
	count, err := w.analyzeService.EnqueueTranscribedVideos(ctx)
	log.Printf("[AnalyzeVideosWorker] enqueued=%d", count)
	if err != nil {
		return fmt.Errorf("failed to enqueue video analysis: %w", err)
	}
	return nil
}
