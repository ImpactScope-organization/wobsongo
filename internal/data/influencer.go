package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// InfluencerRepoer defines the data operations for the influencers table.
type InfluencerRepoer interface {
	// UpsertInfluencer inserts a new influencer, or updates the mutable
	// profile fields of an existing one keyed by (platform, username).
	UpsertInfluencer(ctx context.Context, influencer *model.Influencer) error

	// GetByUsername returns the influencer matching platform+username.
	GetByUsername(ctx context.Context, platform, username string) (*model.Influencer, error)

	// ListAll returns all tracked influencers.
	ListAll(ctx context.Context) ([]*model.Influencer, error)

	// UpdateAggregates persists the upload-cadence numbers computed from an
	// influencer's tracked videos after a monitor run.
	UpdateAggregates(
		ctx context.Context,
		id uuid.UUID,
		firstVideoAt, lastVideoAt *time.Time,
		avgUploadIntervalHours *float64,
		trackedVideoCount int64,
	) error

	// GetByID returns the influencer with the given ID.
	GetByID(ctx context.Context, id uuid.UUID) (*model.Influencer, error)

	// ListDue returns active influencers whose next check is due.
	ListDue(ctx context.Context, limit int) ([]*model.Influencer, error)

	// SetNextCheckAt schedules the next monitor check of an influencer.
	SetNextCheckAt(ctx context.Context, id uuid.UUID, at time.Time) error

	// WithTx executes a function within a database transaction.
	TxAware[InfluencerRepoer]
}
