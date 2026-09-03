package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// InfluencerVideoRepoer defines the data operations for the
// influencer_videos table.
type InfluencerVideoRepoer interface {
	// UpsertVideo inserts a new video, or refreshes its counters if it was already known.
	UpsertVideo(ctx context.Context, video *model.InfluencerVideo) (isNew bool, err error)

	// ListByInfluencerID returns every tracked video for an influencer.
	ListByInfluencerID(
		ctx context.Context,
		influencerID uuid.UUID,
	) ([]*model.InfluencerVideo, error)

	// WithTx executes a function within a database transaction.
	TxAware[InfluencerVideoRepoer]
}
