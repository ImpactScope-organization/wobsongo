package queue

import (
	"context"
)

// JobEnqueuer defines the interface for enqueuing background jobs.
// Repos that need to enqueue jobs (optionally within a DB transaction)
// implement this interface. See data.TxAware for transaction support.
type JobEnqueuer interface {
	// Enqueue adds a job with the specified payload to the queue.
	Enqueue(ctx context.Context, payload BackgroundJob) error
}

// TxAware is a generic interface for types that support transactions.
// T is the repo type that supports transactions.
type TxAware[T any] interface {
	// WithTx executes the given function within a transaction.
	WithTx(ctx context.Context, fn func(T) error) error
}