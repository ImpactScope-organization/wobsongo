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
