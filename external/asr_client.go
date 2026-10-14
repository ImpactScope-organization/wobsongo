package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// asrHTTPTimeout bounds one transcription request (5 minutes). It is
	// generous because the Modal endpoint can scale to zero, so a cold start
	// (container boot + model load) happens before transcription begins.
	asrHTTPTimeout = 5 * time.Minute

	// asrMaxResponseBytes caps the ASR response we decode (1 MB). A transcript
	// is plain text, so this only protects against a misbehaving endpoint.
	asrMaxResponseBytes = 1 * bytesPerMB

	// asrErrorSnippetBytes caps how much of an error body is quoted (512 bytes).
	asrErrorSnippetBytes = 512
)

// ASRClient calls the Modal ASR web endpoint.
type ASRClient struct {
	endpoint   string
	model      string
	sourceLang string
	httpClient *http.Client
}

// NewASRClient creates an ASRClient.
func NewASRClient(endpoint, model, sourceLang string) *ASRClient {
	return &ASRClient{
		endpoint:   endpoint,
		model:      model,
		sourceLang: sourceLang,
		httpClient: &http.Client{Timeout: asrHTTPTimeout},
	}
}

type asrRequest struct {
	Model       string `json:"model"`
	SourceLang  string `json:"source_lang"`
	AudioURL    string `json:"audio_url,omitempty"`
	AudioFormat string `json:"audio_format"`
}

type asrResponse struct {
	Transcript string `json:"transcript"`
	Error      string `json:"error"`
}

// TranscribeURL implements data.SpeechTranscriber. The ASR service downloads
// the audio itself from audioURL (extracted TikTok videos and voice notes
// stored in S3).
func (c *ASRClient) TranscribeURL(ctx context.Context, audioURL, format string) (string, error) {
	return c.do(ctx, &asrRequest{
		AudioURL:    audioURL,
		AudioFormat: format,
	})
}

// do sends one transcription request. The transcript may be empty when no
// speech was recognized.
func (c *ASRClient) do(ctx context.Context, payload *asrRequest) (string, error) {
	payload.Model = c.model
	payload.SourceLang = c.sourceLang

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal asr request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to build asr request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("asr request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		snippet, readErr := io.ReadAll(io.LimitReader(resp.Body, asrErrorSnippetBytes))
		if readErr != nil {
			return "", fmt.Errorf(
				"asr endpoint returned %d (and its body could not be read: %w)",
				resp.StatusCode, readErr,
			)
		}
		return "", fmt.Errorf("asr endpoint returned %d: %s", resp.StatusCode, string(snippet))
	}

	var out asrResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, asrMaxResponseBytes)).
		Decode(&out); err != nil {
		return "", fmt.Errorf("failed to decode asr response: %w", err)
	}
	if out.Error != "" {
		return "", fmt.Errorf("asr application error: %s", out.Error)
	}
	return strings.TrimSpace(out.Transcript), nil
}
