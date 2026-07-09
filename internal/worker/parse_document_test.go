package worker

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
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

// newPassThroughChunkRepo returns a mockrepo.DocumentChunkRepoerMock wired so
// WithTx calls back into itself (the established pattern), ShouldBeStored
// always allows storage, and CreateBatch is a no-op success.
func newPassThroughChunkRepo() *mockrepo.DocumentChunkRepoerMock {
	repo := &mockrepo.DocumentChunkRepoerMock{}
	repo.WithTxFunc = func(_ context.Context, fn func(data.DocumentChunkRepoer) error) error {
		return fn(repo)
	}
	repo.ShouldBeStoredFunc = func(_ context.Context, _ model.DocumentChunk) (bool, error) {
		return true, nil
	}
	repo.CreateBatchFunc = func(_ context.Context, _ []model.DocumentChunk) error {
		return nil
	}
	return repo
}

// newPassThroughDocumentService returns a *service.DocumentService wrapping
// a mockrepo.DocumentRepoerMock whose GetByID/Update always succeed —
// wherever a test just needs the worker's page-count backfill to not error.
func newPassThroughDocumentService() *service.DocumentService {
	repo := &mockrepo.DocumentRepoerMock{}
	repo.GetByIDFunc = func(_ context.Context, id uuid.UUID) (*model.Document, error) {
		return &model.Document{ID: id}, nil
	}
	repo.UpdateFunc = func(_ context.Context, _ *model.Document) error {
		return nil
	}
	return service.NewDocumentService(repo)
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
	w := NewParseDocumentWorker(
		processor,
		mediaService,
		newPassThroughChunkRepo(),
		newPassThroughDocumentService(),
	)

	if err := w.Work(t.Context(), newTestJob("documents/abc.pdf")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseDocumentWorker_Work_PresignError(t *testing.T) {
	mediaService := service.NewMediaService(&stubMediaProvider{err: errors.New("boom")})
	w := NewParseDocumentWorker(
		&stubProcessor{},
		mediaService,
		newPassThroughChunkRepo(),
		newPassThroughDocumentService(),
	)

	if err := w.Work(t.Context(), newTestJob("documents/abc.pdf")); err == nil {
		t.Fatal("expected an error when presigning fails")
	}
}

func TestParseDocumentWorker_Work_ProcessorError(t *testing.T) {
	mediaService := service.NewMediaService(
		&stubMediaProvider{presignedURL: "https://example.com/doc.pdf"},
	)
	processor := &stubProcessor{err: errors.New("docling down")}
	w := NewParseDocumentWorker(
		processor,
		mediaService,
		newPassThroughChunkRepo(),
		newPassThroughDocumentService(),
	)

	if err := w.Work(t.Context(), newTestJob("documents/abc.pdf")); err == nil {
		t.Fatal("expected an error when the processor fails")
	}
}

func TestParseDocumentWorker_Work_UpdatesPageCountAndBackfillsBlankTitle(t *testing.T) {
	mediaService := service.NewMediaService(
		&stubMediaProvider{presignedURL: "https://example.com/doc.pdf"},
	)
	processor := &stubProcessor{
		result: &data.ProcessedDocument{
			Title:     "Docling's Title",
			PageCount: 7,
			Chunks:    []model.ParsedChunk{{LayoutType: model.LayoutTypeParagraph}},
		},
	}

	documentRepo := &mockrepo.DocumentRepoerMock{}
	var gotID uuid.UUID
	var gotPageCount int
	var gotTitle string
	documentRepo.GetByIDFunc = func(_ context.Context, id uuid.UUID) (*model.Document, error) {
		return &model.Document{ID: id}, nil
	}
	documentRepo.UpdateFunc = func(_ context.Context, doc *model.Document) error {
		gotID = doc.ID
		gotPageCount = doc.PageCount
		gotTitle = doc.Title
		return nil
	}
	documentService := service.NewDocumentService(documentRepo)

	job := newTestJob("documents/abc.pdf")
	w := NewParseDocumentWorker(processor, mediaService, newPassThroughChunkRepo(), documentService)

	if err := w.Work(t.Context(), job); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotID != job.Args.DocumentID {
		t.Errorf("expected page count update for document %s, got %s", job.Args.DocumentID, gotID)
	}
	if gotPageCount != 7 {
		t.Errorf("expected page count 7, got %d", gotPageCount)
	}
	if gotTitle != "Docling's Title" {
		t.Errorf("expected blank title to be backfilled from Docling, got %q", gotTitle)
	}
}

func TestParseDocumentWorker_Work_PreservesExistingTitle(t *testing.T) {
	mediaService := service.NewMediaService(
		&stubMediaProvider{presignedURL: "https://example.com/doc.pdf"},
	)
	processor := &stubProcessor{
		result: &data.ProcessedDocument{
			Title:     "Docling's Title",
			PageCount: 7,
			Chunks:    []model.ParsedChunk{{LayoutType: model.LayoutTypeParagraph}},
		},
	}

	documentRepo := &mockrepo.DocumentRepoerMock{}
	var gotTitle string
	documentRepo.GetByIDFunc = func(_ context.Context, id uuid.UUID) (*model.Document, error) {
		return &model.Document{ID: id, Title: "User-Supplied Title"}, nil
	}
	documentRepo.UpdateFunc = func(_ context.Context, doc *model.Document) error {
		gotTitle = doc.Title
		return nil
	}
	documentService := service.NewDocumentService(documentRepo)

	w := NewParseDocumentWorker(processor, mediaService, newPassThroughChunkRepo(), documentService)
	if err := w.Work(t.Context(), newTestJob("documents/abc.pdf")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTitle != "User-Supplied Title" {
		t.Errorf("expected user-supplied title to be preserved, got %q", gotTitle)
	}
}

func TestParseDocumentWorker_Work_UpdateAfterParseError(t *testing.T) {
	mediaService := service.NewMediaService(
		&stubMediaProvider{presignedURL: "https://example.com/doc.pdf"},
	)
	processor := &stubProcessor{result: &data.ProcessedDocument{PageCount: 3}}

	documentRepo := &mockrepo.DocumentRepoerMock{}
	documentRepo.GetByIDFunc = func(_ context.Context, _ uuid.UUID) (*model.Document, error) {
		return nil, errors.New("db down")
	}
	documentService := service.NewDocumentService(documentRepo)

	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.CreateBatchFunc = func(_ context.Context, _ []model.DocumentChunk) error {
		t.Error("CreateBatch should not be called when UpdateAfterParse fails")
		return nil
	}

	w := NewParseDocumentWorker(processor, mediaService, chunkRepo, documentService)
	if err := w.Work(t.Context(), newTestJob("documents/abc.pdf")); err == nil {
		t.Fatal("expected an error when UpdateAfterParse fails")
	}
}

func TestParseDocumentWorker_Work_ShouldBeStoredFiltersChunks(t *testing.T) {
	mediaService := service.NewMediaService(
		&stubMediaProvider{presignedURL: "https://example.com/doc.pdf"},
	)
	processor := &stubProcessor{
		result: &data.ProcessedDocument{
			Chunks: []model.ParsedChunk{
				{Text: "keep me", LayoutType: model.LayoutTypeParagraph},
				{Text: "drop me", LayoutType: model.LayoutTypeParagraph},
				{Text: "keep me too", LayoutType: model.LayoutTypeTitle},
			},
		},
	}

	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.WithTxFunc = func(_ context.Context, fn func(data.DocumentChunkRepoer) error) error {
		return fn(chunkRepo)
	}
	chunkRepo.ShouldBeStoredFunc = func(_ context.Context, chunk model.DocumentChunk) (bool, error) {
		return chunk.Text != "drop me", nil
	}
	var stored []model.DocumentChunk
	chunkRepo.CreateBatchFunc = func(_ context.Context, chunks []model.DocumentChunk) error {
		stored = chunks
		return nil
	}

	w := NewParseDocumentWorker(processor, mediaService, chunkRepo, newPassThroughDocumentService())
	if err := w.Work(t.Context(), newTestJob("documents/abc.pdf")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stored) != 2 {
		t.Fatalf("expected 2 stored chunks, got %d", len(stored))
	}
	for _, c := range stored {
		if c.Text == "drop me" {
			t.Errorf("expected the filtered-out chunk not to be stored")
		}
	}
	// SequenceNumber is the index into `kept` (post-noise-filter order), not
	// the index into the surviving-ShouldBeStored slice.
	if stored[0].SequenceNumber != 0 || stored[1].SequenceNumber != 2 {
		t.Errorf(
			"unexpected sequence numbers: got %d, %d",
			stored[0].SequenceNumber,
			stored[1].SequenceNumber,
		)
	}
}

func TestParseDocumentWorker_Work_CreateBatchError(t *testing.T) {
	mediaService := service.NewMediaService(
		&stubMediaProvider{presignedURL: "https://example.com/doc.pdf"},
	)
	processor := &stubProcessor{
		result: &data.ProcessedDocument{
			Chunks: []model.ParsedChunk{{LayoutType: model.LayoutTypeParagraph}},
		},
	}

	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.WithTxFunc = func(_ context.Context, fn func(data.DocumentChunkRepoer) error) error {
		return fn(chunkRepo)
	}
	chunkRepo.ShouldBeStoredFunc = func(_ context.Context, _ model.DocumentChunk) (bool, error) {
		return true, nil
	}
	chunkRepo.CreateBatchFunc = func(_ context.Context, _ []model.DocumentChunk) error {
		return errors.New("db down")
	}

	w := NewParseDocumentWorker(processor, mediaService, chunkRepo, newPassThroughDocumentService())
	if err := w.Work(t.Context(), newTestJob("documents/abc.pdf")); err == nil {
		t.Fatal("expected an error when CreateBatch fails")
	}
}

func TestParseDocumentWorker_Work_NoChunksSurviveFilter(t *testing.T) {
	mediaService := service.NewMediaService(
		&stubMediaProvider{presignedURL: "https://example.com/doc.pdf"},
	)
	processor := &stubProcessor{
		result: &data.ProcessedDocument{
			Chunks: []model.ParsedChunk{
				{LayoutType: model.LayoutTypePageHeader},
				{LayoutType: model.LayoutTypePageFooter},
			},
		},
	}

	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.WithTxFunc = func(_ context.Context, fn func(data.DocumentChunkRepoer) error) error {
		return fn(chunkRepo)
	}
	// ShouldBeStoredFunc intentionally left nil — it must never be called,
	// since no chunks survive filterNoiseChunks; the mock panics if it is.
	var createBatchCalled bool
	receivedLen := -1
	chunkRepo.CreateBatchFunc = func(_ context.Context, chunks []model.DocumentChunk) error {
		createBatchCalled = true
		receivedLen = len(chunks)
		return nil
	}

	w := NewParseDocumentWorker(processor, mediaService, chunkRepo, newPassThroughDocumentService())
	if err := w.Work(t.Context(), newTestJob("documents/abc.pdf")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !createBatchCalled {
		t.Fatal("expected CreateBatch to be called even with zero surviving chunks")
	}
	if receivedLen != 0 {
		t.Errorf("expected CreateBatch to receive an empty slice, got %d", receivedLen)
	}
}
