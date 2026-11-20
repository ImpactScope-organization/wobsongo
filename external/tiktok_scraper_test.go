package external

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/impactscope-organization/wobsongo/internal/data"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func newScraperTestClient(
	status int,
	body string,
	onRequest func(*http.Request, []byte),
) *TikTokScraperClient {
	c := NewTikTokScraperClient("apify-token", "clockworks~tiktok-scraper")
	c.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if onRequest != nil {
				payload, err := io.ReadAll(r.Body)
				if err != nil {
					return nil, err
				}
				onRequest(r, payload)
			}
			return &http.Response{
				StatusCode: status,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}),
	}
	return c
}

func TestTikTokScraperClient_ResolveMediaURL_Success(t *testing.T) {
	var (
		gotReq     *http.Request
		gotPayload map[string]any
	)
	client := newScraperTestClient(
		http.StatusCreated,
		`[{"id":"1","text":"no media here"},
		  {"id":"2","mediaUrls":["https://api.apify.com/v2/key-value-stores/s/records/v.mp4","https://other"]}]`,
		func(r *http.Request, payload []byte) {
			gotReq = r
			if err := json.Unmarshal(payload, &gotPayload); err != nil {
				t.Errorf("request body is not json: %v", err)
			}
		},
	)

	got, err := client.ResolveMediaURL(t.Context(), "https://www.tiktok.com/@a/video/2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want := "https://api.apify.com/v2/key-value-stores/s/records/v.mp4"; got != want {
		t.Errorf("media url = %q, want the first url of the first item that has one", got)
	}
	if gotReq.Method != http.MethodPost {
		t.Errorf("method = %s, want POST", gotReq.Method)
	}
	if !strings.HasSuffix(
		gotReq.URL.Path,
		"/v2/acts/clockworks~tiktok-scraper/run-sync-get-dataset-items",
	) {
		t.Errorf("path = %q, want the sync run endpoint of the actor", gotReq.URL.Path)
	}
	if auth := gotReq.Header.Get("Authorization"); auth != "Bearer apify-token" {
		t.Errorf("authorization = %q, want the bearer token", auth)
	}
	if gotPayload["shouldDownloadVideos"] != true {
		t.Errorf("shouldDownloadVideos = %v, want true", gotPayload["shouldDownloadVideos"])
	}
	urls, ok := gotPayload["postURLs"].([]any)
	if !ok || len(urls) != 1 || urls[0] != "https://www.tiktok.com/@a/video/2" {
		t.Errorf("postURLs = %v, want the video url", gotPayload["postURLs"])
	}
}

func TestTikTokScraperClient_ResolveMediaURL_GoneVideoIsPermanent(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantText string
	}{
		{"message only", `[{"error":"Post not found or private."}]`, "Post not found or private"},
		{
			"code only",
			`[{"error":"gone","errorCode":"POST_NOT_FOUND_OR_PRIVATE"}]`,
			"gone",
		},
		{"sensitive post", `[{"error":"flagged","errorCode":"POST_SENSITIVE"}]`, "flagged"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newScraperTestClient(http.StatusCreated, tt.body, nil)

			_, err := client.ResolveMediaURL(t.Context(), "https://www.tiktok.com/@a/video/1")

			if !errors.Is(err, data.ErrMediaUnavailable) {
				t.Fatalf("error = %v, want ErrMediaUnavailable", err)
			}
			if !strings.Contains(err.Error(), tt.wantText) {
				t.Errorf("error = %q, want it to keep the actor message %q", err, tt.wantText)
			}
		})
	}
}

// Anything that is not "the platform says this video is gone" must stay
// retryable, so it must never look like ErrMediaUnavailable.
func TestTikTokScraperClient_ResolveMediaURL_FailuresAreRetryable(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"apify error status", http.StatusInternalServerError, `{"error":"boom"}`},
		{"invalid json", http.StatusOK, `not json`},
		{"empty dataset", http.StatusCreated, `[]`},
		{
			"item without media urls",
			http.StatusCreated,
			`[{"id":"1"},{"id":"2","mediaUrls":[]}]`,
		},
		{"load error", http.StatusCreated, `[{"error":"Could not load the page."}]`},
		{"blocked", http.StatusCreated, `[{"error":"challenge","errorCode":"BLOCKED"}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newScraperTestClient(tt.status, tt.body, nil)

			_, err := client.ResolveMediaURL(t.Context(), "https://www.tiktok.com/@a/video/1")

			if err == nil {
				t.Fatal("expected an error")
			}
			if errors.Is(err, data.ErrMediaUnavailable) {
				t.Errorf("error = %v, a failed call must not be reported as missing media", err)
			}
		})
	}
}

func TestNewTikTokScraperClient_HasNoClientTimeout(t *testing.T) {
	c := NewTikTokScraperClient("token", "actor")

	if c.httpClient.Timeout != 0 {
		t.Errorf("client timeout = %v, want none: each call sets its own deadline",
			c.httpClient.Timeout)
	}
}

func TestTikTokScraperClient_RunDeadlines(t *testing.T) {
	tests := []struct {
		name string
		call func(ctx context.Context, c *TikTokScraperClient) error
		want time.Duration
	}{
		{
			"media run",
			func(ctx context.Context, c *TikTokScraperClient) error {
				_, err := c.ResolveMediaURL(ctx, "https://www.tiktok.com/@a/video/1")
				return err
			},
			scraperMediaTimeout,
		},
		{
			"profile scrape",
			func(ctx context.Context, c *TikTokScraperClient) error {
				_, err := c.FetchProfileVideos(ctx, "a", 5)
				return err
			},
			scraperListTimeout,
		},
		{
			"keyword search",
			func(ctx context.Context, c *TikTokScraperClient) error {
				_, err := c.SearchByKeyword(ctx, "lemon", 5)
				return err
			},
			scraperListTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				deadline    time.Time
				hasDeadline bool
			)
			client := newScraperTestClient(
				http.StatusCreated,
				`[{"id":"1","mediaUrls":["https://media.test/v.mp4"]}]`,
				func(r *http.Request, _ []byte) {
					deadline, hasDeadline = r.Context().Deadline()
				},
			)

			if err := tt.call(t.Context(), client); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !hasDeadline {
				t.Fatal("request has no deadline")
			}
			if left := time.Until(deadline); left > tt.want || left < tt.want-10*time.Second {
				t.Errorf("deadline in %v, want about %v", left, tt.want)
			}
		})
	}
}
