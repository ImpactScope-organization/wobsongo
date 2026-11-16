package queue

import "github.com/riverqueue/river"

const (
	// DefaultTranscribeConcurrency is how many videos are transcribed at the
	// same time (Apify run + ASR call each).
	DefaultTranscribeConcurrency = 2

	// TranscribeMaxAttempts is how many times River runs a
	// TranscribeDiscoveredVideoJob. After the last failed attempt the video
	// is marked as failed.
	TranscribeMaxAttempts = 3

	// DefaultAnalyzeConcurrency is how many videos are analyzed at the same time.
	DefaultAnalyzeConcurrency = 2

	// AnalyzeMaxAttempts is how many times River runs an AnalyzeVideoJob.
	AnalyzeMaxAttempts = 3
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
