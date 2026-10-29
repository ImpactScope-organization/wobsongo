package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// NarrativeRepoer defines the data operations of the claim-to-narrative flow.
type NarrativeRepoer interface {
	// WithTx runs fn inside a transaction.
	TxAware[NarrativeRepoer]

	// ListActiveTopics returns every active topic, ordered by name.
	ListActiveTopics(ctx context.Context) ([]model.Topic, error)

	// CreateClaim stores a claim. It returns ErrConflict when the same
	// (message, text) claim already exists.
	CreateClaim(ctx context.Context, claim *model.Claim) (*model.Claim, error)

	// GetClaim returns the claim with the given id, or ErrNotFound.
	GetClaim(ctx context.Context, id uuid.UUID) (*model.Claim, error)

	// EnqueueClusterClaim adds a cluster-claim job to the River queue, inside
	// the current transaction when there is one.
	EnqueueClusterClaim(ctx context.Context, claimID uuid.UUID) error

	// LockTopic takes a transaction-scoped lock on a topic, so claims of one
	// topic are grouped one at a time.
	LockTopic(ctx context.Context, topicID uuid.UUID) error

	// ListNarrativesByTopic returns every narrative of a topic, active or not.
	ListNarrativesByTopic(ctx context.Context, topicID uuid.UUID) ([]model.Narrative, error)

	// GetNarrative returns the narrative with the given id, or ErrNotFound.
	GetNarrative(ctx context.Context, id uuid.UUID) (*model.Narrative, error)

	// PaginateNarratives returns one page of narratives, newest first, with
	// the total number of narratives.
	PaginateNarratives(
		ctx context.Context,
		q SupportsPagination,
	) (*dto.PaginationResults[model.Narrative], error)

	// CreateNarrative stores a new active narrative with the given topic and
	// statement, no harm estimate and the needs_check status.
	CreateNarrative(ctx context.Context, narrative *model.Narrative) (*model.Narrative, error)

	// LinkClaim links a claim to a narrative.
	LinkClaim(ctx context.Context, claimID, narrativeID uuid.UUID) error

	// GetNewestClaim returns the claim of the narrative with the greatest
	// OccurredAt.
	GetNewestClaim(ctx context.Context, narrativeID uuid.UUID) (*model.Claim, error)

	// UpdateNarrativeEstimate sets the harm and status computed from the
	// newest claim and marks the narrative active. A validated harm and an
	// expert_verified status are left untouched.
	UpdateNarrativeEstimate(
		ctx context.Context,
		narrativeID uuid.UUID,
		harm *int,
		status model.NarrativeStatus,
	) error

	// DeactivateStaleNarratives deactivates active narratives without a claim
	// that occurred at or after since, and returns how many it changed.
	DeactivateStaleNarratives(ctx context.Context, since time.Time) (int64, error)
}

// NarrativeStatementGenerator writes the broad statement of a new narrative
// from the first claim that created it.
type NarrativeStatementGenerator interface {
	// Generate returns one broad statement describing the narrative claim
	// belongs to.
	Generate(ctx context.Context, claim string) (string, error)
}
