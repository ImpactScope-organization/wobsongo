package queue

import (
	"context"
)

// TxAware is a generic interface for types that support transactions.
// T is the repo type that supports transactions.
type TxAware[T any] interface {
	// WithTx executes the given function within a transaction.
	WithTx(ctx context.Context, fn func(T) error) error
}

// JobEnqueuer defines the interface for enqueuing jobs with a payload.
type JobEnqueuer interface {
	// Enqueue adds a job with the specified payload to the queue.
	Enqueue(ctx context.Context, payload BackgroundJob) error
}
