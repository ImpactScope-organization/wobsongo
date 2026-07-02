package data

import (
	"context"

	"github.com/impactscope-organization/wobsongo/internal/queue"
)

// ApifyRepoer defines the data operations for Apify related tasks.
type ApifyRepoer interface {
	EnqueueExtraction(ctx context.Context, args queue.ExtractMediaDTO) error
}
