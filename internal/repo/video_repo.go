package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type videoRepo struct {
	q           *db.Queries
	pool        *pgxpool.Pool
	tx          pgx.Tx
	riverClient *river.Client[pgx.Tx]
}

func NewVideoRepo(
	q *db.Queries,
	pool *pgxpool.Pool,
	riverClient *river.Client[pgx.Tx],
) data.VideoRepoer {
	return &videoRepo{
		q:           q,
		pool:        pool,
		riverClient: riverClient,
	}
}

func (r *videoRepo) WithTx(
	ctx context.Context,
	fn func(data.VideoRepoer) error,
) error {
	if r.tx != nil {
		return fn(r)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	qtx := r.q.WithTx(tx)

	repoWithTx := &videoRepo{
		q:           qtx,
		pool:        r.pool,
		tx:          tx,
		riverClient: r.riverClient,
	}

	if err := fn(repoWithTx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// CreateVideos inserts a new video record into the database.
func (r *videoRepo) CreateVideos(ctx context.Context, v *model.Video) error {
	caption := pgtype.Text{Valid: false}
	if v.Caption != "" {
		caption = pgtype.Text{String: v.Caption, Valid: true}
	}

	location := pgtype.Text{Valid: false}
	if v.LocationCreated != "" {
		location = pgtype.Text{String: v.LocationCreated, Valid: true}
	}

	videoCreatedAt := pgtype.Timestamptz{
		Time:  v.VideoCreatedAt,
		Valid: !v.VideoCreatedAt.IsZero(),
	}

	// Generate the parameters for the SQL query using sqlc's generated struct
	params := db.CreateVideosParams{
		VideoUrl:         v.VideoURL,
		AuthorUsername:   v.AuthorUsername,
		AuthorProfileUrl: v.AuthorProfileURL,
		Caption:          caption,
		PlayCount:        pgtype.Int8{Int64: v.PlayCount, Valid: true},
		LikeCount:        pgtype.Int8{Int64: v.LikeCount, Valid: true},
		ThumbnailUrl:     v.ThumbnailURL,
		LocationCreated:  location,
		VideoCreatedAt:   videoCreatedAt,
		VideoType:        v.VideoType,
		Hashtags:         v.Hashtags,
	}

	// Call the sqlc-generated method to insert the video record
	row, err := r.q.CreateVideos(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to upsert video via sqlc: %w", mapPostgresError(err))
	}

	// Update the original Video struct with the returned values from the database
	v.ID = row.ID
	v.CreatedAt = row.CreatedAt
	v.UpdatedAt = row.UpdatedAt

	return nil
}

// EnqueueTranscriptionJob adds a new transcription job to the River queue.
func (r *videoRepo) EnqueueTranscriptionJob(
	ctx context.Context,
	payload queue.TranscriptionJob,
) error {
	if r.tx != nil {
		_, err := r.riverClient.InsertTx(ctx, r.tx, payload, nil)
		if err != nil {
			return fmt.Errorf("failed to insert transcription job into river queue: %w", err)
		}

		return nil
	}

	err := r.WithTx(ctx, func(txRepo data.VideoRepoer) error {
		return txRepo.EnqueueTranscriptionJob(ctx, payload)
	})
	if err != nil {
		return fmt.Errorf("failed to execute transcription job with tx: %w", err)
	}

	return nil
}

// UpdateVideoTranscription updates the transcription text for a video by its ID.
func (r *videoRepo) UpdateVideoTranscription(
	ctx context.Context,
	text pgtype.Text,
	id uuid.UUID,
) error {
	err := r.q.UpdateVideoTranscription(ctx, db.UpdateVideoTranscriptionParams{
		TranscriptionText: text,
		ID:                id,
	})
	if err != nil {
		return mapPostgresError(err)
	}

	return nil
}

// GetByVideoURL retrieves a video record from the database.
// If the requested video does not exist (pgx.ErrNoRows), it intentionally
// returns (nil, nil) to represent a normal cache miss rather than an error.
// Any other database execution errors are wrapped and returned.
func (r *videoRepo) GetByVideoURL(ctx context.Context, videoURL string) (*model.Video, error) {
	row, err := r.q.GetVideoByURL(ctx, videoURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, data.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get video by url: %w", mapPostgresError(err))
	}

	// Safely map the nullable transcription text field
	var transcriptionText *string
	if row.TranscriptionText.Valid {
		transcriptionText = &row.TranscriptionText.String
	}

	return &model.Video{
		ID:                row.ID,
		VideoURL:          row.VideoUrl,
		AuthorUsername:    row.AuthorUsername,
		AuthorProfileURL:  row.AuthorProfileUrl,
		Caption:           row.Caption.String,
		PlayCount:         row.PlayCount.Int64,
		LikeCount:         row.LikeCount.Int64,
		ThumbnailURL:      row.ThumbnailUrl,
		LocationCreated:   row.LocationCreated.String,
		VideoCreatedAt:    row.VideoCreatedAt.Time,
		TranscriptionText: transcriptionText,
		VideoType:         row.VideoType,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}, nil
}
