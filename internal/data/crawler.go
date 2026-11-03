package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// CrawlerRepoer defines the data operations for the crawler settings,
// discovery keywords and crawl run history.
type CrawlerRepoer interface {
	// GetSetting returns the crawler configuration, creating the default
	// row on first use.
	GetSetting(ctx context.Context) (*model.AdminSetting, error)

	// ListActiveKeywords returns every enabled discovery keyword.
	ListActiveKeywords(ctx context.Context) ([]*model.DiscoveryKeyword, error)

	// MarkKeywordRun stamps last_run_at on a keyword.
	MarkKeywordRun(ctx context.Context, id uuid.UUID) error

	// GetLastRunStartedAt returns when the latest run of a kind started. It
	// returns ErrNotFound if there has been no run of that kind yet.
	GetLastRunStartedAt(ctx context.Context, kind model.CrawlKind) (*time.Time, error)

	// FailStaleRuns marks runs of a kind that are still 'running' after
	// olderThan as failed, so a crashed run cannot block new ones forever.
	FailStaleRuns(ctx context.Context, kind model.CrawlKind, olderThan time.Duration) error

	// CreateRun inserts a run with status running and fills its ID and
	// StartedAt. It returns ErrConflict when a run of the same kind is
	// already running.
	CreateRun(ctx context.Context, run *model.CrawlRun) error

	// FinishRun saves the final status, counts and error of a run.
	FinishRun(ctx context.Context, run *model.CrawlRun) error

	// UpdateSetting saves the crawler configuration, creating the row if it
	// does not exist yet, and fills setting.UpdatedAt.
	UpdateSetting(ctx context.Context, setting *model.AdminSetting) error
}

// DiscoveryRescheduler defines an interface for updating the schedule
// of the periodic discovery job.
type DiscoveryRescheduler interface {
	Reschedule(intervalHours int, runOnStart bool)
}
