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
	// TranscribeSingleDiscoveredVideoJob. After the last failed attempt the video
	// is marked as failed.
	TranscribeMaxAttempts = 3

	// DefaultAnalyzeConcurrency is how many jobs of the analysis queue run at
	// the same time (its MaxWorkers), unless ANALYZE_MAX_CONCURRENCY says
	// otherwise.
	DefaultAnalyzeConcurrency = 2

	// AnalyzeMaxAttempts is how many times River runs an AnalyzeSingleVideoJob.
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

// TranscribeSingleDiscoveredVideoJob transcribes one screened-in video.
type TranscribeSingleDiscoveredVideoJob struct {
	// VideoID is the videos.id of the video to transcribe.
	VideoID string `json:"video_id"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (TranscribeSingleDiscoveredVideoJob) Kind() string {
	return string(JobTypeTranscribeSingleDiscoveredVideo)
}

// InsertOpts implements river.JobArgsWithInsertOpts.
func (TranscribeSingleDiscoveredVideoJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueTranscription,
		MaxAttempts: TranscribeMaxAttempts,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: videoJobUniqueStates,
		},
	}
}

// AnalyzeSingleVideoJob extracts, verifies and stores the claims of one
// transcribed video.
type AnalyzeSingleVideoJob struct {
	// VideoID is the videos.id of the video to analyze.
	VideoID string `json:"video_id"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (AnalyzeSingleVideoJob) Kind() string { return string(JobTypeAnalyzeSingleVideo) }

// InsertOpts implements river.JobArgsWithInsertOpts.
func (AnalyzeSingleVideoJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueAnalysis,
		MaxAttempts: AnalyzeMaxAttempts,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: videoJobUniqueStates,
		},
	}
}

// TranscribeMultipleDiscoveredVideoJob enqueues a TranscribeSingleDiscoveredVideoJob for
// every video that is still in the 'screened_in' status.
type TranscribeMultipleDiscoveredVideoJob struct{}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (TranscribeMultipleDiscoveredVideoJob) Kind() string {
	return string(JobTypeTranscribeMultipleDiscoveredVideo)
}

// InsertOpts implements river.JobArgsWithInsertOpts. Only one run may be
// waiting or running at a time, so overlapping triggers collapse into one.
func (TranscribeMultipleDiscoveredVideoJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueTranscribeDispatch,
		MaxAttempts: 1,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: videoJobUniqueStates,
		},
	}
}

// AnalyzeMultipleVideoJob enqueues an AnalyzeSingleVideoJob for every video that is still
// in the 'transcribed' status.
type AnalyzeMultipleVideoJob struct{}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (AnalyzeMultipleVideoJob) Kind() string { return string(JobTypeAnalyzeMultipleVideo) }

// InsertOpts implements river.JobArgsWithInsertOpts. Only one run may be
// waiting or running at a time, so overlapping triggers collapse into one.
func (AnalyzeMultipleVideoJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:       QueueAnalyzeDispatch,
		MaxAttempts: 1,
		UniqueOpts: river.UniqueOpts{
			ByArgs:  true,
			ByState: videoJobUniqueStates,
		},
	}
}
