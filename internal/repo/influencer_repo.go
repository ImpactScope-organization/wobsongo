package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/dto"
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
		IsActive:           row.IsActive,
		EmptyCheckCount:    int(row.EmptyCheckCount),
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
	if row.NextCheckAt.Valid {
		t := row.NextCheckAt.Time
		m.NextCheckAt = &t
	}
	return m
}

// GetByID returns the influencer with the given ID.
func (r *influencerRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Influencer, error) {
	row, err := r.q.GetInfluencerByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get influencer by id: %w", mapPostgresError(err))
	}
	return dbInfluencerToModel(&row), nil
}

// ListDue returns up to limit active influencers whose next check is due.
func (r *influencerRepo) ListDue(ctx context.Context, limit int) ([]*model.Influencer, error) {
	rows, err := r.q.ListDueInfluencers(ctx, int32(limit))
	if err != nil {
		return nil, fmt.Errorf("failed to list due influencers: %w", mapPostgresError(err))
	}

	influencers := make([]*model.Influencer, 0, len(rows))
	for i := range rows {
		influencers = append(influencers, dbInfluencerToModel(&rows[i]))
	}
	return influencers, nil
}

// SetNextCheckAt schedules the next monitor check of an influencer.
func (r *influencerRepo) SetNextCheckAt(ctx context.Context, id uuid.UUID, at time.Time) error {
	err := r.q.SetInfluencerNextCheckAt(ctx, db.SetInfluencerNextCheckAtParams{
		NextCheckAt: pgtype.Timestamptz{Time: at, Valid: true},
		ID:          id,
	})
	if err != nil {
		return fmt.Errorf("failed to set influencer next check: %w", mapPostgresError(err))
	}
	return nil
}

// RecordEmptyCheck counts one more empty check and deactivates the influencer
// once maxEmptyChecks in a row is reached.
func (r *influencerRepo) RecordEmptyCheck(
	ctx context.Context,
	id uuid.UUID,
	maxEmptyChecks int,
	nextCheckAt time.Time,
) (bool, error) {
	row, err := r.q.RecordInfluencerEmptyCheck(ctx, db.RecordInfluencerEmptyCheckParams{
		MaxEmptyChecks: int32(maxEmptyChecks),
		NextCheckAt:    nextCheckAt,
		ID:             id,
	})
	if err != nil {
		return false, fmt.Errorf("failed to record empty check: %w", mapPostgresError(err))
	}
	return row.IsActive, nil
}

// Paginate returns a filtered page of influencers, newest first.
func (r *influencerRepo) Paginate(
	ctx context.Context,
	q data.SupportsPagination,
	filter data.InfluencerListFilter,
) (*dto.PaginationResults[model.Influencer], error) {
	limit, offset := q.Limit(), q.Offset()
	isActive, source, search := filterBool(filter.IsActive),
		filterText(filter.Source), filterText(filter.Search)

	rows, err := r.q.PaginateInfluencers(ctx, db.PaginateInfluencersParams{
		IsActive:   isActive,
		Source:     source,
		Search:     search,
		SortBy:     filter.Sort,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list influencers: %w", mapPostgresError(err))
	}

	total, err := r.q.CountInfluencers(ctx, db.CountInfluencersParams{
		IsActive: isActive,
		Source:   source,
		Search:   search,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count influencers: %w", mapPostgresError(err))
	}

	items := make([]model.Influencer, 0, len(rows))
	for i := range rows {
		items = append(items, *dbInfluencerToModel(&rows[i]))
	}

	return &dto.PaginationResults[model.Influencer]{
		Page:       pageOf(limit, offset),
		PerPage:    int(limit),
		TotalItems: int(total),
		Items:      items,
	}, nil
}

// Create inserts a new influencer. No row comes back from the query when the
// username already exists, which is reported as a conflict.
func (r *influencerRepo) Create(ctx context.Context, m *model.Influencer) error {
	row, err := r.q.CreateInfluencer(ctx, db.CreateInfluencerParams{
		Platform:   m.Platform,
		Username:   m.Username,
		ProfileUrl: m.ProfileURL,
		Source:     string(m.Source),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: influencer %s already exists", data.ErrConflict, m.Username)
	}
	if err != nil {
		return fmt.Errorf("failed to create influencer: %w", mapPostgresError(err))
	}

	*m = *dbInfluencerToModel(&row)
	return nil
}

// SetActive turns monitoring of an influencer on or off.
func (r *influencerRepo) SetActive(
	ctx context.Context,
	id uuid.UUID,
	active bool,
) (*model.Influencer, error) {
	row, err := r.q.SetInfluencerActive(ctx, db.SetInfluencerActiveParams{
		IsActive: active,
		ID:       id,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set influencer active: %w", mapPostgresError(err))
	}
	return dbInfluencerToModel(&row), nil
}

// PaginateTopCreators returns the influencers that posted since filter.Since,
// the most posts first.
func (r *influencerRepo) PaginateTopCreators(
	ctx context.Context,
	q data.SupportsPagination,
	filter data.TopCreatorFilter,
) (*dto.PaginationResults[model.TopCreator], error) {
	limit, offset := q.Limit(), q.Offset()
	isActive, search := filterBool(filter.IsActive), filterText(filter.Search)

	rows, err := r.q.PaginateTopCreators(ctx, db.PaginateTopCreatorsParams{
		Since:      filter.Since,
		IsActive:   isActive,
		Search:     search,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list top creators: %w", mapPostgresError(err))
	}

	total, err := r.q.CountTopCreators(ctx, db.CountTopCreatorsParams{
		Since:    filter.Since,
		IsActive: isActive,
		Search:   search,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count top creators: %w", mapPostgresError(err))
	}

	items := make([]model.TopCreator, 0, len(rows))
	for i := range rows {
		items = append(items, dbTopCreatorToModel(&rows[i]))
	}

	return &dto.PaginationResults[model.TopCreator]{
		Page:       pageOf(limit, offset),
		PerPage:    int(limit),
		TotalItems: int(total),
		Items:      items,
	}, nil
}

func dbTopCreatorToModel(row *db.PaginateTopCreatorsRow) model.TopCreator {
	return model.TopCreator{
		ID:             row.ID,
		Platform:       row.Platform,
		Username:       row.Username,
		Nickname:       row.Nickname,
		ProfileURL:     row.ProfileUrl,
		AvatarURL:      row.AvatarUrl,
		FollowersCount: row.FollowersCount,
		PostsCount:     row.PostsCount,
	}
}
