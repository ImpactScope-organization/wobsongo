package external

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/impactscope-organization/wobsongo/internal/dto"
)

// Dispatcher is implementation from MediaExtractor interface that handles communication with Apify API.
type Dispatcher struct {
	apiToken      string
	tiktokActorID string
	igActorID     string
	httpClient    *http.Client
}

// NewDispatcher is a constructor for Dispatcher.
// It initializes the struct with the provided API token and Actor IDs for TikTok and Instagram.
func NewDispatcher(apiToken, tiktokActorID, igActorID string) *Dispatcher {
	return &Dispatcher{
		apiToken:      apiToken,
		tiktokActorID: tiktokActorID,
		igActorID:     igActorID,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// tiktokApifyInput defines the strict payload for clockworks/tiktok-scraper.
type tiktokApifyInput struct {
	PostURLs                      []string `json:"postURLs"`
	ShouldDownloadVideos          bool     `json:"shouldDownloadVideos"`
	ShouldDownloadCovers          bool     `json:"shouldDownloadCovers"`
	ShouldDownloadSubtitles       bool     `json:"shouldDownloadSubtitles"`
	ShouldDownloadSlideshowImages bool     `json:"shouldDownloadSlideshowImages"`
}

// igApifyInput defines the strict payload for the legacy Instagram scraper.
type igApifyInput struct {
	Link string `json:"link"`
}

// apifyWebhook is the internal strict DTO for the Apify webhook configuration.
type apifyWebhook struct {
	EventTypes []string `json:"eventTypes"`
	RequestURL string   `json:"requestUrl"`
}

// TriggerAudioExtraction executes an HTTP call to the Apify API.
func (d *Dispatcher) TriggerAudioExtraction(ctx context.Context, req dto.ExtractionRequest) error {
	var actorID string
	var payloadBytes []byte
	var err error

	// Routing based on the platform URL provided in the request.
	switch {
	case strings.Contains(req.TargetURL, "tiktok.com"):
		actorID = d.tiktokActorID

		// Setup input specifically for TikTok Scraper
		input := tiktokApifyInput{
			PostURLs:                      []string{req.TargetURL},
			ShouldDownloadVideos:          true,
			ShouldDownloadCovers:          false,
			ShouldDownloadSubtitles:       false,
			ShouldDownloadSlideshowImages: false,
		}

		payloadBytes, err = json.Marshal(input)
		if err != nil {
			return fmt.Errorf("failed to marshal tiktok apify payload: %w", err)
		}

	case strings.Contains(req.TargetURL, "instagram.com"):
		actorID = d.igActorID

		// Setup input specifically for Instagram Scraper
		input := igApifyInput{
			Link: req.TargetURL,
		}

		payloadBytes, err = json.Marshal(input)
		if err != nil {
			return fmt.Errorf("failed to marshal ig apify payload: %w", err)
		}

	default:
		return errors.New("unsupported platform: only TikTok and Instagram are supported")
	}

	// 1. Prepare the webhook configuration for Apify API
	webhooks := []apifyWebhook{
		{
			EventTypes: []string{
				"ACTOR.RUN.SUCCEEDED",
				"ACTOR.RUN.FAILED",
				"ACTOR.RUN.TIMED_OUT",
				"ACTOR.RUN.ABORTED",
			},
			RequestURL: req.WebhookURL,
		},
	}

	// 2. Marshal the webhook configuration to JSON
	webhookBytes, err := json.Marshal(webhooks)
	if err != nil {
		return fmt.Errorf("failed to marshal webhooks: %w", err)
	}

	// 3. Encode to base64 and then URL escape the webhook JSON for Apify API
	base64Webhooks := base64.StdEncoding.EncodeToString(webhookBytes)
	encodedWebhooks := url.QueryEscape(base64Webhooks)

	apiURL := fmt.Sprintf(
		"https://api.apify.com/v2/acts/%s/runs?webhooks=%s",
		actorID,
		encodedWebhooks,
	)

	// Prepare the HTTP request with context, method, URL, and payload.
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		apiURL,
		bytes.NewBuffer(payloadBytes),
	)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+d.apiToken)

	// Execute the HTTP request and handle the response.
	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call Apify API: %w", err)
	}
	defer resp.Body.Close()

	// Validate the response status code from Apify API.
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		// Read the error message from the response body
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"apify API returned error status: %d. Body: %s",
			resp.StatusCode,
			string(bodyBytes),
		)
	}

	return nil
}
