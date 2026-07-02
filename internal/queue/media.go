package queue

// ExtractMediaDTO is the river job kind for extracting media from a given source.
type ExtractMediaDTO struct {
	TargetURL  string `json:"target_url"`
	WebhookURL string `json:"webhook_url"`
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (ExtractMediaDTO) Kind() string {
	return string(JobTypeExtractMedia)
}
