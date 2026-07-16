package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// BotClient handles HTTP communication with the external bot service.
// It manages the base URL, authentication via Pre-Shared Key.
type BotClient struct {
	baseURL    string
	psk        string
	httpClient *http.Client
}

// NewBotClient creates and returns a new instance of BotClient.
// It initializes the underlying HTTP client with a default timeout of 10 seconds
// to prevent indefinite hanging on network requests.
func NewBotClient(baseURL, psk string) *BotClient {
	return &BotClient{
		baseURL:    baseURL,
		psk:        psk,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// extractDoneCallback represents the JSON payload structure sent to the bot service
// to report the completion status of an extraction job.
type extractDoneCallback struct {
	JobID  string `json:"jobId"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// NotifyExtractDone sends a POST request to the bot service callback endpoint
// (/callback/extract-done) to notify it that an extraction job has finished.
// It includes the job ID, its final status, and an optional error message.
// Returns an error if the request fails to build, execute, or if the server
// returns a status code other than 204 No Content.
func (c *BotClient) NotifyExtractDone(ctx context.Context, jobID, status, errMsg string) error {
	body, err := json.Marshal(extractDoneCallback{JobID: jobID, Status: status, Error: errMsg})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/callback/extract-done",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	// Authenticate the request using the Pre-Shared Key.
	req.Header.Set("Authorization", "PSK "+c.psk)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("bot callback returned status %d", resp.StatusCode)
	}
	return nil
}
