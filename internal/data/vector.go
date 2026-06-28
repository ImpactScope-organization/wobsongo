package data

import "context"

// Embedder provides a standard interface to compute embedding vector
// of a given text.
type Embedder interface {
	// Embed computes the embedding vector of text.
	Embed(ctx context.Context, text string) ([]float64, error)
}
