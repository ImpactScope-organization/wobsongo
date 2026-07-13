package queue

<<<<<<< HEAD
import "github.com/google/uuid"

=======
>>>>>>> main
// ExtractMediaDTO is the river job kind for extracting media from a given source.
type ExtractMediaDTO struct {
	TargetURL  string `json:"target_url"`
	WebhookURL string `json:"webhook_url"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (ExtractMediaDTO) Kind() string {
	return string(JobTypeExtractMedia)
}

// TranscriptionJobDTO is the river job kind for transcribing downloaded media via Modal.
type TranscriptionJobDTO struct {
	VideoID     uuid.UUID `json:"video_id"`
	DownloadURL string    `json:"download_url"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (TranscriptionJobDTO) Kind() string {
	return string(JobTypeTranscribeVideo)
}
