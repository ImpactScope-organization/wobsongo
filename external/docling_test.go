package external_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

const cannedDoclingResponse = `{
	"status": "success",
	"document": {
		"json_content": {
			"name": "sample.pdf",
			"texts": [
				{"text": "Sample Guideline", "label": "title", "prov": [{"page_no": 1, "bbox": [0, 0, 10, 10]}]},
				{"text": "Page 1 of 5", "label": "page_footer", "prov": [{"page_no": 1, "bbox": [0, 0, 1, 1]}]},
				{"text": "Some body text.", "label": "paragraph", "prov": [{"page_no": 2, "bbox": [1, 2, 3, 4]}]}
			],
			"tables": [
				{"label": "table", "prov": [{"page_no": 3, "bbox": [0, 0, 5, 5]}]}
			],
			"pictures": [
				{"label": "picture", "prov": [{"page_no": 4, "bbox": [0, 0, 2, 2]}], "image": {"uri": "data:image/png;base64,aGk="}}
			]
		}
	}
}`

func TestDoclingClient_FetchRawFromURL_And_ParseRaw_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/convert/source" {
			t.Errorf("expected path /v1/convert/source, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cannedDoclingResponse))
	}))
	defer server.Close()

	client := external.NewDoclingClient(server.URL)

	raw, err := client.FetchRawFromURL(context.Background(), "https://example.com/doc.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(string(raw), "sample.pdf") {
		t.Fatalf("expected raw response to contain the canned body, got: %s", raw)
	}

	result, err := external.ParseRaw(raw)
	if err != nil {
		t.Fatalf("unexpected error parsing raw response: %v", err)
	}

	if result.Title != "sample.pdf" {
		t.Errorf("expected title %q, got %q", "sample.pdf", result.Title)
	}

	// Filtering is NOT this package's job — noise-type items (page_footer) must
	// still be present, unfiltered.
	if len(result.Chunks) != 5 {
		t.Fatalf(
			"expected 5 unfiltered chunks (3 text + 1 table + 1 picture), got %d",
			len(result.Chunks),
		)
	}

	var sawPageFooter, sawTable bool
	var picture *model.ParsedChunk
	for i, c := range result.Chunks {
		switch c.LayoutType {
		case model.LayoutTypePageFooter:
			sawPageFooter = true
		case model.LayoutTypeTable:
			sawTable = true
		case model.LayoutTypePicture:
			picture = &result.Chunks[i]
		}
	}
	if !sawPageFooter {
		t.Error("expected the noise page_footer chunk to be present, unfiltered")
	}
	if !sawTable {
		t.Error("expected the table chunk to be present")
	}
	if picture == nil {
		t.Fatal("expected the picture chunk to be present")
	}
	if string(picture.RawImageData) != "hi" {
		t.Errorf("expected decoded image bytes %q, got %q", "hi", picture.RawImageData)
	}
	if picture.RawImageContentType != "image/png" {
		t.Errorf("expected content type %q, got %q", "image/png", picture.RawImageContentType)
	}

	first := result.Chunks[0]
	if first.Text != "Sample Guideline" || first.LayoutType != model.LayoutTypeTitle ||
		first.Page != 1 {
		t.Errorf("unexpected first chunk mapping: %+v", first)
	}
}

func TestParseRaw_MalformedImageDoesNotFailParse(t *testing.T) {
	const raw = `{
		"status": "success",
		"document": {
			"json_content": {
				"name": "sample.pdf",
				"pictures": [
					{"label": "picture", "prov": [{"page_no": 1, "bbox": [0, 0, 1, 1]}], "image": {"uri": "not-a-data-uri"}}
				]
			}
		}
	}`

	result, err := external.ParseRaw([]byte(raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Chunks) != 1 {
		t.Fatalf("expected the picture chunk to still be kept, got %d chunks", len(result.Chunks))
	}
	if result.Chunks[0].RawImageData != nil {
		t.Errorf(
			"expected no decoded image data for a malformed URI, got %q",
			result.Chunks[0].RawImageData,
		)
	}
}

func TestDoclingClient_FetchRawFromURL_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("docling exploded"))
	}))
	defer server.Close()

	client := external.NewDoclingClient(server.URL)

	_, err := client.FetchRawFromURL(context.Background(), "https://example.com/doc.pdf")
	if err == nil {
		t.Fatal("expected an error for a non-200 response")
	}
	if !strings.Contains(err.Error(), "docling exploded") {
		t.Errorf("expected error to include response body, got: %v", err)
	}
}
