package data

import (
	"context"

	"github.com/impactscope-organization/wobsongo/internal/model"
)

// VideoRepoer defines the data operations for video table.
type VideoRepoer interface {
	CreateVideos(ctx context.Context, video *model.Video) error
}
