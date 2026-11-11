package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// AdminListFilter holds the optional filters shared by the admin list endpoints.
type AdminListFilter struct {
	// IsActive keeps only active (true) or inactive (false) rows when set.
	IsActive *bool

	// Search keeps rows whose name contains this text, ignoring case.
	Search string
}

// KeywordRepoer defines the admin operations on discovery keywords.
type KeywordRepoer interface {
	Paginate(
		ctx context.Context,
		q SupportsPagination,
		filter AdminListFilter,
	) (*dto.PaginationResults[model.DiscoveryKeyword], error)

	GetByID(ctx context.Context, id uuid.UUID) (*model.DiscoveryKeyword, error)

	// Create inserts the keyword and fills the stored fields. It returns
	// ErrConflict when the same keyword and language already exist.
	Create(ctx context.Context, keyword *model.DiscoveryKeyword) error

	// Update replaces keyword, language and is_active of keyword.ID and fills
	// the stored fields. It returns ErrNotFound or ErrConflict.
	Update(ctx context.Context, keyword *model.DiscoveryKeyword) error

	// Delete removes a keyword. It returns ErrNotFound when it does not exist.
	Delete(ctx context.Context, id uuid.UUID) error
}

// InfluencerListFilter adds the influencer-only filters.
type InfluencerListFilter struct {
	AdminListFilter

	// Source keeps only influencers from this source ("seed" or "discovered").
	Source string

	// Sort orders the list: "posts" puts the influencers with the most
	// tracked posts first (then the most followers). Anything else, newest first.
	Sort string
}

// TopicRepoer defines the admin operations on topics. Topics are never
// deleted because claims and narratives reference them.
type TopicRepoer interface {
	TxAware[TopicRepoer]

	Paginate(
		ctx context.Context,
		q SupportsPagination,
		filter AdminListFilter,
	) (*dto.PaginationResults[model.Topic], error)

	GetByID(ctx context.Context, id uuid.UUID) (*model.Topic, error)

	// Create inserts the topic and fills the stored fields. It returns
	// ErrConflict when the name is taken.
	Create(ctx context.Context, topic *model.Topic) error

	// Update replaces name, description and is_active of topic.ID and fills
	// the stored fields. It returns ErrNotFound when the topic does not exist
	// or when this would deactivate the last active topic, and ErrConflict
	// when the name is taken.
	Update(ctx context.Context, topic *model.Topic) error

	// LockActiveGuard serializes updates that may deactivate a topic.
	// It must be called inside WithTx.
	LockActiveGuard(ctx context.Context) error
}
