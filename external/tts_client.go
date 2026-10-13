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
	"time"
)

const (
	// ttsHTTPTimeout bounds a single synthesis call.
	ttsHTTPTimeout = 10 * time.Minute

	// ttsMaxAudioBytes caps how much audio we read back from the endpoint.
	ttsMaxAudioBytes = 64 << 20

	// ttsErrorSnippetBytes caps how much of an error response body is quoted.
	ttsErrorSnippetBytes = 512
)

// TTSClient calls the Modal MeloTTS web endpoint.
type TTSClient struct {
	endpoint   string
	httpClient *http.Client
}

// NewTTSClient creates a TTSClient. An empty endpoint is allowed; Synthesize
// then returns an error instead of calling out.
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
	if c.endpoint == "" {
		return nil, errors.New("MODAL_TTS_ENDPOINT is not set")
	}

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

	audio, err := io.ReadAll(io.LimitReader(resp.Body, ttsMaxAudioBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read tts audio: %w", err)
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
