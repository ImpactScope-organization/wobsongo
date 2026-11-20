package external

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
)

const (
	// scraperListTimeout bounds a profile or keyword scrape.
	scraperListTimeout = 120 * time.Second

	// scraperMediaTimeout bounds a run that downloads the video.
	scraperMediaTimeout = 5 * time.Minute
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
		apiToken:   apiToken,
		actorID:    actorID,
		httpClient: &http.Client{},
	}
}

// errNoMediaItem means the run finished but gave no media for the video. It
// is not ErrMediaUnavailable: an empty result is also what a blocked run
// looks like, so it must stay retryable.
var errNoMediaItem = errors.New("apify returned no media")

// permanentScraperCodes are the actor error codes that mean the post cannot
// be downloaded, however often it is tried.
var permanentScraperCodes = map[string]bool{
	"POST_NOT_FOUND_OR_PRIVATE": true,
	"NOT_FOUND":                 true,
	"POST_SENSITIVE":            true,
}

var permanentScraperMessages = []string{"not found or private", "does not exist"}

// isPermanentScraperError reports whether an error item says the video itself
// is gone, private or not downloadable.
func isPermanentScraperError(item *dto.TikTokScraperItem) bool {
	if permanentScraperCodes[item.ErrorCode] {
		return true
	}
	msg := strings.ToLower(item.Error)
	return slices.ContainsFunc(permanentScraperMessages, func(m string) bool {
		return strings.Contains(msg, m)
	})
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
	return c.run(ctx, input, scraperListTimeout)
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
	return c.run(ctx, input, scraperListTimeout)
}

func (c *TikTokScraperClient) run(
	ctx context.Context,
	input map[string]any,
	timeout time.Duration,
) ([]dto.TikTokScraperItem, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	payload, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal apify input: %w", err)
	}

	apiURL := fmt.Sprintf(
		"https://api.apify.com/v2/acts/%s/run-sync-get-dataset-items",
		c.actorID,
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
	httpReq.Header.Set("Authorization", "Bearer "+c.apiToken)
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

// ResolveMediaURL implements data.VideoMediaResolver. It runs the actor for
// one video with video download on and returns the first media URL.
func (c *TikTokScraperClient) ResolveMediaURL(
	ctx context.Context,
	videoURL string,
) (string, error) {
	items, err := c.run(ctx, map[string]any{
		"postURLs":                      []string{videoURL},
		"shouldDownloadVideos":          true,
		"shouldDownloadCovers":          false,
		"shouldDownloadSubtitles":       false,
		"shouldDownloadSlideshowImages": false,
	}, scraperMediaTimeout)
	if err != nil {
		return "", fmt.Errorf("failed to run apify for media url: %w", err)
	}

	for i := range items {
		if len(items[i].MediaURLs) > 0 {
			return items[i].MediaURLs[0], nil
		}
	}

	for i := range items {
		item := &items[i]
		if item.Error == "" {
			continue
		}
		if isPermanentScraperError(item) {
			return "", fmt.Errorf("%w: %s", data.ErrMediaUnavailable, item.Error)
		}
		return "", fmt.Errorf("apify could not scrape %s: %s", videoURL, item.Error)
	}
	return "", fmt.Errorf("%w for %s", errNoMediaItem, videoURL)
}
