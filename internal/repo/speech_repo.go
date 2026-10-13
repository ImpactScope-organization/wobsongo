package repo

import (
	"context"
	"fmt"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

// speechRepo implements data.SpeechRepoer.
type speechRepo struct {
	riverClientFn func() *river.Client[pgx.Tx]
}

// NewSpeechRepo creates a new repository for speech-synthesis jobs.
func NewSpeechRepo(riverClientFn func() *river.Client[pgx.Tx]) data.SpeechRepoer {
	return &speechRepo{
		riverClientFn: riverClientFn,
	}
}

// EnqueueSpeech adds a speech job to the River queue.
func (r *speechRepo) EnqueueSpeech(ctx context.Context, payload queue.SpeechJob) error {
	_, err := r.riverClientFn().Insert(ctx, payload, nil)
	if err != nil {
		return fmt.Errorf("failed to insert speech job into river queue: %w", err)
	}
	return nil
}
