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

// AnalyzeSingleVideoWorker analyzes one transcribed video.
type AnalyzeSingleVideoWorker struct {
	river.WorkerDefaults[queue.AnalyzeSingleVideoJob]
	analyzeService *service.AnalyzeService
}

// NewAnalyzeSingleVideoWorker creates a new AnalyzeSingleVideoWorker.
func NewAnalyzeSingleVideoWorker(analyzeService *service.AnalyzeService) *AnalyzeSingleVideoWorker {
	return &AnalyzeSingleVideoWorker{analyzeService: analyzeService}
}

func (w *AnalyzeSingleVideoWorker) Timeout(
	_ *river.Job[queue.AnalyzeSingleVideoJob],
) time.Duration {
	return analyzeJobTimeout
}

func (w *AnalyzeSingleVideoWorker) NextRetry(
	job *river.Job[queue.AnalyzeSingleVideoJob],
) time.Time {
	i := min(max(job.Attempt-1, 0), len(analyzeRetryDelays)-1)
	return time.Now().Add(analyzeRetryDelays[i])
}

// Work analyzes the video. When the last attempt fails, the video is marked as
// failed so it is not analyzed again.
func (w *AnalyzeSingleVideoWorker) Work(
	ctx context.Context,
	job *river.Job[queue.AnalyzeSingleVideoJob],
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
func (w *AnalyzeSingleVideoWorker) handleAnalyzeError(
	ctx context.Context,
	job *river.Job[queue.AnalyzeSingleVideoJob],
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
	log.Printf("[AnalyzeSingleVideoWorker] video %s marked as failed after %d attempts: %v",
		videoID, job.Attempt, analyzeErr)
	return wrapped
}

// AnalyzeMultipleVideoWorker enqueues an analysis job for every transcribed video.
type AnalyzeMultipleVideoWorker struct {
	river.WorkerDefaults[queue.AnalyzeMultipleVideoJob]
	analyzeService *service.AnalyzeService
}

// NewAnalyzeMultipleVideoWorker creates a new AnalyzeMultipleVideoWorker.
func NewAnalyzeMultipleVideoWorker(
	analyzeService *service.AnalyzeService,
) *AnalyzeMultipleVideoWorker {
	return &AnalyzeMultipleVideoWorker{analyzeService: analyzeService}
}

func (w *AnalyzeMultipleVideoWorker) Timeout(
	_ *river.Job[queue.AnalyzeMultipleVideoJob],
) time.Duration {
	return analyzeDispatchTimeout
}

// Work enqueues an analysis job for every video that is still 'transcribed'.
func (w *AnalyzeMultipleVideoWorker) Work(
	ctx context.Context,
	_ *river.Job[queue.AnalyzeMultipleVideoJob],
) error {
	count, err := w.analyzeService.EnqueueTranscribedVideos(ctx)
	log.Printf("[AnalyzeMultipleVideoWorker] enqueued=%d", count)
	if err != nil {
		return fmt.Errorf("failed to enqueue video analysis: %w", err)
	}
	return nil
}
