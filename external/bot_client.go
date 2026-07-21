package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// BotClient handles HTTP communication with the external bot service.
// It manages the base URL, authentication via Pre-Shared Key.
type BotClient struct {
	baseURL     string
	callbackPSK string
	controlPSK  string
	httpClient  *http.Client
}

type BotStatus struct {
	Status string `json:"status"`
	QR     string `json:"qr,omitempty"`
}

// NewBotClient creates and returns a new instance of BotClient.
// It initializes the underlying HTTP client with a default timeout of 10 seconds
// to prevent indefinite hanging on network requests.
func NewBotClient(baseURL, callbackPSK, controlPSK string) *BotClient {
	return &BotClient{
		baseURL:     baseURL,
		callbackPSK: callbackPSK,
		controlPSK:  controlPSK,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// ExtractCallbackData contains the result returned directly in the
// callback. Used for RAG responses (not persisted), while
// TranscriptionWorker leaves it nil and retrieves the cached transcript
// via /api/extract.
type ExtractCallbackData struct {
	Transcript string `json:"transcript,omitempty"`
	Answer     string `json:"answer,omitempty"`
}

// extractDoneCallback represents the JSON payload structure sent to the bot service
// to report the completion status of an extraction job.
type extractDoneCallback struct {
	JobID  string               `json:"jobId"`
	Status string               `json:"status"`
	Error  string               `json:"error,omitempty"`
	Data   *ExtractCallbackData `json:"data,omitempty"`
}

// NotifyExtractDone sends a POST request to the bot service callback endpoint
// (/callback/extract-done) to notify it that an extraction job has finished.
// It includes the job ID, its final status, and an optional error message.
// Returns an error if the request fails to build, execute, or if the server
// returns a status code other than 204 No Content.
func (c *BotClient) NotifyExtractDone(
	ctx context.Context,
	jobID, status, errMsg string,
	data *ExtractCallbackData,
) error {
	body, err := json.Marshal(extractDoneCallback{
		JobID:  jobID,
		Status: status,
		Error:  errMsg,
		Data:   data,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal callback payload: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/callback/extract-done",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return fmt.Errorf("failed to create callback request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Authenticate the request using the Pre-Shared Key.
	req.Header.Set("Authorization", "PSK "+c.callbackPSK)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute callback request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("bot callback returned status %d", resp.StatusCode)
	}
	return nil
}

// doControlRequest is an internal helper function that constructs and executes an HTTP request
// to the bot control API endpoint.

func (c *BotClient) doControlRequest(
	ctx context.Context,
	method, path string,
	body any,
) (*BotStatus, error) {
	var reqBody bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&reqBody).Encode(body); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, &reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "PSK "+c.controlPSK)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute control request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"bot control API returned status %d: %s",
			resp.StatusCode,
			string(respBody),
		)
	}

	var status BotStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}
	return &status, nil
}

// Start sends a request to the bot's control API to initialize and start
// the WhatsApp socket connection.
func (c *BotClient) Start(ctx context.Context) (*BotStatus, error) {
	return c.doControlRequest(ctx, http.MethodPost, "/bot/start", nil)
}

// Stop sends a request to the bot's control API to gracefully disconnect
// the WhatsApp socket. The purgeData parameter determines whether the bot's
// local session and authentication data should be deleted upon stopping.
func (c *BotClient) Stop(ctx context.Context, purgeData bool) (*BotStatus, error) {
	return c.doControlRequest(
		ctx,
		http.MethodPost,
		"/bot/stop",
		map[string]bool{"purgeData": purgeData},
	)
}

// Status sends a request to the bot's control API to retrieve its current
// operational state and active QR code (if applicable), without altering its lifecycle.
func (c *BotClient) Status(ctx context.Context) (*BotStatus, error) {
	return c.doControlRequest(ctx, http.MethodGet, "/bot/status", nil)
}
