package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type influencerRepo struct {
	q    *db.Queries
	pool *pgxpool.Pool
	tx   pgx.Tx
}

// NewInfluencerRepo creates a repository for the influencers table.
func NewInfluencerRepo(q *db.Queries, pool *pgxpool.Pool) data.InfluencerRepoer {
	return &influencerRepo{q: q, pool: pool}
}

func (r *influencerRepo) WithTx(ctx context.Context, fn func(data.InfluencerRepoer) error) error {
	if r.tx != nil {
		return fn(r)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	repoWithTx := &influencerRepo{q: r.q.WithTx(tx), pool: r.pool, tx: tx}
	if err := fn(repoWithTx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UpsertInfluencer inserts a new influencer or refreshes the mutable
// profile fields of an existing one, keyed by (platform, username).
func (r *influencerRepo) UpsertInfluencer(ctx context.Context, m *model.Influencer) error {
	var keyword, sourceVideo pgtype.Text
	if m.DiscoveredViaKeyword != "" {
		keyword = pgtype.Text{String: m.DiscoveredViaKeyword, Valid: true}
	}
	if m.DiscoveredFromVideoURL != "" {
		sourceVideo = pgtype.Text{String: m.DiscoveredFromVideoURL, Valid: true}
	}

	row, err := r.q.UpsertInfluencer(ctx, db.UpsertInfluencerParams{
		Platform:               m.Platform,
		Username:               m.Username,
		ProfileUrl:             m.ProfileURL,
		Nickname:               m.Nickname,
		Bio:                    m.Bio,
		AvatarUrl:              m.AvatarURL,
		Verified:               m.Verified,
		FollowersCount:         m.FollowersCount,
		FollowingCount:         m.FollowingCount,
		HeartsCount:            m.HeartsCount,
		ReportedVideoCount:     m.ReportedVideoCount,
		Source:                 string(m.Source),
		DiscoveredViaKeyword:   keyword,
		DiscoveredFromVideoUrl: sourceVideo,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert influencer via sqlc: %w", mapPostgresError(err))
	}

	m.ID = row.ID
	m.CreatedAt = row.CreatedAt
	m.UpdatedAt = row.UpdatedAt
	return nil
}

// GetByUsername returns the influencer matching platform+username.
func (r *influencerRepo) GetByUsername(
	ctx context.Context,
	platform, username string,
) (*model.Influencer, error) {
	row, err := r.q.GetInfluencerByUsername(ctx, db.GetInfluencerByUsernameParams{
		Platform: platform,
		Username: username,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get influencer by username: %w", mapPostgresError(err))
	}
	return dbInfluencerToModel(&row), nil
}

// ListAll returns every tracked influencer (seed + discovered).
func (r *influencerRepo) ListAll(ctx context.Context) ([]*model.Influencer, error) {
	rows, err := r.q.ListInfluencers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list influencers: %w", mapPostgresError(err))
	}

	influencers := make([]*model.Influencer, 0, len(rows))
	for i := range rows {
		influencers = append(influencers, dbInfluencerToModel(&rows[i]))
	}
	return influencers, nil
}

// UpdateAggregates persists the upload-cadence numbers computed from an
// influencer's tracked videos after a monitor run.
func (r *influencerRepo) UpdateAggregates(
	ctx context.Context,
	id uuid.UUID,
	firstVideoAt, lastVideoAt *time.Time,
	avgUploadIntervalHours *float64,
	trackedVideoCount int64,
) error {
	params := db.UpdateInfluencerAggregatesParams{
		ID:                id,
		TrackedVideoCount: trackedVideoCount,
	}
	if firstVideoAt != nil {
		params.FirstVideoAt = pgtype.Timestamptz{Time: *firstVideoAt, Valid: true}
	}
	if lastVideoAt != nil {
		params.LastVideoAt = pgtype.Timestamptz{Time: *lastVideoAt, Valid: true}
	}
	if avgUploadIntervalHours != nil {
		params.AvgUploadIntervalHours = pgtype.Float8{Float64: *avgUploadIntervalHours, Valid: true}
	}

	if err := r.q.UpdateInfluencerAggregates(ctx, params); err != nil {
		return fmt.Errorf("failed to update influencer aggregates: %w", mapPostgresError(err))
	}
	return nil
}

func dbInfluencerToModel(row *db.Influencer) *model.Influencer {
	if row == nil {
		return nil
	}
	m := &model.Influencer{
		ID:                 row.ID,
		Platform:           row.Platform,
		Username:           row.Username,
		ProfileURL:         row.ProfileUrl,
		Nickname:           row.Nickname,
		Bio:                row.Bio,
		AvatarURL:          row.AvatarUrl,
		Verified:           row.Verified,
		FollowersCount:     row.FollowersCount,
		FollowingCount:     row.FollowingCount,
		HeartsCount:        row.HeartsCount,
		ReportedVideoCount: row.ReportedVideoCount,
		TrackedVideoCount:  row.TrackedVideoCount,
		Source:             model.InfluencerSource(row.Source),
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
	if row.DiscoveredViaKeyword.Valid {
		m.DiscoveredViaKeyword = row.DiscoveredViaKeyword.String
	}
	if row.DiscoveredFromVideoUrl.Valid {
		m.DiscoveredFromVideoURL = row.DiscoveredFromVideoUrl.String
	}
	if row.FirstVideoAt.Valid {
		t := row.FirstVideoAt.Time
		m.FirstVideoAt = &t
	}
	if row.LastVideoAt.Valid {
		t := row.LastVideoAt.Time
		m.LastVideoAt = &t
	}
	if row.AvgUploadIntervalHours.Valid {
		v := row.AvgUploadIntervalHours.Float64
		m.AvgUploadIntervalHours = &v
	}
	if row.LastCheckedAt.Valid {
		t := row.LastCheckedAt.Time
		m.LastCheckedAt = &t
	}
	return m
}
