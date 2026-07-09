package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

// DocumentChunkRepo is a Postgres-backed implementation of data.DocumentChunkRepoer.
type DocumentChunkRepo struct {
	q           *db.Queries
	pool        *pgxpool.Pool
	riverClient *river.Client[pgx.Tx]
	tx          pgx.Tx // set only on the tx-scoped instance WithTx constructs; nil otherwise
}

// Ensure DocumentChunkRepo implements data.DocumentChunkRepoer.
var _ data.DocumentChunkRepoer = (*DocumentChunkRepo)(nil)

// NewDocumentChunkRepo creates a new Postgres-backed document chunk repository.
// q is accepted externally (not built internally from pool) so callers
// (including tests) can supply a tx-scoped *db.Queries. riverClient may be
// nil if Enqueue is never called (see cmd/server.go for why that's currently
// the case).
func NewDocumentChunkRepo(
	q *db.Queries,
	pool *pgxpool.Pool,
	riverClient *river.Client[pgx.Tx],
) data.DocumentChunkRepoer {
	return &DocumentChunkRepo{q: q, pool: pool, riverClient: riverClient}
}

// GetByID retrieves a document chunk by its ID.
func (r *DocumentChunkRepo) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*model.DocumentChunk, error) {
	chunk, err := r.q.GetDocumentChunkByID(ctx, id)
	if err != nil {
		return nil, mapPostgresError(err)
	}
	return toModelDocumentChunk(&chunk), nil
}

// ListByDocumentID retrieves all chunks for a document, ordered by SequenceNumber.
func (r *DocumentChunkRepo) ListByDocumentID(
	ctx context.Context,
	documentID uuid.UUID,
) ([]model.DocumentChunk, error) {
	rows, err := r.q.ListDocumentChunksByDocumentID(ctx, documentID)
	if err != nil {
		return nil, mapPostgresError(err)
	}

	chunks := make([]model.DocumentChunk, 0, len(rows))
	for i := range rows {
		chunks = append(chunks, *toModelDocumentChunk(&rows[i]))
	}
	return chunks, nil
}

// CreateBatch inserts multiple fully-formed chunks in a single COPY operation.
func (r *DocumentChunkRepo) CreateBatch(ctx context.Context, chunks []model.DocumentChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	params := make([]db.CreateDocumentChunksBatchParams, len(chunks))
	for i := range chunks {
		params[i] = toCreateDocumentChunksBatchParams(&chunks[i])
	}

	if _, err := r.q.CreateDocumentChunksBatch(ctx, params); err != nil {
		return mapPostgresError(err)
	}
	return nil
}

// Update saves an existing document chunk.
func (r *DocumentChunkRepo) Update(ctx context.Context, chunk *model.DocumentChunk) error {
	updated, err := r.q.UpdateDocumentChunk(ctx, db.UpdateDocumentChunkParams{
		ID:              chunk.ID,
		UpdatedAt:       chunk.UpdatedAt,
		Topics:          chunk.Topics,
		FactualityScore: chunk.FactualityScore,
		Text:            chunk.Text,
		Chapter:         chunk.Chapter,
		AssetUrl:        chunk.AssetURL,
	})
	if err != nil {
		return mapPostgresError(err)
	}
	*chunk = *toModelDocumentChunk(&updated)
	return nil
}

// ShouldBeStored decides whether a chunk carries enough information/context
// to be worth persisting. A pass-through today (always true); real filtering
// logic (heuristics and/or NLP/LLM-based scoring) lands later.
//
//nolint:gocritic // chunk is passed by value: fixed by the data.DocumentChunkRepoer interface signature.
func (r *DocumentChunkRepo) ShouldBeStored(_ context.Context, _ model.DocumentChunk) (bool, error) {
	return true, nil
}

// WithTx executes fn within a Postgres transaction, giving it a
// transaction-scoped repo whose Enqueue calls are part of the same
// transaction as any CRUD calls it makes.
func (r *DocumentChunkRepo) WithTx(
	ctx context.Context,
	fn func(data.DocumentChunkRepoer) error,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("document chunk repo: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	txRepo := &DocumentChunkRepo{
		q:           r.q.WithTx(tx),
		pool:        r.pool,
		riverClient: r.riverClient,
		tx:          tx,
	}

	if err := fn(txRepo); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Enqueue adds a job to the queue, using the open transaction when called
// from within WithTx so the job insert is atomic with any CRUD writes.
func (r *DocumentChunkRepo) Enqueue(ctx context.Context, payload queue.BackgroundJob) error {
	if r.tx != nil {
		_, err := r.riverClient.InsertTx(ctx, r.tx, payload, nil)
		return err
	}
	_, err := r.riverClient.Insert(ctx, payload, nil)
	return err
}

// toModelDocumentChunk maps a sqlc-generated db.DocumentChunk row to model.DocumentChunk.
func toModelDocumentChunk(d *db.DocumentChunk) *model.DocumentChunk {
	return &model.DocumentChunk{
		ID:              d.ID,
		CreatedAt:       d.CreatedAt,
		UpdatedAt:       d.UpdatedAt,
		DocumentID:      d.DocumentID,
		SequenceNumber:  int(d.SequenceNumber),
		Topics:          d.Topics,
		FactualityScore: d.FactualityScore,
		ParsedChunk: model.ParsedChunk{
			Text:        d.Text,
			Page:        int(d.Page),
			LayoutType:  model.LayoutType(d.LayoutType),
			BoundingBox: toBoundingBox(d.BoundingBox),
			AssetURL:    d.AssetUrl,
		},
	}
}

// toCreateDocumentChunksBatchParams maps a model.DocumentChunk to sqlc's batch-insert params.
func toCreateDocumentChunksBatchParams(c *model.DocumentChunk) db.CreateDocumentChunksBatchParams {
	return db.CreateDocumentChunksBatchParams{
		ID:              c.ID,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
		DocumentID:      c.DocumentID,
		SequenceNumber:  toInt32(c.SequenceNumber),
		Topics:          c.Topics,
		FactualityScore: c.FactualityScore,
		Text:            c.Text,
		Page:            toInt32(c.Page),
		Chapter:         c.Chapter,
		LayoutType:      string(c.LayoutType),
		BoundingBox:     c.BoundingBox[:],
		AssetUrl:        c.AssetURL,
	}
}

// toBoundingBox converts a Postgres float8[] column into model.BoundingBox,
// defaulting to the zero value if the array isn't exactly 4 elements.
func toBoundingBox(v []float64) model.BoundingBox {
	var bbox model.BoundingBox
	copy(bbox[:], v)
	return bbox
}
