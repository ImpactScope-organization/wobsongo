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

	// EnqueueTranscriptionJob adds a new transcription job to the River queue.
	EnqueueTranscriptionJob(ctx context.Context, payload queue.TranscriptionJobDTO) error

	// UpdateVideoTranscription updates row in the video table with the transcription result.
	UpdateVideoTranscription(ctx context.Context, text pgtype.Text, id uuid.UUID) error

	// WithTx executes a function within a database transaction.
	TxAware[VideoRepoer]
}
