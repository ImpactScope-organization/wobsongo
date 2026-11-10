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

	// DefaultScreenConcurrency is how many jobs of the screening queue run at
	// the same time (its MaxWorkers), unless SCREEN_MAX_CONCURRENCY says otherwise.
	DefaultScreenConcurrency = 2

	// ScreenMaxAttempts is how many times River runs a ScreenVideoJob. After
	// the last failed attempt the video is marked as failed.
	ScreenMaxAttempts = 3
)

// screenUniqueStates are the job states in which an identical screening job
// counts as a duplicate. Completed jobs are left out on purpose: a finished
// job must never block a new one, for example when a video is put back to
// 'discovered' to be screened again.
var screenUniqueStates = []rivertype.JobState{
	rivertype.JobStateAvailable,
	rivertype.JobStatePending,
	rivertype.JobStateRunning,
	rivertype.JobStateRetryable,
	rivertype.JobStateScheduled,
}

// ScreenVideosJob enqueues a ScreenVideoJob for every video that is still in
// the 'discovered' status.
type ScreenVideosJob struct{}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (ScreenVideosJob) Kind() string { return string(JobTypeScreenVideos) }

// InsertOpts implements river.JobArgsWithInsertOpts. Only one run may be
// waiting or running at a time, so overlapping triggers collapse into one.
// There is no retry: the periodic run picks up whatever is left.
func (ScreenVideosJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueScreening,
		MaxAttempts: 1,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: screenUniqueStates,
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
// once while a job for it is waiting, running or waiting for a retry. After
// ScreenMaxAttempts failed attempts the video is marked as failed.
func (ScreenVideoJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueScreening,
		MaxAttempts: ScreenMaxAttempts,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: screenUniqueStates,
		},
	}
}
