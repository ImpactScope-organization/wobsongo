package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
)

// DocumentChunkRepoer defines the interface for interacting with document
// chunk data storage. Deliberately not CrudableWithTx — chunks are never
// listed generically (paginated across every document); the only real list
// operation is "chunks for document X, in reading order."
type DocumentChunkRepoer interface {
	// GetByID retrieves a document chunk by its ID.
	GetByID(ctx context.Context, id uuid.UUID) (*model.DocumentChunk, error)

	// ListByDocumentID retrieves all chunks for a document, ordered by SequenceNumber.
	ListByDocumentID(ctx context.Context, documentID uuid.UUID) ([]model.DocumentChunk, error)

	// CreateBatch inserts multiple fully-formed chunks in a single operation.
	CreateBatch(ctx context.Context, chunks []model.DocumentChunk) error

	// Update saves an existing document chunk.
	Update(ctx context.Context, chunk *model.DocumentChunk) error

	// ShouldBeStored decides whether a chunk carries enough information/context
	// to be worth persisting. A pass-through today (always true); real
	// filtering logic (heuristics and/or NLP/LLM-based scoring) lands later.
	ShouldBeStored(ctx context.Context, chunk model.DocumentChunk) (bool, error)

	TxAware[DocumentChunkRepoer]
	queue.JobEnqueuer
}
