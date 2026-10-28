package queue

import "github.com/riverqueue/river"

type ClaimCheckJob struct {
	ExtractionID string `json:"extractionId"`
	Text         string `json:"text"`
	JID          string `json:"jid,omitempty"`
	MessageID    string `json:"messageId,omitempty"`
}

func (ClaimCheckJob) Kind() string { return string(JobTypeClaimCheck) }

func (ClaimCheckJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: QueueMediaProcessing,
	}
}
