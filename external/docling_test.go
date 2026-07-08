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
				{"label": "picture", "prov": [{"page_no": 4, "bbox": [0, 0, 2, 2]}], "image": {"uri": "data:image/png;base64,abc"}}
			]
		}
	}
}`

func TestDoclingClient_ProcessFromURL_Success(t *testing.T) {
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

	result, err := client.ProcessFromURL(context.Background(), "https://example.com/doc.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Title != "sample.pdf" {
		t.Errorf("expected title %q, got %q", "sample.pdf", result.Title)
	}

	// Filtering is NOT this client's job — noise-type items (page_footer) must
	// still be present, unfiltered.
	if len(result.Chunks) != 5 {
		t.Fatalf(
			"expected 5 unfiltered chunks (3 text + 1 table + 1 picture), got %d",
			len(result.Chunks),
		)
	}

	var sawPageFooter, sawTable, sawPicture bool
	for _, c := range result.Chunks {
		switch c.LayoutType {
		case model.LayoutTypePageFooter:
			sawPageFooter = true
		case model.LayoutTypeTable:
			sawTable = true
		case model.LayoutTypePicture:
			sawPicture = true
		}
	}
	if !sawPageFooter {
		t.Error("expected the noise page_footer chunk to be present, unfiltered")
	}
	if !sawTable {
		t.Error("expected the table chunk to be present")
	}
	if !sawPicture {
		t.Error("expected the picture chunk to be present")
	}

	first := result.Chunks[0]
	if first.Text != "Sample Guideline" || first.LayoutType != model.LayoutTypeTitle ||
		first.Page != 1 {
		t.Errorf("unexpected first chunk mapping: %+v", first)
	}
}

func TestDoclingClient_ProcessFromURL_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("docling exploded"))
	}))
	defer server.Close()

	client := external.NewDoclingClient(server.URL)

	_, err := client.ProcessFromURL(context.Background(), "https://example.com/doc.pdf")
	if err == nil {
		t.Fatal("expected an error for a non-200 response")
	}
	if !strings.Contains(err.Error(), "docling exploded") {
		t.Errorf("expected error to include response body, got: %v", err)
	}
}
