package repo

import (
	"context"
	"fmt"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type videoRepo struct {
	q           *db.Queries
	pool        *pgxpool.Pool
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
	}

	// Call the sqlc-generated method to insert the video record
	row, err := r.q.CreateVideos(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to upsert video via sqlc: %w", err)
	}

	// Update the original Video struct with the returned values from the database
	v.ID = row.ID
	v.CreatedAt = row.CreatedAt
	v.UpdatedAt = row.UpdatedAt

	return nil
}
