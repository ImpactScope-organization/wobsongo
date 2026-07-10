package data

import "context"

// ImageCaptioner generates a textual description of an image, so it can be
// stored as a document chunk's Text and be embeddable/searchable like any
// other chunk. Provider-agnostic by design — no concrete implementation is
// wired in yet; see external.NoOpImageCaptioner for the current stand-in.
type ImageCaptioner interface {
	// Caption returns a detailed description of imageBytes (contentType
	// e.g. "image/png").
	Caption(ctx context.Context, imageBytes []byte, contentType string) (string, error)
}
