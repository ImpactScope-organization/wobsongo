package queue

import "github.com/riverqueue/river"

type ClaimCheckJob struct {
	ExtractionID string `json:"extractionId"`
	Text         string `json:"text"`
	Jid          string `json:"jid,omitempty"`
}

func (ClaimCheckJob) Kind() string { return string(JobTypeClaimCheck) }

func (ClaimCheckJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: QueueMediaProcessing,
	}
}
