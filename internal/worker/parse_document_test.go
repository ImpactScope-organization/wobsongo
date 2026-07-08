package worker

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

func TestFilterNoiseChunks(t *testing.T) {
	chunks := []model.ParsedChunk{
		{LayoutType: model.LayoutTypeParagraph},
		{LayoutType: model.LayoutTypePageHeader},
		{LayoutType: model.LayoutTypeTitle},
		{LayoutType: model.LayoutTypePageFooter},
		{LayoutType: model.LayoutTypeTable},
		{LayoutType: model.LayoutTypeDocumentIndex},
	}

	kept, dropped := filterNoiseChunks(chunks)

	if dropped != 3 {
		t.Errorf("expected 3 dropped, got %d", dropped)
	}
	if len(kept) != 3 {
		t.Fatalf("expected 3 kept, got %d", len(kept))
	}
	for _, c := range kept {
		if noiseLayoutTypes[c.LayoutType] {
			t.Errorf("kept a noise chunk: %s", c.LayoutType)
		}
	}
}

// stubProcessor is a hand-rolled data.DocumentProcessor for testing the
// worker without a real Docling Serve instance.
type stubProcessor struct {
	result *data.ProcessedDocument
	err    error
}

func (s *stubProcessor) ProcessFromURL(
	_ context.Context,
	_ string,
) (*data.ProcessedDocument, error) {
	return s.result, s.err
}

// stubMediaProvider is a hand-rolled data.MediaUploadProvider for testing
// ParseDocumentWorker via a real *service.MediaService without real S3/MinIO.
type stubMediaProvider struct {
	presignedURL string
	err          error
}

func (s *stubMediaProvider) GetPresignedPOSTURL(
	_ context.Context,
	_ data.MediaUploadIntent,
	_, _ string,
) (*url.URL, map[string]string, error) {
	panic("not used in this test")
}

func (s *stubMediaProvider) GetPresignedGETURL(
	_ context.Context,
	_ string,
	_ int64,
) (string, error) {
	return s.presignedURL, s.err
}

func (s *stubMediaProvider) GetPresignedGETURLs(
	_ context.Context,
	_ []string,
	_ int64,
) (map[string]string, error) {
	panic("not used in this test")
}

func newTestJob(fileKey string) *river.Job[queue.ParseDocumentDTO] {
	return &river.Job[queue.ParseDocumentDTO]{
		Args: queue.ParseDocumentDTO{DocumentID: uuid.New(), FileKey: fileKey},
	}
}

func TestParseDocumentWorker_Work_Success(t *testing.T) {
	mediaService := service.NewMediaService(
		&stubMediaProvider{presignedURL: "https://example.com/doc.pdf"},
	)
	processor := &stubProcessor{
		result: &data.ProcessedDocument{
			Title:     "Test Doc",
			PageCount: 3,
			Chunks: []model.ParsedChunk{
				{LayoutType: model.LayoutTypeParagraph},
				{LayoutType: model.LayoutTypePageFooter},
			},
		},
	}
	w := NewParseDocumentWorker(processor, mediaService)

	if err := w.Work(t.Context(), newTestJob("documents/abc.pdf")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseDocumentWorker_Work_PresignError(t *testing.T) {
	mediaService := service.NewMediaService(&stubMediaProvider{err: errors.New("boom")})
	w := NewParseDocumentWorker(&stubProcessor{}, mediaService)

	if err := w.Work(t.Context(), newTestJob("documents/abc.pdf")); err == nil {
		t.Fatal("expected an error when presigning fails")
	}
}

func TestParseDocumentWorker_Work_ProcessorError(t *testing.T) {
	mediaService := service.NewMediaService(
		&stubMediaProvider{presignedURL: "https://example.com/doc.pdf"},
	)
	processor := &stubProcessor{err: errors.New("docling down")}
	w := NewParseDocumentWorker(processor, mediaService)

	if err := w.Work(t.Context(), newTestJob("documents/abc.pdf")); err == nil {
		t.Fatal("expected an error when the processor fails")
	}
}
