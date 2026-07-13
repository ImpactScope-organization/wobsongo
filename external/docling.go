package external

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// DoclingClient implements data.DocumentProcessor using a Docling Serve HTTP API instance.
type DoclingClient struct {
	baseURL    string
	httpClient *http.Client
}

// Ensure DoclingClient implements data.DocumentProcessor.
var _ data.DocumentProcessor = (*DoclingClient)(nil)

// NewDoclingClient creates a new DoclingClient targeting the given Docling Serve base URL.
func NewDoclingClient(baseURL string) *DoclingClient {
	return &DoclingClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 6 * time.Minute,
		},
	}
}

// doclingConvertRequest is the request payload for Docling Serve's /v1/convert/source endpoint.
type doclingConvertRequest struct {
	Sources []doclingSource `json:"sources"`
	Options doclingOptions  `json:"options"`
}

type doclingSource struct {
	Kind string `json:"kind"`
	URL  string `json:"url"`
}

type doclingOptions struct {
	ToFormats       []string `json:"to_formats"`
	ImageExportMode string   `json:"image_export_mode"`
}

// doclingServeResponse represents the root JSON object returned by Docling Serve.
type doclingServeResponse struct {
	Document struct {
		JSONContent doclingDocument `json:"json_content"`
	} `json:"document"`
	Status string `json:"status"`
}

// doclingDocument is the native unified document format exported by docling-core.
type doclingDocument struct {
	Name     string               `json:"name"`
	Texts    []doclingTextItem    `json:"texts"`
	Tables   []doclingTableItem   `json:"tables"`
	Pictures []doclingPictureItem `json:"pictures"`
}

type doclingTextItem struct {
	Text  string        `json:"text"`
	Label string        `json:"label"`
	Prov  []doclingProv `json:"prov"`
}

type doclingTableItem struct {
	Label string        `json:"label"`
	Prov  []doclingProv `json:"prov"`
	// Table cell/grid data (Data field) intentionally not mapped yet —
	// out of scope until a future sub-task needs structured table content.
}

type doclingPictureItem struct {
	Label string        `json:"label"`
	Prov  []doclingProv `json:"prov"`
	Image *doclingImage `json:"image,omitempty"`
}

// doclingProv contains the physical location of an element on the PDF.
// An element can span multiple pages, but only the first entry is used here.
type doclingProv struct {
	PageNo int       `json:"page_no"`
	BBox   []float64 `json:"bbox"` // [left, top, right, bottom]
}

// doclingImage holds the base64 representation of an extracted image, as a
// "data:<mime-type>;base64,<payload>" URI. Decoded by decodeDataURI in
// mapDoclingDocument.
type doclingImage struct {
	URI string `json:"uri"`
}

// FetchRawFromURL fetches the document at documentURL via Docling Serve's
// synchronous /v1/convert/source endpoint and returns the raw, unparsed
// response body — implements data.DocumentProcessor. The HTTP fetch and the
// response interpretation (ParseRaw) are deliberately separate, independently
// retryable steps: a raw response can be large (embedded images push some
// past 100MB) and expensive to re-fetch, so callers persist it once (e.g. to
// S3) and parse it out of band.
func (c *DoclingClient) FetchRawFromURL(ctx context.Context, documentURL string) ([]byte, error) {
	return c.convertFromURL(ctx, documentURL)
}

// ParseRaw unmarshals Docling Serve's raw JSON response (as returned by
// FetchRawFromURL or ConvertFileRaw) into data.ProcessedDocument. Filtering
// by layout type is left to the caller.
func ParseRaw(raw []byte) (*data.ProcessedDocument, error) {
	var parsed doclingServeResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal docling response: %w", err)
	}

	return mapDoclingDocument(&parsed.Document.JSONContent), nil
}

// ConvertFileRaw uploads r's content directly to Docling Serve's
// /v1/convert/file endpoint and returns the raw, unparsed response body.
// Unlike ConvertFromURLRaw, this doesn't require the document to already be
// reachable via a URL — needed when the caller can't hand Docling Serve a
// fetchable URL (e.g. a local dev S3/MinIO that a cloud-hosted Docling
// instance can't reach).
func (c *DoclingClient) ConvertFileRaw(
	ctx context.Context,
	filename string,
	r io.Reader,
) ([]byte, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("to_formats", "json"); err != nil {
		return nil, fmt.Errorf("failed to write to_formats field: %w", err)
	}
	if err := writer.WriteField("image_export_mode", "embedded"); err != nil {
		return nil, fmt.Errorf("failed to write image_export_mode field: %w", err)
	}
	part, err := writer.CreateFormFile("files", filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, r); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return c.doRequest(ctx, "/v1/convert/file", writer.FormDataContentType(), body)
}

// convertFromURL performs the actual Docling Serve /v1/convert/source call
// and returns the raw response body.
func (c *DoclingClient) convertFromURL(ctx context.Context, documentURL string) ([]byte, error) {
	payload := doclingConvertRequest{
		Sources: []doclingSource{
			{Kind: "http", URL: documentURL},
		},
		Options: doclingOptions{
			ToFormats:       []string{"json"},
			ImageExportMode: "embedded",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal docling request: %w", err)
	}

	return c.doRequest(ctx, "/v1/convert/source", "application/json", bytes.NewReader(body))
}

// doRequest POSTs body to path on Docling Serve and returns the raw
// response body, shared by every convert variant (URL-based, file-upload).
func (c *DoclingClient) doRequest(
	ctx context.Context,
	path, contentType string,
	body io.Reader,
) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create docling request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call docling serve: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read docling response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"docling serve returned error status: %d. Body: %s",
			resp.StatusCode,
			string(respBytes),
		)
	}

	return respBytes, nil
}

// mapDoclingDocument maps a doclingDocument into data.ProcessedDocument.
//
// Known simplification: Docling returns texts/tables/pictures as three
// separate arrays, not one true cross-type reading-order array, so
// concatenating them in this fixed order is an approximation of the real
// reading order, not a faithful global one. True interleaved ordering (by
// page + position) is deferred until a future sub-task needs it.
func mapDoclingDocument(doc *doclingDocument) *data.ProcessedDocument {
	chunks := make([]model.ParsedChunk, 0, len(doc.Texts)+len(doc.Tables)+len(doc.Pictures))
	maxPage := 0

	for _, t := range doc.Texts {
		page, bbox := firstProv(t.Prov)
		maxPage = max(maxPage, page)
		chunks = append(chunks, model.ParsedChunk{
			Text:        t.Text,
			Page:        page,
			LayoutType:  model.LayoutType(t.Label),
			BoundingBox: bbox,
		})
	}

	for _, t := range doc.Tables {
		page, bbox := firstProv(t.Prov)
		maxPage = max(maxPage, page)
		chunks = append(chunks, model.ParsedChunk{
			Page:        page,
			LayoutType:  model.LayoutType(t.Label),
			BoundingBox: bbox,
		})
	}

	for _, p := range doc.Pictures {
		page, bbox := firstProv(p.Prov)
		maxPage = max(maxPage, page)
		chunk := model.ParsedChunk{
			Page:        page,
			LayoutType:  model.LayoutType(p.Label),
			BoundingBox: bbox,
		}
		if p.Image != nil {
			contentType, decoded, err := decodeDataURI(p.Image.URI)
			if err != nil {
				// A malformed/undecodable image shouldn't sink the whole
				// document parse — keep the chunk (page/bbox/layout-type
				// still useful) just without image data to upload later.
				log.Printf("[docling] failed to decode embedded image: %v", err)
			} else {
				chunk.RawImageData = decoded
				chunk.RawImageContentType = contentType
			}
		}
		chunks = append(chunks, chunk)
	}

	return &data.ProcessedDocument{
		Title:     doc.Name,
		PageCount: maxPage,
		Chunks:    chunks,
	}
}

// firstProv extracts the page number and bounding box from the first
// provenance entry, defaulting to zero values when prov is empty.
func firstProv(prov []doclingProv) (int, model.BoundingBox) {
	if len(prov) == 0 {
		return 0, model.BoundingBox{}
	}
	var bbox model.BoundingBox
	copy(bbox[:], prov[0].BBox)
	return prov[0].PageNo, bbox
}

// decodeDataURI splits a "data:<mime-type>;base64,<payload>" URI (Docling's
// embedded-image format) into its declared content type and decoded bytes.
func decodeDataURI(uri string) (contentType string, decoded []byte, err error) {
	rest, ok := strings.CutPrefix(uri, "data:")
	if !ok {
		return "", nil, errors.New("missing \"data:\" prefix")
	}
	meta, payload, ok := strings.Cut(rest, ",")
	if !ok {
		return "", nil, errors.New("missing comma separator")
	}
	mimeType, ok := strings.CutSuffix(meta, ";base64")
	if !ok {
		return "", nil, fmt.Errorf("unsupported data URI encoding %q (only base64 supported)", meta)
	}
	decoded, err = base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", nil, fmt.Errorf("failed to base64-decode image data: %w", err)
	}
	return mimeType, decoded, nil
}
