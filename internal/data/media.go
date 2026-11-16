package data

import (
	"context"

	"github.com/impactscope-organization/wobsongo/internal/dto"
)

// MediaExtractor defines the contract for external media extraction services.
// Any external service (like Apify) must implement this interface to be used by the system.
type MediaExtractor interface {
	// TriggerAudioExtraction initiates an asynchronous media extraction process.
	TriggerAudioExtraction(ctx context.Context, req dto.ExtractionRequest) error
}

// VideoMediaResolver finds a downloadable media URL for a video page URL.
type VideoMediaResolver interface {
	// ResolveMediaURL returns a downloadable media URL, or ErrMediaUnavailable
	// when the platform has none for the video.
	ResolveMediaURL(ctx context.Context, videoURL string) (string, error)
}
