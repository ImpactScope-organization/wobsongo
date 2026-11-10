package queue

import (
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

const (
	// ScreenCheckInterval is how often River enqueues a screening run on its
	// own, so videos left over by a failed run or a missed enqueue are still
	// picked up.
	ScreenCheckInterval = 15 * time.Minute

	// DefaultScreenConcurrency is how many videos are screened at the same
	// time (the MaxWorkers of the screening queue), unless
	// SCREEN_MAX_CONCURRENCY says otherwise.
	DefaultScreenConcurrency = 2
)

// ScreenVideosJob enqueues a ScreenVideoJob for every video that is still in
// the 'discovered' status.
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

// ScreenVideoJob screens one discovered video from its caption and hashtags.
type ScreenVideoJob struct {
	// VideoID is the videos.id of the video to screen.
	VideoID string `json:"video_id"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (ScreenVideoJob) Kind() string { return string(JobTypeScreenVideo) }

// InsertOpts implements river.JobArgsWithInsertOpts. A video is only queued
// once even if the job is enqueued twice. There is no retry: the periodic
// ScreenVideosJob enqueues the video again if it is still 'discovered'.
func (ScreenVideoJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueScreening,
		MaxAttempts: 1,
		UniqueOpts:  river.UniqueOpts{ByArgs: true},
	}
}
