package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/impactscope-organization/wobsongo/internal/data"
)

// ModalBGEClient implements data.Embedder against a bespoke Modal-hosted
// sentence-transformers embedding service (see the wobsongo-asr project's
// embedding.py: a modal.fastapi_endpoint mounted directly at the function's
// base URL — POST {baseURL} with {"texts": [...]}, not the OpenAI-compatible
// POST {baseURL}/v1/embeddings with {"model", "input"}). Order of returned
// vectors matches the input order 1:1 — the response carries no index field
// to re-sort by, unlike EmbeddingClient.
type ModalBGEClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Ensure ModalBGEClient implements data.Embedder.
var _ data.Embedder = (*ModalBGEClient)(nil)

// NewModalBGEClient creates a new ModalBGEClient targeting the given base
// URL. apiKey may be empty — this deployment shape currently has no auth.
func NewModalBGEClient(baseURL, apiKey string) *ModalBGEClient {
	return &ModalBGEClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

// modalBGEEmbedRequest is embedding.py's EmbedRequest.
type modalBGEEmbedRequest struct {
	Texts []string `json:"texts"`
}

// modalBGEEmbedResponse is embedding.py's response shape. Error is only
// populated when the handler catches a bad request (e.g. empty texts) —
// still delivered as HTTP 200, so it must be checked explicitly.
type modalBGEEmbedResponse struct {
	Model      string      `json:"model"`
	Embeddings [][]float32 `json:"embeddings"`
	Error      string      `json:"error"`
}

// Embed implements data.Embedder.
func (c *ModalBGEClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	body, err := json.Marshal(modalBGEEmbedRequest{Texts: texts})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embeddings request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call embeddings endpoint: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read embeddings response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"embeddings endpoint returned error status: %d. Body: %s",
			resp.StatusCode,
			string(respBytes),
		)
	}

	var parsed modalBGEEmbedResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embeddings response: %w", err)
	}
	if parsed.Error != "" {
		return nil, fmt.Errorf("embeddings endpoint returned an error: %s", parsed.Error)
	}
	if len(parsed.Embeddings) != len(texts) {
		return nil, fmt.Errorf(
			"embeddings endpoint returned %d vectors for %d inputs",
			len(parsed.Embeddings),
			len(texts),
		)
	}

	return parsed.Embeddings, nil
}
