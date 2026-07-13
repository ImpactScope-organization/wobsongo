package data

import "context"

// Embedder provides a standard interface to compute embedding vectors for a
// batch of texts. Provider-agnostic by design, same as ImageCaptioner; see
// external.EmbeddingClient for the concrete implementation (a generic
// OpenAI-compatible embeddings API). Batched (not one-text-at-a-time) so a
// caller embedding many chunks for one document can do it in a single call.
type Embedder interface {
	// Embed computes the embedding vector for each of texts, in the same
	// order. []float32 matches pgvector.Vector's underlying representation,
	// so no lossy float64/float32 conversion is needed at the repo boundary.
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}
