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

	// WithTx executes a function within a database transaction.
	TxAware[InfluencerRepoer]
}
