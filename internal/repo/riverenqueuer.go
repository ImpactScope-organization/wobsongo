package repo

import (
	"context"
	"fmt"

	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

// RiverJobEnqueuer is a standalone queue.JobEnqueuer that inserts River jobs
// using a regular (non-transactional) database connection from the pool.
// It is used to enqueue jobs from services that do not hold an open transaction.
type RiverJobEnqueuer struct {
	pool        *pgxpool.Pool
	riverClient *river.Client[pgx.Tx]
}

// Ensure RiverJobEnqueuer implements queue.JobEnqueuer.
var _ queue.JobEnqueuer = (*RiverJobEnqueuer)(nil)

// NewRiverJobEnqueuer creates a new RiverJobEnqueuer.
func NewRiverJobEnqueuer(pool *pgxpool.Pool, riverClient *river.Client[pgx.Tx]) *RiverJobEnqueuer {
	return &RiverJobEnqueuer{
		pool:        pool,
		riverClient: riverClient,
	}
}

// Enqueue inserts a River job within a short-lived transaction obtained from the pool.
func (e *RiverJobEnqueuer) Enqueue(ctx context.Context, payload queue.BackgroundJob) error {
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("RiverJobEnqueuer.Enqueue: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := e.riverClient.InsertTx(ctx, tx, payload, nil); err != nil {
		return fmt.Errorf("RiverJobEnqueuer.Enqueue: insert job: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("RiverJobEnqueuer.Enqueue: commit tx: %w", err)
	}

	return nil
}
