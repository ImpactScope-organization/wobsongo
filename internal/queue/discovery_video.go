package queue

import (
	"time"

	"github.com/riverqueue/river"
)

const (
	// DefaultTranscribeConcurrency is how many jobs of the transcription queue
	// run at the same time (its MaxWorkers; Apify run + ASR call each), unless
	// TRANSCRIBE_MAX_CONCURRENCY says otherwise.
	DefaultTranscribeConcurrency = 2

	// TranscribeMaxAttempts is how many times River runs a
	// TranscribeDiscoveredVideoJob. After the last failed attempt the video
	// is marked as failed.
	TranscribeMaxAttempts = 3

	// DefaultAnalyzeConcurrency is how many jobs of the analysis queue run at
	// the same time (its MaxWorkers), unless ANALYZE_MAX_CONCURRENCY says
	// otherwise.
	DefaultAnalyzeConcurrency = 2

	// AnalyzeMaxAttempts is how many times River runs an AnalyzeVideoJob.
	AnalyzeMaxAttempts = 3

	// TranscribeCheckInterval is how often River enqueues a transcription
	// sweep on its own, so screened-in videos left over by a failed run or a
	// missed enqueue are still picked up.
	TranscribeCheckInterval = 15 * time.Minute

	// TranscribeDispatchMaxWorkers allows one transcription dispatch run at a
	// time.
	TranscribeDispatchMaxWorkers = 1

	// AnalyzeCheckInterval is how often River enqueues an analysis sweep on
	// its own, so transcribed videos left over by a failed run or a missed
	// enqueue are still picked up.
	AnalyzeCheckInterval = 15 * time.Minute

	// AnalyzeDispatchMaxWorkers allows one analysis dispatch run at a time.
	AnalyzeDispatchMaxWorkers = 1
)

// TranscribeDiscoveredVideoJob transcribes one screened-in video.
type TranscribeDiscoveredVideoJob struct {
	// VideoID is the videos.id of the video to transcribe.
	VideoID string `json:"video_id"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (TranscribeDiscoveredVideoJob) Kind() string {
	return string(JobTypeTranscribeDiscoveredVideo)
}

// InsertOpts implements river.JobArgsWithInsertOpts.
func (TranscribeDiscoveredVideoJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueTranscription,
		MaxAttempts: TranscribeMaxAttempts,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: videoJobUniqueStates,
		},
	}
}

// AnalyzeVideoJob extracts, verifies and stores the claims of one
// transcribed video.
type AnalyzeVideoJob struct {
	// VideoID is the videos.id of the video to analyze.
	VideoID string `json:"video_id"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (AnalyzeVideoJob) Kind() string { return string(JobTypeAnalyzeVideo) }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (AnalyzeVideoJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueAnalysis,
		MaxAttempts: AnalyzeMaxAttempts,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: videoJobUniqueStates,
		},
	}
}

// TranscribeDiscoveredVideosJob enqueues a TranscribeDiscoveredVideoJob for
// every video that is still in the 'screened_in' status.
type TranscribeDiscoveredVideosJob struct{}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (TranscribeDiscoveredVideosJob) Kind() string {
	return string(JobTypeTranscribeDiscoveredVideos)
}

// InsertOpts implements river.JobArgsWithInsertOpts. Only one run may be
// waiting or running at a time, so overlapping triggers collapse into one.
func (TranscribeDiscoveredVideosJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueTranscribeDispatch,
		MaxAttempts: 1,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: videoJobUniqueStates,
		},
	}
}

// AnalyzeVideosJob enqueues an AnalyzeVideoJob for every video that is still
// in the 'transcribed' status.
type AnalyzeVideosJob struct{}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (AnalyzeVideosJob) Kind() string { return string(JobTypeAnalyzeVideos) }

// InsertOpts implements river.JobArgsWithInsertOpts. Only one run may be
// waiting or running at a time, so overlapping triggers collapse into one.
func (AnalyzeVideosJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueAnalyzeDispatch,
		MaxAttempts: 1,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: videoJobUniqueStates,
		},
	}
}
