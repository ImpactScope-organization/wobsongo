package queue

import (
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

const (
	// ScreeningMaxWorkers allows one screening run at a time.
	ScreeningMaxWorkers = 1

	// ScreenCheckInterval is how often River enqueues a screening run on its
	// own, so videos left over by a failed run or a missed enqueue are still
	// picked up.
	ScreenCheckInterval = 15 * time.Minute

	// DefaultScreenConcurrency is how many videos are analyzed at the same
	// time, unless SCREEN_MAX_CONCURRENCY says otherwise.
	DefaultScreenConcurrency = 2
)

// ScreenVideosJob screens every video that is still in the 'discovered'
// status.
type ScreenVideosJob struct{}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (ScreenVideosJob) Kind() string { return string(JobTypeScreenVideos) }

// InsertOpts implements river.JobArgsWithInsertOpts. Only one run may be
// waiting or running at a time, so overlapping triggers collapse into one.
// Completed jobs are left out of ByState so a finished run never blocks the
// next one. There is no retry: the periodic run picks up whatever is left.
func (ScreenVideosJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueScreening,
		MaxAttempts: 1,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
			ByState: []rivertype.JobState{
				rivertype.JobStateAvailable,
				rivertype.JobStatePending,
				rivertype.JobStateRunning,
				rivertype.JobStateRetryable,
				rivertype.JobStateScheduled,
			},
		},
	}
}
