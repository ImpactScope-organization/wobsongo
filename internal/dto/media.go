package dto

type ExtractStatus string

const (
	// StatusProcessing indicates that the extraction job is currently in progress.
	StatusProcessing ExtractStatus = "processing"

	// StatusCompleted indicates that the extraction job has finished successfully.
	StatusCompleted ExtractStatus = "completed"

	// StatusFailed indicates that the extraction job encountered an error and could not complete.
	StatusFailed ExtractStatus = "failed"
)

// ExtractionRequest represents the payload required to trigger a media extraction.
// We strictly define the types as strings and enforce validation rules.
type ExtractionRequest struct {
	// TargetURL is the link to the video (e.g., TikTok or Instagram reels).
	TargetURL string `json:"target_url" validate:"required,url"`

	// WebhookURL is the endpoint Apify will call once the process is complete.
	WebhookURL string `json:"webhook_url" validate:"required,url"`
}

// ExtractAPIRequest is the public-facing payload sent by the WA bot.
type ExtractAPIRequest struct {
	// URL is the target media link provided by the bot.
	URL string `json:"url,omitempty" validate:"required_without=Question,omitempty,url"`

	// Question is a free-text question to search the knowledge base.
	Question string `json:"question,omitempty" validate:"required_without=URL"`
}

// ExtractResponse is the response returned to the bot, both on first call
// (cache miss -> processing) and on the re-call after callback (cache hit
// -> completed) same shape either way.
type ExtractResponse struct {
	// Status indicates the current state of the extraction job.
	Status ExtractStatus `json:"status"`

	// JobID is the unique identifier for the extraction task.
	JobID string `json:"jobId"`

	// Data contains the final extraction results. It is omitted if the job is not yet completed.
	Data *ExtractData `json:"data,omitempty"`

	// Error contains the error message if the extraction failed. It is omitted otherwise.
	Error string `json:"error,omitempty"`
}

// ExtractData holds the successfully extracted information.
type ExtractData struct {
	// Transcript contains the transcribed text from the tiktok video.
	Transcript string `json:"transcript"`
}

// ApifyWebhookPayload represents the JSON sent by Apify.
type ApifyWebhookPayload struct {
	// EventType indicates the type of event that triggered the webhook.
	EventType string `json:"eventType" validate:"required"`

	// Resource contains details of the Apify Actor run that just finished.
	Resource ApifyResource `json:"resource" validate:"required"`
}

// ApifyResource contains details of the Apify Actor run that just finished.
type ApifyResource struct {
	// DefaultDatasetId is the ID of the dataset created by the Actor run.
	DefaultDatasetId string `json:"defaultDatasetId" validate:"required"`

	// Status indicates the final status of the Actor run (e.g., SUCCEEDED, FAILED).
	Status string `json:"status" validate:"required"`
}

// GetPresignedURLDTO is used for requesting a presigned URL for media upload.
type GetPresignedURLDTO struct {
	// Intent specifies the purpose of the media upload, e.g., "document".
	Intent string `query:"intent" validate:"required"`

	// Filename is the name of the file to be uploaded.
	Filename string `query:"filename" validate:"required,min=1,max=255"`

	// ContentType is the MIME type of the file to be uploaded.
	ContentType string `query:"content_type" validate:"required,mimetype"`
}

// GetPresignedGETURLDTO is used for requesting a presigned GET URL for accessing uploaded media.
type GetPresignedGETURLDTO struct {
	// S3Key is the full S3 object key (e.g., "documents/abc123.pdf")
	S3Key string `query:"s3_key" validate:"required,s3key"`

	// TTL is the time-to-live for the presigned URL in seconds (default: 900 = 15 minutes)
	TTL int64 `query:"ttl" validate:"omitempty,min=60,max=86400"`
}

// PresignedURL represents the response containing a presigned URL for accessing media.
type PresignedURL struct {
	PresignedURL string `json:"presigned_url"`
	TTL          int64  `json:"ttl"`
}
