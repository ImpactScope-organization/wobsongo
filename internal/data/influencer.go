package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/dto"
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

	// Paginate returns a filtered page of influencers, newest first unless the
	// filter asks to sort by posts.
	Paginate(
		ctx context.Context,
		q SupportsPagination,
		filter InfluencerListFilter,
	) (*dto.PaginationResults[model.Influencer], error)

	// Create inserts a new influencer and fills the stored fields. It returns
	// ErrConflict when the username already exists.
	Create(ctx context.Context, influencer *model.Influencer) error

	// SetActive turns monitoring of an influencer on or off. Turning it on
	// again resets the empty-check counter and schedules a check right away.
	// Influencers are never deleted, because videos reference them.
	SetActive(ctx context.Context, id uuid.UUID, active bool) (*model.Influencer, error)

	// ListDue returns active influencers whose next check is due.
	ListDue(ctx context.Context, limit int) ([]*model.Influencer, error)

	// SetNextCheckAt schedules the next monitor check of an influencer.
	SetNextCheckAt(ctx context.Context, id uuid.UUID, at time.Time) error

	// RecordEmptyCheck counts one more check that found no usable video,
	// schedules the next check at nextCheckAt, and deactivates the influencer
	// once maxEmptyChecks in a row is reached. It reports whether the
	// influencer is still active. All of this happens in one statement.
	RecordEmptyCheck(
		ctx context.Context,
		id uuid.UUID,
		maxEmptyChecks int,
		nextCheckAt time.Time,
	) (stillActive bool, err error)

	// PaginateTopCreators returns the influencers that posted since
	// filter.Since, the most posts first, then the most followers.
	PaginateTopCreators(
		ctx context.Context,
		q SupportsPagination,
		filter TopCreatorFilter,
	) (*dto.PaginationResults[model.TopCreator], error)

	// WithTx executes a function within a database transaction.
	TxAware[InfluencerRepoer]
}
