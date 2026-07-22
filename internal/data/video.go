package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/jackc/pgx/v5/pgtype"
)

// VideoRepoer defines the data operations for video table.
type VideoRepoer interface {
	// CreateVideos inserts a new video record into the database.
	CreateVideos(ctx context.Context, video *model.Video) error

	// GetByVideoURL returns the video record matching the given URL, or
	// nil (with no error) if none exists yet — used by ApifyService to
	// check cache before triggering a new Apify run.
	GetByVideoURL(ctx context.Context, videoURL string) (*model.Video, error)

	// EnqueueTranscriptionJob adds a new transcription job to the River queue.
	EnqueueTranscriptionJob(ctx context.Context, payload queue.TranscriptionJob) error

	// UpdateVideoTranscription updates row in the video table with the transcription result.
	UpdateVideoTranscription(ctx context.Context, text pgtype.Text, id uuid.UUID) error

	// EnqueueClaimCheckJob adds a new claim-check job to the River queue.
	EnqueueClaimCheckJob(ctx context.Context, payload queue.ClaimCheckJob) error

	// WithTx executes a function within a database transaction.
	TxAware[VideoRepoer]
}
