package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/impactscope-organization/wobsongo/internal/dto"
)

// TikTokScraperClient runs the Apify TikTok Scraper synchronously
// and returns the scraped items directly.
type TikTokScraperClient struct {
	apiToken   string
	actorID    string
	httpClient *http.Client
}

// NewTikTokScraperClient constructs a TikTokScraperClient.
func NewTikTokScraperClient(apiToken, actorID string) *TikTokScraperClient {
	return &TikTokScraperClient{
		apiToken: apiToken,
		actorID:  actorID,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// FetchProfileVideos scrapes the most recent videos of a single TikTok
// username, newest first.
func (c *TikTokScraperClient) FetchProfileVideos(
	ctx context.Context,
	username string,
	limit int,
) ([]dto.TikTokScraperItem, error) {
	input := map[string]any{
		"profiles":       []string{username},
		"resultsPerPage": limit,
		"profileSorting": "latest",
	}
	return c.run(ctx, input)
}

// SearchByKeyword scrapes videos matching a free-text keyword, used to
// discover influencers talking about the same topic who aren't tracked yet.
func (c *TikTokScraperClient) SearchByKeyword(
	ctx context.Context,
	keyword string,
	limit int,
) ([]dto.TikTokScraperItem, error) {
	input := map[string]any{
		"searchQueries":  []string{keyword},
		"resultsPerPage": limit,
		"searchSection":  "/video",
	}
	return c.run(ctx, input)
}

func (c *TikTokScraperClient) run(
	ctx context.Context,
	input map[string]any,
) ([]dto.TikTokScraperItem, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal apify input: %w", err)
	}

	apiURL := fmt.Sprintf(
		"https://api.apify.com/v2/acts/%s/run-sync-get-dataset-items?token=%s",
		c.actorID,
		c.apiToken,
	)

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		apiURL,
		bytes.NewBuffer(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call apify API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read apify response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf(
			"apify API returned error status: %d. Body: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var items []dto.TikTokScraperItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("failed to decode apify dataset items: %w", err)
	}

	return items, nil
}
