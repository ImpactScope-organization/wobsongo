package queue

import (
	"time"

	"github.com/riverqueue/river"
)

// DefaultClaimBackfillMaxWorkers is how many backfill claim checks run at once.
const DefaultClaimBackfillMaxWorkers = 2

type ClaimCheckJob struct {
	ExtractionID string    `json:"extractionId"`
	Text         string    `json:"text"`
	JID          string    `json:"jid,omitempty"`
	MessageID    string    `json:"messageId,omitempty"`
	Platform     string    `json:"platform,omitempty"`
	Format       string    `json:"format,omitempty"`
	OccurredAt   time.Time `json:"occurredAt,omitempty"`
}

func (ClaimCheckJob) Kind() string { return string(JobTypeClaimCheck) }

func (ClaimCheckJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: QueueMediaProcessing,
	}
}
