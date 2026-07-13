package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AtomicKnowledgeRepo is a Postgres-backed implementation of data.AtomicKnowledgeRepoer.
type AtomicKnowledgeRepo struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

// Ensure AtomicKnowledgeRepo implements data.AtomicKnowledgeRepoer.
var _ data.AtomicKnowledgeRepoer = (*AtomicKnowledgeRepo)(nil)

// NewAtomicKnowledgeRepo creates a new Postgres-backed atomic knowledge repository.
// q is accepted externally (not built internally from pool) so callers
// (including tests) can supply a tx-scoped *db.Queries.
func NewAtomicKnowledgeRepo(q *db.Queries, pool *pgxpool.Pool) data.AtomicKnowledgeRepoer {
	return &AtomicKnowledgeRepo{q: q, pool: pool}
}

// CreateBatch inserts multiple fully-formed knowledge facts in a single COPY operation.
func (r *AtomicKnowledgeRepo) CreateBatch(ctx context.Context, knowledge []model.AtomicKnowledge) error {
	if len(knowledge) == 0 {
		return nil
	}

	params := make([]db.CreateAtomicKnowledgeBatchParams, len(knowledge))
	for i := range knowledge {
		params[i] = toCreateAtomicKnowledgeBatchParams(&knowledge[i])
	}

	if _, err := r.q.CreateAtomicKnowledgeBatch(ctx, params); err != nil {
		return mapPostgresError(err)
	}
	return nil
}

// MarkChunkKnowledgeExtracted records that extraction has run for the given
// chunk, even if it produced zero facts. Operates on document_chunks — see
// data.AtomicKnowledgeRepoer's doc comment for why this repo owns it rather
// than depending on DocumentChunkRepoer: sqlc generates every query onto the
// same shared *db.Queries regardless of which .sql file defines it.
func (r *AtomicKnowledgeRepo) MarkChunkKnowledgeExtracted(ctx context.Context, chunkID uuid.UUID) error {
	if err := r.q.MarkChunkKnowledgeExtracted(ctx, chunkID); err != nil {
		return mapPostgresError(err)
	}
	return nil
}

// ListNeedingEmbedding retrieves facts for a document that don't have an
// embedding yet, ordered by CreatedAt.
func (r *AtomicKnowledgeRepo) ListNeedingEmbedding(
	ctx context.Context,
	documentID uuid.UUID,
) ([]model.AtomicKnowledge, error) {
	rows, err := r.q.ListKnowledgeNeedingEmbedding(ctx, documentID)
	if err != nil {
		return nil, mapPostgresError(err)
	}

	facts := make([]model.AtomicKnowledge, 0, len(rows))
	for i := range rows {
		facts = append(facts, *toModelAtomicKnowledge(&rows[i]))
	}
	return facts, nil
}

// UpdateEmbedding persists the embedding vector for a single fact.
func (r *AtomicKnowledgeRepo) UpdateEmbedding(
	ctx context.Context,
	id uuid.UUID,
	embedding []float32,
) error {
	if err := r.q.UpdateAtomicKnowledgeEmbedding(ctx, db.UpdateAtomicKnowledgeEmbeddingParams{
		ID:        id,
		Embedding: toPgvector(embedding),
		UpdatedAt: time.Now(),
	}); err != nil {
		return mapPostgresError(err)
	}
	return nil
}

// WithTx executes fn within a Postgres transaction, giving it a
// transaction-scoped repo so CreateBatch and MarkChunkKnowledgeExtracted
// commit atomically.
func (r *AtomicKnowledgeRepo) WithTx(ctx context.Context, fn func(data.AtomicKnowledgeRepoer) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("atomic knowledge repo: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txRepo := &AtomicKnowledgeRepo{
		q:    r.q.WithTx(tx),
		pool: r.pool,
	}

	if err := fn(txRepo); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// toModelAtomicKnowledge maps a sqlc-generated db.AtomicKnowledge row to model.AtomicKnowledge.
func toModelAtomicKnowledge(k *db.AtomicKnowledge) *model.AtomicKnowledge {
	return &model.AtomicKnowledge{
		ID:                 k.ID,
		CreatedAt:          k.CreatedAt,
		UpdatedAt:          k.UpdatedAt,
		DocumentID:         k.DocumentID,
		DocumentChunkID:    k.DocumentChunkID,
		TruthTier:          model.TruthTier(k.TruthTier),
		Topics:             k.Topics,
		Subject:            k.Subject,
		Predicate:          k.Predicate,
		Object:             k.Object,
		Note:               k.Note,
		Embedding:          fromPgvector(k.Embedding),
		MarkedAsInvalid:    k.MarkedAsInvalid,
		MarkedAsIrrelevant: k.MarkedAsIrrelevant,
	}
}

// toCreateAtomicKnowledgeBatchParams maps a model.AtomicKnowledge to sqlc's batch-insert params.
func toCreateAtomicKnowledgeBatchParams(k *model.AtomicKnowledge) db.CreateAtomicKnowledgeBatchParams {
	return db.CreateAtomicKnowledgeBatchParams{
		ID:                 k.ID,
		CreatedAt:          k.CreatedAt,
		UpdatedAt:          k.UpdatedAt,
		DocumentID:         k.DocumentID,
		DocumentChunkID:    k.DocumentChunkID,
		TruthTier:          toInt32(int(k.TruthTier)),
		Topics:             k.Topics,
		Subject:            k.Subject,
		Predicate:          k.Predicate,
		Object:             k.Object,
		Note:               k.Note,
		MarkedAsInvalid:    k.MarkedAsInvalid,
		MarkedAsIrrelevant: k.MarkedAsIrrelevant,
	}
}
