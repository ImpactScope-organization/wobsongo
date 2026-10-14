package external

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/impactscope-organization/wobsongo/internal/queue"
)

const (
	// bytesPerMB is the number of bytes in a megabyte (1024 * 1024).
	bytesPerMB = 1024 * 1024

	// ttsHTTPTimeout is the maximum time allowed for a synthesis request,
	// including cold starts.
	ttsHTTPTimeout = queue.SpeechJobTTL

	// ttsMaxAudioBytes caps how much audio we read back from the endpoint
	// (16 MB). WhatsApp rejects audio files larger than 16 MB when they are
	// sent as regular audio, so anything bigger could not be delivered.
	ttsMaxAudioBytes = 16 * bytesPerMB

	// ttsErrorSnippetBytes is the maximum error response snippet size (512 bytes).
	// enough to see the reason in the logs without flooding them
	// when the endpoint returns a large HTML error page.
	ttsErrorSnippetBytes = 512
)

// TTSClient calls the Modal MeloTTS web endpoint.
type TTSClient struct {
	endpoint   string
	httpClient *http.Client
}

// NewTTSClient creates a TTSClient for the given endpoint.
func NewTTSClient(endpoint string) *TTSClient {
	return &TTSClient{
		endpoint:   endpoint,
		httpClient: &http.Client{Timeout: ttsHTTPTimeout},
	}
}

type ttsRequest struct {
	Text     string `json:"text"`
	Language string `json:"language"`
	Format   string `json:"format"`
}

// Synthesize implements data.SpeechSynthesizer. It returns OGG/Opus audio.
func (c *TTSClient) Synthesize(ctx context.Context, text, language string) ([]byte, error) {
	body, err := json.Marshal(ttsRequest{
		Text:     text,
		Language: strings.ToUpper(language),
		Format:   "ogg",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tts request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build tts request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tts request failed: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("[TTSClient] failed to close response body: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, ttsStatusError(resp)
	}

	audio, err := io.ReadAll(io.LimitReader(resp.Body, ttsMaxAudioBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read tts audio: %w", err)
	}
	if len(audio) > ttsMaxAudioBytes {
		return nil, fmt.Errorf("tts audio exceeds the %d MB limit", ttsMaxAudioBytes/bytesPerMB)
	}
	if len(audio) == 0 {
		return nil, errors.New("tts endpoint returned empty audio")
	}
	return audio, nil
}

// ttsStatusError builds the error for a non-200 response.
func ttsStatusError(resp *http.Response) error {
	snippet, err := io.ReadAll(io.LimitReader(resp.Body, ttsErrorSnippetBytes))
	if err != nil {
		return fmt.Errorf(
			"tts endpoint returned %d (and its body could not be read: %w)",
			resp.StatusCode, err,
		)
	}
	return fmt.Errorf("tts endpoint returned %d: %s", resp.StatusCode, string(snippet))
}
