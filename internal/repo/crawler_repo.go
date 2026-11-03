package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/jackc/pgx/v5/pgtype"
)

type crawlerRepo struct {
	q *db.Queries
}

// NewCrawlerRepo creates a repository for the crawler settings, discovery
// keywords and crawl runs.
func NewCrawlerRepo(q *db.Queries) data.CrawlerRepoer {
	return &crawlerRepo{q: q}
}

// GetSetting returns the crawler configuration.
func (r *crawlerRepo) GetSetting(ctx context.Context) (*model.AdminSetting, error) {
	row, err := r.q.GetCrawlerSetting(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get crawler setting: %w", mapPostgresError(err))
	}

	setting := &model.AdminSetting{
		DiscoveryIntervalHours: int(row.DiscoveryIntervalHours),
		MaxResultsKeyword:      int(row.MaxResultsKeyword),
		MaxResultsInfluencer:   int(row.MaxResultsInfluencer),
		MaxCostPerRun:          row.MaxCostPerRun,
		UpdatedAt:              row.UpdatedAt,
	}
	if row.UpdatedBy.Valid {
		id := uuid.UUID(row.UpdatedBy.Bytes)
		setting.UpdatedBy = &id
	}
	return setting, nil
}

// ListActiveKeywords returns every enabled discovery keyword.
func (r *crawlerRepo) ListActiveKeywords(ctx context.Context) ([]*model.DiscoveryKeyword, error) {
	rows, err := r.q.ListActiveDiscoveryKeywords(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list discovery keywords: %w", mapPostgresError(err))
	}

	keywords := make([]*model.DiscoveryKeyword, 0, len(rows))
	for i := range rows {
		row := &rows[i]
		keyword := &model.DiscoveryKeyword{
			ID:        row.ID,
			Keyword:   row.Keyword,
			Language:  model.Language(row.Language),
			IsActive:  row.IsActive,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
		if row.LastRunAt.Valid {
			t := row.LastRunAt.Time
			keyword.LastRunAt = &t
		}
		keywords = append(keywords, keyword)
	}
	return keywords, nil
}

// MarkKeywordRun stamps last_run_at on a keyword.
func (r *crawlerRepo) MarkKeywordRun(ctx context.Context, id uuid.UUID) error {
	if err := r.q.MarkDiscoveryKeywordRun(ctx, id); err != nil {
		return fmt.Errorf("failed to mark keyword run: %w", mapPostgresError(err))
	}
	return nil
}

// GetLastRunStartedAt returns when the latest run of a kind started. It
// returns data.ErrNotFound when there has been no run of that kind yet.
func (r *crawlerRepo) GetLastRunStartedAt(
	ctx context.Context,
	kind model.CrawlKind,
) (*time.Time, error) {
	startedAt, err := r.q.GetLastCrawlRunStartedAt(ctx, string(kind))
	if err != nil {
		return nil, fmt.Errorf("failed to get last crawl run: %w", mapPostgresError(err))
	}
	return &startedAt, nil
}

// FailStaleRuns marks runs that stayed 'running' for too long as failed.
func (r *crawlerRepo) FailStaleRuns(
	ctx context.Context,
	kind model.CrawlKind,
	olderThan time.Duration,
) error {
	err := r.q.FailStaleCrawlRuns(ctx, db.FailStaleCrawlRunsParams{
		Kind:         string(kind),
		StaleSeconds: olderThan.Seconds(),
	})
	if err != nil {
		return fmt.Errorf("failed to fail stale crawl runs: %w", mapPostgresError(err))
	}
	return nil
}

// CreateRun inserts a run with status running. It returns data.ErrConflict
// when a run of the same kind is already running.
func (r *crawlerRepo) CreateRun(ctx context.Context, run *model.CrawlRun) error {
	var triggeredBy pgtype.UUID
	if run.TriggeredBy != nil {
		triggeredBy = pgtype.UUID{Bytes: *run.TriggeredBy, Valid: true}
	}

	params := run.Params
	if len(params) == 0 {
		params = []byte("{}")
	}

	row, err := r.q.CreateCrawlRun(ctx, db.CreateCrawlRunParams{
		Kind:        string(run.Kind),
		Trigger:     string(run.Trigger),
		TriggeredBy: triggeredBy,
		Params:      params,
	})
	if err != nil {
		return fmt.Errorf("failed to create crawl run: %w", mapPostgresError(err))
	}

	run.ID = row.ID
	run.StartedAt = row.StartedAt
	run.Status = model.CrawlStatusRunning
	return nil
}

// FinishRun saves the final status, counts and error of a run.
func (r *crawlerRepo) FinishRun(ctx context.Context, run *model.CrawlRun) error {
	var errText pgtype.Text
	if run.Error != nil {
		errText = pgtype.Text{String: *run.Error, Valid: true}
	}

	err := r.q.FinishCrawlRun(ctx, db.FinishCrawlRunParams{
		ID:         run.ID,
		Status:     string(run.Status),
		VideoFound: int32(run.VideoFound),
		VideoNew:   int32(run.VideoNew),
		Error:      errText,
	})
	if err != nil {
		return fmt.Errorf("failed to finish crawl run: %w", mapPostgresError(err))
	}
	return nil
}

// UpdateSetting saves the crawler configuration.
func (r *crawlerRepo) UpdateSetting(ctx context.Context, setting *model.AdminSetting) error {
	var updatedBy pgtype.UUID
	if setting.UpdatedBy != nil {
		updatedBy = pgtype.UUID{Bytes: *setting.UpdatedBy, Valid: true}
	}

	updatedAt, err := r.q.UpdateCrawlerSetting(ctx, db.UpdateCrawlerSettingParams{
		DiscoveryIntervalHours: int32(setting.DiscoveryIntervalHours),
		MaxResultsKeyword:      int32(setting.MaxResultsKeyword),
		MaxResultsInfluencer:   int32(setting.MaxResultsInfluencer),
		MaxCostPerRun:          setting.MaxCostPerRun,
		UpdatedBy:              updatedBy,
	})
	if err != nil {
		return fmt.Errorf("failed to update crawler setting: %w", mapPostgresError(err))
	}

	setting.UpdatedAt = updatedAt
	return nil
}
