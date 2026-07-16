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

	// ListChunksNeedingEmbedding retrieves chunks for a document that have
	// text but no embedding yet, ordered by SequenceNumber. Used by
	// EmbedChunksWorker; the filter also makes retries idempotent — a chunk
	// already embedded is never returned again.
	ListChunksNeedingEmbedding(
		ctx context.Context,
		documentID uuid.UUID,
	) ([]model.DocumentChunk, error)

	// ListChunksNeedingKnowledgeExtraction retrieves chunks for a document
	// that have text but haven't had atomic-knowledge extraction run yet,
	// ordered by SequenceNumber. Used by ExtractKnowledgeWorker; the filter
	// also makes retries idempotent.
	ListChunksNeedingKnowledgeExtraction(
		ctx context.Context,
		documentID uuid.UUID,
	) ([]model.DocumentChunk, error)

	// CreateBatch inserts multiple fully-formed chunks in a single operation.
	CreateBatch(ctx context.Context, chunks []model.DocumentChunk) error

	// Update saves an existing document chunk.
	Update(ctx context.Context, chunk *model.DocumentChunk) error

	// ShouldBeStored decides whether a chunk carries enough information/context
	// to be worth persisting. doc is threaded through alongside chunk so the
	// decision can be informed by document-level context, not just the chunk
	// in isolation, even though today's implementation only looks at
	// chunk.LayoutType.
	ShouldBeStored(ctx context.Context, doc model.Document, chunk model.DocumentChunk) (bool, error)

	// SearchByEmbedding returns the limit chunks whose embedding is closest
	// (cosine distance) to queryVector, ordered nearest-first. One of the
	// hybrid-search retrieval methods; see service.RAGService.
	SearchByEmbedding(
		ctx context.Context,
		queryVector []float32,
		limit int,
	) ([]ScoredResult[model.DocumentChunk], error)

	// SearchByFullText returns the limit chunks whose text best matches query
	// via Postgres full-text search (ts_rank_cd), ordered best-first. One of
	// the hybrid-search retrieval methods; see service.RAGService.
	SearchByFullText(
		ctx context.Context,
		query string,
		limit int,
	) ([]ScoredResult[model.DocumentChunk], error)

	TxAware[DocumentChunkRepoer]
	queue.JobEnqueuer
}
