package dto

// ExtractionRequest represents the payload required to trigger a media extraction.
// We strictly define the types as strings and enforce validation rules.
type ExtractionRequest struct {
	// TargetURL is the link to the video (e.g., TikTok or Instagram reels).
	TargetURL string `json:"target_url" validate:"required,url"`

	// WebhookURL is the endpoint Apify will call once the process is complete.
	WebhookURL string `json:"webhook_url" validate:"required,url"`
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
