package repo

import (
	"context"
	"errors"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type speechRepo struct{}

// NewSpeechRepo is a placeholder until the River-backed repo lands in PR 3.
func NewSpeechRepo(_ func() *river.Client[pgx.Tx], _ int) data.SpeechRepoer {
	return &speechRepo{}
}

func (*speechRepo) EnqueueSpeech(context.Context, queue.SpeechJob) error {
	return errors.New("speech queue is not implemented yet")
}
