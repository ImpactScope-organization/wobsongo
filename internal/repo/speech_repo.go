package repo

import (
	"context"
	"fmt"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// speechWaitingStates are the River job states in which a speech job is
// queued but not yet running. Running jobs are bounded by the queue's
// MaxWorkers, so only waiting jobs count toward the pending limit.
var speechWaitingStates = []rivertype.JobState{
	rivertype.JobStateAvailable,
	rivertype.JobStatePending,
	rivertype.JobStateRetryable,
	rivertype.JobStateScheduled,
}

// speechRepo implements data.SpeechRepoer.
type speechRepo struct {
	riverClientFn  func() *river.Client[pgx.Tx]
	maxPendingJobs int
}

// NewSpeechRepo creates a new repository for speech-synthesis jobs.
func NewSpeechRepo(
	riverClientFn func() *river.Client[pgx.Tx],
	maxPendingJobs int,
) data.SpeechRepoer {
	return &speechRepo{
		riverClientFn:  riverClientFn,
		maxPendingJobs: maxPendingJobs,
	}
}

// EnqueueSpeech adds a speech job to the River queue. It returns
// data.ErrQueueFull when queue.SpeechMaxPendingJobs jobs are already waiting.
// The limit is a soft one: the check and the insert are not atomic, so a burst
// of concurrent requests can overshoot it by a few jobs. That is acceptable
// here, the goal is to stop unbounded growth, not to enforce an exact size.
func (r *speechRepo) EnqueueSpeech(ctx context.Context, payload queue.SpeechJob) error {
	client := r.riverClientFn()

	// Fetching at most maxPendingJobs rows is enough to know whether the queue is
	// at capacity, no need to count all of them.
	waiting, err := client.JobList(ctx, river.NewJobListParams().
		Queues(queue.QueueSpeechSynthesis).
		States(speechWaitingStates...).
		First(r.maxPendingJobs))
	if err != nil {
		return fmt.Errorf("failed to check speech queue length: %w", err)
	}
	if len(waiting.Jobs) >= r.maxPendingJobs {
		return fmt.Errorf(
			"%w: %d speech jobs already waiting",
			data.ErrQueueFull, len(waiting.Jobs),
		)
	}

	if _, err := client.Insert(ctx, payload, nil); err != nil {
		return fmt.Errorf("failed to insert speech job into river queue: %w", err)
	}
	return nil
}
