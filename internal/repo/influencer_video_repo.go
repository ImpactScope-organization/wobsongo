package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type influencerVideoRepo struct {
	q    *db.Queries
	pool *pgxpool.Pool
	tx   pgx.Tx
}

// NewInfluencerVideoRepo creates a repository for the influencer_videos table.
func NewInfluencerVideoRepo(q *db.Queries, pool *pgxpool.Pool) data.InfluencerVideoRepoer {
	return &influencerVideoRepo{q: q, pool: pool}
}

func (r *influencerVideoRepo) WithTx(
	ctx context.Context,
	fn func(data.InfluencerVideoRepoer) error,
) error {
	if r.tx != nil {
		return fn(r)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	repoWithTx := &influencerVideoRepo{q: r.q.WithTx(tx), pool: r.pool, tx: tx}
	if err := fn(repoWithTx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UpsertVideo inserts a new video, or refreshes its counters if it was
// already known. isNew reports whether the row was newly inserted.
func (r *influencerVideoRepo) UpsertVideo(
	ctx context.Context,
	v *model.InfluencerVideo,
) (bool, error) {
	row, err := r.q.UpsertInfluencerVideo(ctx, db.UpsertInfluencerVideoParams{
		InfluencerID:   v.InfluencerID,
		TiktokVideoID:  v.TikTokVideoID,
		VideoUrl:       v.VideoURL,
		Caption:        v.Caption,
		LikeCount:      v.LikeCount,
		CommentCount:   v.CommentCount,
		ShareCount:     v.ShareCount,
		PlayCount:      v.PlayCount,
		CollectCount:   v.CollectCount,
		Hashtags:       v.Hashtags,
		VideoCreatedAt: v.VideoCreatedAt,
	})
	if err != nil {
		return false, fmt.Errorf(
			"failed to upsert influencer video via sqlc: %w",
			mapPostgresError(err),
		)
	}

	v.ID = row.ID
	v.CreatedAt = row.CreatedAt
	v.UpdatedAt = row.UpdatedAt
	return row.IsNew, nil
}

// ListByInfluencerID returns every tracked video for an influencer, oldest first.
func (r *influencerVideoRepo) ListByInfluencerID(
	ctx context.Context,
	influencerID uuid.UUID,
) ([]*model.InfluencerVideo, error) {
	rows, err := r.q.ListInfluencerVideosByInfluencerID(ctx, influencerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list influencer videos: %w", mapPostgresError(err))
	}

	videos := make([]*model.InfluencerVideo, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		videos = append(videos, &model.InfluencerVideo{
			ID:             row.ID,
			InfluencerID:   row.InfluencerID,
			TikTokVideoID:  row.TiktokVideoID,
			VideoURL:       row.VideoUrl,
			Caption:        row.Caption,
			LikeCount:      row.LikeCount,
			CommentCount:   row.CommentCount,
			ShareCount:     row.ShareCount,
			PlayCount:      row.PlayCount,
			CollectCount:   row.CollectCount,
			Hashtags:       row.Hashtags,
			VideoCreatedAt: row.VideoCreatedAt,
			CreatedAt:      row.CreatedAt,
			UpdatedAt:      row.UpdatedAt,
		})
	}
	return videos, nil
}
