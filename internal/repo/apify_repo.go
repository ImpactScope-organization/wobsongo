package repo

import (
	"context"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// apifyRepo implements data.ApifyRepoer.
type apifyRepo struct {
	riverClientFn func() *river.Client[pgx.Tx]
}

// NewApifyRepo creates a new repository for Apify tasks.
func NewApifyRepo(riverClientFn func() *river.Client[pgx.Tx]) data.ApifyRepoer {
	return &apifyRepo{
		riverClientFn: riverClientFn,
	}
}

// EnqueueExtraction enqueues a media extraction job.
func (r *apifyRepo) EnqueueExtraction(ctx context.Context, args queue.ExtractMediaDTO) error {
	_, err := r.riverClientFn().Insert(ctx, args, nil)
	if err != nil {
		return err
	}
	return nil
}
