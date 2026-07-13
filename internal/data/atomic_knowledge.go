package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// AtomicKnowledgeRepoer defines the interface for interacting with atomic
// knowledge data storage. Deliberately not CrudableWithTx and doesn't embed
// queue.JobEnqueuer — knowledge facts are never listed/updated generically,
// and this repo doesn't chain to a further job (fact embedding is a separate,
// not-yet-built step).
type AtomicKnowledgeRepoer interface {
	// CreateBatch inserts multiple fully-formed knowledge facts in a single operation.
	CreateBatch(ctx context.Context, knowledge []model.AtomicKnowledge) error

	// MarkChunkKnowledgeExtracted records that extraction has run for the
	// given chunk, even if it produced zero facts.
	MarkChunkKnowledgeExtracted(ctx context.Context, chunkID uuid.UUID) error

	TxAware[AtomicKnowledgeRepoer]
}
