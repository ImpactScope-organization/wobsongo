package external

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

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
				payload, _ := io.ReadAll(r.Body)
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
	urls, _ := gotPayload["postURLs"].([]any)
	if len(urls) != 1 || urls[0] != "https://www.tiktok.com/@a/video/2" {
		t.Errorf("postURLs = %v, want the video url", gotPayload["postURLs"])
	}
}

func TestTikTokScraperClient_ResolveMediaURL_NoMedia(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"no items", `[]`},
		{"items without media urls", `[{"id":"1"},{"id":"2","mediaUrls":[]}]`},
		{"actor error item", `[{"error":"Post not found or private"}]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newScraperTestClient(http.StatusCreated, tt.body, nil)

			_, err := client.ResolveMediaURL(t.Context(), "https://www.tiktok.com/@a/video/1")

			if !errors.Is(err, data.ErrMediaUnavailable) {
				t.Errorf("error = %v, want ErrMediaUnavailable", err)
			}
		})
	}
}

// Anything that is not "this video has no media" must stay retryable, so it
// must never look like ErrMediaUnavailable.
func TestTikTokScraperClient_ResolveMediaURL_FailuresAreRetryable(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"apify error status", http.StatusInternalServerError, `{"error":"boom"}`},
		{"invalid json", http.StatusOK, `not json`},
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
