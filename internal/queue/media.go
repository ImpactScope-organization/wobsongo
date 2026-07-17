package queue

import (
	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

// ExtractMediaDTO is the river job kind for extracting media from a given source.
type ExtractMediaDTO struct {
	ExtractionID string `json:"extraction_id"`
	TargetURL    string `json:"target_url"`
	WebhookURL   string `json:"webhook_url"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (ExtractMediaDTO) Kind() string {
	return string(JobTypeExtractMedia)
}

// InsertOpts implements river.JobArgsWithInsertOpts.
func (ExtractMediaDTO) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueMediaProcessing}
}

// TranscriptionJob is the river job kind for transcribing downloaded media via Modal.
type TranscriptionJob struct {
	ExtractionID string    `json:"extraction_id"`
	VideoID      uuid.UUID `json:"video_id"`
	DownloadURL  string    `json:"download_url"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (TranscriptionJob) Kind() string {
	return string(JobTypeTranscribeVideo)
}

// InsertOpts implements river.JobArgsWithInsertOpts.
func (TranscriptionJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueMediaProcessing}
}
