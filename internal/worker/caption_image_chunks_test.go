package worker

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/riverqueue/river"
)

// stubCaptioner is a hand-rolled data.ImageCaptioner for testing without a
// real VLM provider.
type stubCaptioner struct {
	caption string
	err     error
}

func (s *stubCaptioner) Caption(_ context.Context, _ []byte, _ string) (string, error) {
	return s.caption, s.err
}

func newCaptionJob(
	documentID uuid.UUID,
	chunkIDs ...uuid.UUID,
) *river.Job[queue.CaptionImageChunksDTO] {
	return &river.Job[queue.CaptionImageChunksDTO]{
		Args: queue.CaptionImageChunksDTO{DocumentID: documentID, ChunkIDs: chunkIDs},
	}
}

func TestCaptionImageChunksWorker_Work_Success(t *testing.T) {
	chunkID := uuid.New()
	chunk := &model.DocumentChunk{
		ID: chunkID,
		ParsedChunk: model.ParsedChunk{
			AssetURL: "document_images/abc.png",
		},
	}

	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.GetByIDFunc = func(_ context.Context, id uuid.UUID) (*model.DocumentChunk, error) {
		return chunk, nil
	}
	var updatedText string
	chunkRepo.UpdateFunc = func(_ context.Context, c *model.DocumentChunk) error {
		updatedText = c.Text
		return nil
	}

	var gotContentType string
	rawStore := &stubRawStore{
		getObjectFunc: func(_ context.Context, key string) (io.ReadCloser, error) {
			if key != chunk.AssetURL {
				t.Errorf("expected to fetch %q, got %q", chunk.AssetURL, key)
			}
			return io.NopCloser(io.LimitReader(nil, 0)), nil
		},
	}
	w := &CaptionImageChunksWorker{
		RawStore:  rawStore,
		ChunkRepo: chunkRepo,
		Captioner: &captureContentTypeCaptioner{result: "a detailed caption", got: &gotContentType},
	}

	job := newCaptionJob(uuid.New(), chunkID)
	if err := w.Work(t.Context(), job); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedText != "a detailed caption" {
		t.Errorf("expected chunk Text to be set to the caption, got %q", updatedText)
	}
	if gotContentType != "image/png" {
		t.Errorf("expected content type image/png inferred from .png key, got %q", gotContentType)
	}
}

// captureContentTypeCaptioner records the contentType it was called with.
type captureContentTypeCaptioner struct {
	result string
	err    error
	got    *string
}

func (c *captureContentTypeCaptioner) Caption(
	_ context.Context,
	_ []byte,
	contentType string,
) (string, error) {
	*c.got = contentType
	return c.result, c.err
}

func TestCaptionImageChunksWorker_Work_SkipsAlreadyCaptionedChunk(t *testing.T) {
	chunkID := uuid.New()
	chunk := &model.DocumentChunk{
		ID: chunkID,
		ParsedChunk: model.ParsedChunk{
			AssetURL: "document_images/abc.png",
			Text:     "already captioned",
		},
	}

	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.GetByIDFunc = func(_ context.Context, id uuid.UUID) (*model.DocumentChunk, error) {
		return chunk, nil
	}
	chunkRepo.UpdateFunc = func(context.Context, *model.DocumentChunk) error {
		t.Error("Update should not be called for an already-captioned chunk")
		return nil
	}

	rawStore := &stubRawStore{
		getObjectFunc: func(context.Context, string) (io.ReadCloser, error) {
			t.Error("GetObject should not be called for an already-captioned chunk")
			return nil, errors.New("should not be called")
		},
	}
	captioner := &stubCaptioner{
		err: errors.New("Caption should not be called for an already-captioned chunk"),
	}

	w := &CaptionImageChunksWorker{RawStore: rawStore, ChunkRepo: chunkRepo, Captioner: captioner}
	if err := w.Work(t.Context(), newCaptionJob(uuid.New(), chunkID)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCaptionImageChunksWorker_Work_CaptionerError(t *testing.T) {
	chunkID := uuid.New()
	chunk := &model.DocumentChunk{
		ID:          chunkID,
		ParsedChunk: model.ParsedChunk{AssetURL: "document_images/abc.png"},
	}

	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.GetByIDFunc = func(_ context.Context, id uuid.UUID) (*model.DocumentChunk, error) {
		return chunk, nil
	}
	chunkRepo.UpdateFunc = func(context.Context, *model.DocumentChunk) error {
		t.Error("Update should not be called when Caption fails")
		return nil
	}

	rawStore := &stubRawStore{
		getObjectFunc: func(context.Context, string) (io.ReadCloser, error) {
			return io.NopCloser(io.LimitReader(nil, 0)), nil
		},
	}
	captioner := &stubCaptioner{err: errors.New("not yet implemented")}

	w := &CaptionImageChunksWorker{RawStore: rawStore, ChunkRepo: chunkRepo, Captioner: captioner}
	if err := w.Work(t.Context(), newCaptionJob(uuid.New(), chunkID)); err == nil {
		t.Fatal("expected an error when the captioner fails")
	}
}
