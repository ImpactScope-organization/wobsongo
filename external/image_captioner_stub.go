package external

import (
	"context"
	"errors"

	"github.com/impactscope-organization/wobsongo/internal/data"
)

// ErrImageCaptioningNotImplemented is returned by NoOpImageCaptioner —
// wired in as a placeholder until a real (open-weight-model-backed)
// data.ImageCaptioner implementation is built.
var ErrImageCaptioningNotImplemented = errors.New("image captioning not yet implemented")

// NoOpImageCaptioner is a placeholder data.ImageCaptioner that always fails
// clearly, so CaptionImageChunksWorker's plumbing can be built and wired end
// to end before a real captioning provider exists.
type NoOpImageCaptioner struct{}

// Ensure NoOpImageCaptioner implements data.ImageCaptioner.
var _ data.ImageCaptioner = NoOpImageCaptioner{}

// Caption always returns ErrImageCaptioningNotImplemented.
func (NoOpImageCaptioner) Caption(context.Context, []byte, string) (string, error) {
	return "", ErrImageCaptioningNotImplemented
}
