package worker

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

// rawDoclingJSON builds a minimal doclingServeResponse-shaped JSON payload
// with the given text/picture snippets embedded directly, for testing
// external.ParseRaw's caller (ProcessParsedDocumentWorker) without a real
// Docling Serve instance.
func rawDoclingJSON(title, textsJSON, picturesJSON string) string {
	return `{"status":"success","document":{"json_content":{"name":"` + title +
		`","texts":[` + textsJSON + `],"pictures":[` + picturesJSON + `]}}}`
}

func readCloserFromString(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

func TestFilterNoiseChunks(t *testing.T) {
	chunks := []model.ParsedChunk{
		{Text: "paragraph", LayoutType: model.LayoutTypeParagraph},
		{Text: "page header", LayoutType: model.LayoutTypePageHeader},
		{Text: "title", LayoutType: model.LayoutTypeTitle},
		{Text: "page footer", LayoutType: model.LayoutTypePageFooter},
		{LayoutType: model.LayoutTypeTable},
		{Text: "document index", LayoutType: model.LayoutTypeDocumentIndex},
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

func TestFilterNoiseChunks_DropsEmptyTextChunks(t *testing.T) {
	chunks := []model.ParsedChunk{
		{
			Text:       "   ",
			LayoutType: model.LayoutTypeParagraph,
		},
		{
			Text:       "Important information",
			LayoutType: model.LayoutTypeParagraph,
		},
		{
			Text:       "",
			LayoutType: model.LayoutTypeTable,
		},
	}

	kept, dropped := filterNoiseChunks(chunks)

	if dropped != 1 {
		t.Fatalf("expected 1 dropped chunk, got %d", dropped)
	}

	if len(kept) != 2 {
		t.Fatalf("expected 2 kept chunks, got %d", len(kept))
	}

	if kept[0].Text != "Important information" {
		t.Errorf("unexpected first chunk: %q", kept[0].Text)
	}

	if kept[1].LayoutType != model.LayoutTypeTable {
		t.Errorf("expected empty table chunk to be preserved")
	}
}

func TestFilterNoiseChunks_MergesSectionHeaderWithText(t *testing.T) {
	header := model.ParsedChunk{
		Text:       "Methodology",
		Page:       1,
		LayoutType: model.LayoutTypeSectionHeader,
	}

	paragraph := model.ParsedChunk{
		Text:       "This research uses a quantitative approach.",
		Page:       1,
		LayoutType: model.LayoutTypeText,
		BoundingBox: model.BoundingBox{
			1, 2, 3, 4,
		},
	}

	kept, dropped := filterNoiseChunks([]model.ParsedChunk{
		header,
		paragraph,
	})

	if dropped != 0 {
		t.Fatalf("expected 0 dropped chunks, got %d", dropped)
	}

	if len(kept) != 1 {
		t.Fatalf("expected 1 merged chunk, got %d", len(kept))
	}

	expected := "Methodology\nThis research uses a quantitative approach."

	if kept[0].Text != expected {
		t.Errorf(
			"unexpected merged text:\nwant: %q\ngot: %q",
			expected,
			kept[0].Text,
		)
	}

	if kept[0].Page != paragraph.Page {
		t.Error("expected merged chunk to preserve paragraph page")
	}

	if kept[0].BoundingBox != paragraph.BoundingBox {
		t.Error("expected merged chunk to preserve paragraph bounding box")
	}
}

func TestFilterNoiseChunks_MergesSectionHeaderWithConsecutiveListItems(t *testing.T) {
	header := model.ParsedChunk{
		Text:       "Recommendations",
		LayoutType: model.LayoutTypeSectionHeader,
	}

	first := model.ParsedChunk{
		Text:       "Use treatment A",
		Page:       2,
		LayoutType: model.LayoutTypeListItem,
		BoundingBox: model.BoundingBox{
			1, 2, 3, 4,
		},
	}

	empty := model.ParsedChunk{
		Text:       "",
		LayoutType: model.LayoutTypeListItem,
	}

	second := model.ParsedChunk{
		Text:       "Avoid treatment B",
		Page:       3,
		LayoutType: model.LayoutTypeListItem,
	}

	paragraph := model.ParsedChunk{
		Text:       "Additional notes",
		LayoutType: model.LayoutTypeParagraph,
	}

	kept, dropped := filterNoiseChunks([]model.ParsedChunk{
		header,
		first,
		empty,
		second,
		paragraph,
	})

	if dropped != 1 {
		t.Fatalf("expected 1 dropped chunk, got %d", dropped)
	}

	if len(kept) != 3 {
		t.Fatalf("expected 3 kept chunks, got %d", len(kept))
	}

	if kept[0].Text != "Recommendations\nUse treatment A" {
		t.Errorf("unexpected first item: %q", kept[0].Text)
	}

	if kept[1].Text != "Recommendations\nAvoid treatment B" {
		t.Errorf("unexpected second item: %q", kept[1].Text)
	}

	if kept[0].Page != first.Page ||
		kept[0].BoundingBox != first.BoundingBox {
		t.Error("expected first item metadata to be preserved")
	}

	if kept[2].Text != paragraph.Text {
		t.Errorf("paragraph after list was unexpectedly changed")
	}
}

func TestFilterNoiseChunks_DoesNotMergeSectionHeaderWithOtherLayouts(t *testing.T) {
	layouts := []model.LayoutType{
		model.LayoutTypeParagraph,
		model.LayoutTypeTable,
		model.LayoutTypeCaption,
	}

	for _, layout := range layouts {
		chunks := []model.ParsedChunk{
			{
				Text:       "A much longer section title",
				LayoutType: model.LayoutTypeSectionHeader,
			},
			{
				Text:       "Content",
				LayoutType: layout,
			},
		}

		kept, dropped := filterNoiseChunks(chunks)

		if dropped != 0 {
			t.Errorf(
				"layout %s: expected 0 dropped, got %d",
				layout,
				dropped,
			)
		}

		if len(kept) != 2 {
			t.Errorf(
				"layout %s: expected 2 chunks, got %d",
				layout,
				len(kept),
			)
			continue
		}

		if kept[0].Text != "A much longer section title" ||
			kept[1].Text != "Content" {
			t.Errorf(
				"layout %s: section header was merged unexpectedly",
				layout,
			)
		}
	}
}

func TestFilterNoiseChunks_PreservesStandaloneSectionHeaders(t *testing.T) {
	chunks := []model.ParsedChunk{
		{
			Text:       "Methods",
			LayoutType: model.LayoutTypeSectionHeader,
		},
		{
			Text:       "Risk factors",
			LayoutType: model.LayoutTypeSectionHeader,
		},
		{
			Text:       "Treatment",
			LayoutType: model.LayoutTypeSectionHeader,
		},
		{
			Text:       "Conclusion",
			LayoutType: model.LayoutTypeSectionHeader,
		},
		{
			Text:       "A much longer standalone section header",
			LayoutType: model.LayoutTypeSectionHeader,
		},
	}

	kept, dropped := filterNoiseChunks(chunks)

	if dropped != 0 {
		t.Fatalf("expected 0 dropped chunks, got %d", dropped)
	}

	if len(kept) != len(chunks) {
		t.Fatalf(
			"expected %d kept chunks, got %d",
			len(chunks),
			len(kept),
		)
	}

	for i, chunk := range chunks {
		if kept[i].Text != chunk.Text {
			t.Errorf(
				"chunk %d: expected %q, got %q",
				i,
				chunk.Text,
				kept[i].Text,
			)
		}

		if kept[i].LayoutType != model.LayoutTypeSectionHeader {
			t.Errorf(
				"chunk %d: expected section_header, got %s",
				i,
				kept[i].LayoutType,
			)
		}
	}
}

func TestFilterNoiseChunksWithDecisions(t *testing.T) {
	chunks := []model.ParsedChunk{
		{
			Text:       "Recommendations",
			LayoutType: model.LayoutTypeSectionHeader,
		},
		{
			Text:       "Use treatment A",
			LayoutType: model.LayoutTypeListItem,
		},
		{
			Text:       "Avoid treatment B",
			LayoutType: model.LayoutTypeListItem,
		},
		{
			Text:       "Additional notes",
			LayoutType: model.LayoutTypeParagraph,
		},
		{
			Text:       "Document header",
			LayoutType: model.LayoutTypePageHeader,
		},
	}

	kept, dropped, decisions := filterNoiseChunksWithDecisions(chunks)

	if len(kept) != 3 {
		t.Fatalf("expected 3 kept chunks, got %d", len(kept))
	}

	if dropped != 1 {
		t.Fatalf("expected 1 dropped chunk, got %d", dropped)
	}

	if len(decisions) != 4 {
		t.Fatalf("expected 4 decisions, got %d", len(decisions))
	}

	expected := []struct {
		status    chunkFilterStatus
		keptIndex int
	}{
		{chunkFilterStatusMerge, 0},
		{chunkFilterStatusMerge, 1},
		{chunkFilterStatusKeep, 2},
		{chunkFilterStatusDrop, -1},
	}

	for i, want := range expected {
		if decisions[i].Status != want.status {
			t.Errorf(
				"decision %d: expected status %s, got %s",
				i,
				want.status,
				decisions[i].Status,
			)
		}

		if decisions[i].KeptIndex != want.keptIndex {
			t.Errorf(
				"decision %d: expected kept index %d, got %d",
				i,
				want.keptIndex,
				decisions[i].KeptIndex,
			)
		}
	}

	if decisions[0].Chunk.Text != "Recommendations\nUse treatment A" {
		t.Errorf(
			"unexpected first merged text: %q",
			decisions[0].Chunk.Text,
		)
	}

	if decisions[1].Chunk.Text != "Recommendations\nAvoid treatment B" {
		t.Errorf(
			"unexpected second merged text: %q",
			decisions[1].Chunk.Text,
		)
	}

	if decisions[0].SourceIndexes[0] != 0 ||
		decisions[0].SourceIndexes[1] != 1 {
		t.Errorf(
			"unexpected first merge source indexes: %v",
			decisions[0].SourceIndexes,
		)
	}
}

// newPassThroughChunkRepo returns a mockrepo.DocumentChunkRepoerMock wired so
// WithTx calls back into itself (the established pattern), ShouldBeStored
// always allows storage, CreateBatch/Enqueue are no-op successes.
func newPassThroughChunkRepo() *mockrepo.DocumentChunkRepoerMock {
	repo := &mockrepo.DocumentChunkRepoerMock{}
	repo.WithTxFunc = func(_ context.Context, fn func(data.DocumentChunkRepoer) error) error {
		return fn(repo)
	}
	repo.ShouldBeStoredFunc = func(_ context.Context, _ model.Document, _ model.DocumentChunk) (bool, error) {
		return true, nil
	}
	repo.CreateBatchFunc = func(_ context.Context, _ []model.DocumentChunk) error {
		return nil
	}
	repo.EnqueueFunc = func(_ context.Context, _ queue.BackgroundJob) error {
		return nil
	}
	return repo
}

// newPassThroughDocumentService returns a *service.DocumentService wrapping
// a mockrepo.DocumentRepoerMock whose GetByID/Update always succeed —
// wherever a test just needs the worker's page-count/title backfill to not error.
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

func newProcessParsedDocumentJob(rawOutputKey string) *river.Job[queue.ProcessParsedDocumentDTO] {
	return &river.Job[queue.ProcessParsedDocumentDTO]{
		Args: queue.ProcessParsedDocumentDTO{DocumentID: uuid.New(), RawOutputKey: rawOutputKey},
	}
}

func TestProcessParsedDocumentWorker_Work_Success(t *testing.T) {
	raw := rawDoclingJSON(
		"Test Doc",
		`{"text":"body","label":"paragraph","prov":[{"page_no":3,"bbox":{"l":0,"t":0,"r":1,"b":1}}]}`,
		"",
	)
	rawStore := &stubRawStore{
		getObjectFunc: func(context.Context, string) (io.ReadCloser, error) {
			return readCloserFromString(raw), nil
		},
	}

	var gotTitle string
	var gotPageCount int
	documentRepo := &mockrepo.DocumentRepoerMock{}
	documentRepo.GetByIDFunc = func(_ context.Context, id uuid.UUID) (*model.Document, error) {
		return &model.Document{ID: id}, nil
	}
	documentRepo.UpdateFunc = func(_ context.Context, doc *model.Document) error {
		gotTitle = doc.Title
		gotPageCount = doc.PageCount
		return nil
	}
	documentService := service.NewDocumentService(documentRepo)

	w := NewProcessParsedDocumentWorker(rawStore, documentService, newPassThroughChunkRepo())
	if err := w.Work(
		t.Context(),
		newProcessParsedDocumentJob("parsed_output/doc.json"),
	); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTitle != "Test Doc" {
		t.Errorf("expected title %q, got %q", "Test Doc", gotTitle)
	}
	if gotPageCount != 3 {
		t.Errorf("expected page count 3, got %d", gotPageCount)
	}
}

func TestProcessParsedDocumentWorker_Work_UploadsImageAndEnqueuesCaption(t *testing.T) {
	raw := rawDoclingJSON(
		"Doc With Image",
		"",
		`{"label":"picture","prov":[{"page_no":1,"bbox":{"l":0,"t":0,"r":1,"b":1}}],"image":{"uri":"data:image/png;base64,aGk="}}`,
	)
	rawStore := &stubRawStore{
		getObjectFunc: func(context.Context, string) (io.ReadCloser, error) {
			return readCloserFromString(raw), nil
		},
		putObjectFunc: func(_ context.Context, key string, r io.Reader, _ int64, contentType string) error {
			if !strings.HasPrefix(key, "document_images/") || !strings.HasSuffix(key, ".png") {
				t.Errorf(
					"expected an image key under document_images/ with .png extension, got %q",
					key,
				)
			}
			if contentType != "image/png" {
				t.Errorf("expected content type image/png, got %q", contentType)
			}
			body, _ := io.ReadAll(r)
			if string(body) != "hi" {
				t.Errorf("expected decoded image bytes %q, got %q", "hi", body)
			}
			return nil
		},
	}

	var stored []model.DocumentChunk
	var enqueuedDTO queue.CaptionImageChunksDTO
	var enqueuedCalled bool
	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.WithTxFunc = func(_ context.Context, fn func(data.DocumentChunkRepoer) error) error {
		return fn(chunkRepo)
	}
	chunkRepo.ShouldBeStoredFunc = func(_ context.Context, _ model.Document, _ model.DocumentChunk) (bool, error) {
		return true, nil
	}
	chunkRepo.CreateBatchFunc = func(_ context.Context, chunks []model.DocumentChunk) error {
		stored = chunks
		return nil
	}
	chunkRepo.EnqueueFunc = func(_ context.Context, payload queue.BackgroundJob) error {
		enqueuedCalled = true
		dto, ok := payload.(queue.CaptionImageChunksDTO)
		if !ok {
			t.Fatalf("expected queue.CaptionImageChunksDTO, got %T", payload)
		}
		enqueuedDTO = dto
		return nil
	}

	job := newProcessParsedDocumentJob("parsed_output/doc.json")
	w := NewProcessParsedDocumentWorker(rawStore, newPassThroughDocumentService(), chunkRepo)
	if err := w.Work(t.Context(), job); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(stored) != 1 {
		t.Fatalf("expected 1 stored chunk, got %d", len(stored))
	}
	if stored[0].AssetURL == "" {
		t.Error("expected the image chunk to have a non-empty AssetURL")
	}
	if stored[0].RawImageData != nil {
		t.Error("expected RawImageData to be cleared before storage")
	}
	if !enqueuedCalled {
		t.Fatal("expected image captioning to be enqueued")
	}
	if enqueuedDTO.DocumentID != job.Args.DocumentID {
		t.Errorf(
			"expected enqueued DocumentID %s, got %s",
			job.Args.DocumentID,
			enqueuedDTO.DocumentID,
		)
	}
	if len(enqueuedDTO.ChunkIDs) != 1 || enqueuedDTO.ChunkIDs[0] != stored[0].ID {
		t.Errorf(
			"expected enqueued ChunkIDs to be exactly [%s], got %v",
			stored[0].ID,
			enqueuedDTO.ChunkIDs,
		)
	}
}

func TestProcessParsedDocumentWorker_Work_NoImagesEnqueuesEmbeddingNotCaption(t *testing.T) {
	raw := rawDoclingJSON(
		"Plain Doc",
		`{"text":"body","label":"paragraph","prov":[{"page_no":1,"bbox":{"l":0,"t":0,"r":1,"b":1}}]}`,
		"",
	)
	rawStore := &stubRawStore{
		getObjectFunc: func(context.Context, string) (io.ReadCloser, error) {
			return readCloserFromString(raw), nil
		},
	}

	var enqueued []queue.BackgroundJob
	chunkRepo := newPassThroughChunkRepo()
	chunkRepo.EnqueueFunc = func(_ context.Context, payload queue.BackgroundJob) error {
		enqueued = append(enqueued, payload)
		return nil
	}

	job := newProcessParsedDocumentJob("parsed_output/doc.json")
	w := NewProcessParsedDocumentWorker(rawStore, newPassThroughDocumentService(), chunkRepo)
	if err := w.Work(t.Context(), job); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(enqueued) != 3 {
		t.Fatalf(
			"expected 3 jobs enqueued (embed + extract + translate), got %d: %+v",
			len(enqueued), enqueued,
		)
	}
	embedJob, ok := enqueued[0].(queue.EmbedChunksDTO)
	if !ok {
		t.Fatalf("expected first enqueued job to be queue.EmbedChunksDTO, got %T", enqueued[0])
	}
	if embedJob.DocumentID != job.Args.DocumentID {
		t.Errorf(
			"expected enqueued embed job DocumentID %s, got %s",
			job.Args.DocumentID,
			embedJob.DocumentID,
		)
	}
	extractJob, ok := enqueued[1].(queue.ExtractKnowledgeDTO)
	if !ok {
		t.Fatalf(
			"expected second enqueued job to be queue.ExtractKnowledgeDTO, got %T",
			enqueued[1],
		)
	}
	if extractJob.DocumentID != job.Args.DocumentID {
		t.Errorf(
			"expected enqueued extract job DocumentID %s, got %s",
			job.Args.DocumentID,
			extractJob.DocumentID,
		)
	}
	translateJob, ok := enqueued[2].(queue.TranslateChunksDTO)
	if !ok {
		t.Fatalf(
			"expected third enqueued job to be queue.TranslateChunksDTO, got %T",
			enqueued[2],
		)
	}
	if translateJob.DocumentID != job.Args.DocumentID {
		t.Errorf(
			"expected enqueued translate job DocumentID %s, got %s",
			job.Args.DocumentID,
			translateJob.DocumentID,
		)
	}
}

func TestProcessParsedDocumentWorker_Work_ImageUploadError_NoCreateBatch(t *testing.T) {
	raw := rawDoclingJSON(
		"Doc With Image",
		"",
		`{"label":"picture","prov":[{"page_no":1,"bbox":{"l":0,"t":0,"r":1,"b":1}}],"image":{"uri":"data:image/png;base64,aGk="}}`,
	)
	rawStore := &stubRawStore{
		getObjectFunc: func(context.Context, string) (io.ReadCloser, error) {
			return readCloserFromString(raw), nil
		},
		putObjectFunc: func(context.Context, string, io.Reader, int64, string) error {
			return errors.New("s3 down")
		},
	}

	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.WithTxFunc = func(_ context.Context, fn func(data.DocumentChunkRepoer) error) error {
		return fn(chunkRepo)
	}
	chunkRepo.CreateBatchFunc = func(context.Context, []model.DocumentChunk) error {
		t.Error("CreateBatch should not be called when image upload fails")
		return nil
	}

	w := NewProcessParsedDocumentWorker(rawStore, newPassThroughDocumentService(), chunkRepo)
	if err := w.Work(
		t.Context(),
		newProcessParsedDocumentJob("parsed_output/doc.json"),
	); err == nil {
		t.Fatal("expected an error when image upload fails")
	}
}

func TestProcessParsedDocumentWorker_Work_GetObjectError(t *testing.T) {
	rawStore := &stubRawStore{
		getObjectFunc: func(context.Context, string) (io.ReadCloser, error) {
			return nil, errors.New("s3 down")
		},
	}
	w := NewProcessParsedDocumentWorker(
		rawStore,
		newPassThroughDocumentService(),
		newPassThroughChunkRepo(),
	)
	if err := w.Work(
		t.Context(),
		newProcessParsedDocumentJob("parsed_output/doc.json"),
	); err == nil {
		t.Fatal("expected an error when GetObject fails")
	}
}

func TestProcessParsedDocumentWorker_Work_ParseRawError(t *testing.T) {
	rawStore := &stubRawStore{
		getObjectFunc: func(context.Context, string) (io.ReadCloser, error) {
			return readCloserFromString("not json"), nil
		},
	}
	w := NewProcessParsedDocumentWorker(
		rawStore,
		newPassThroughDocumentService(),
		newPassThroughChunkRepo(),
	)
	if err := w.Work(
		t.Context(),
		newProcessParsedDocumentJob("parsed_output/doc.json"),
	); err == nil {
		t.Fatal("expected an error when the raw output isn't valid JSON")
	}
}

func TestProcessParsedDocumentWorker_Work_UpdatesPageCountAndBackfillsBlankTitle(t *testing.T) {
	raw := rawDoclingJSON(
		"Docling's Title",
		`{"text":"body","label":"paragraph","prov":[{"page_no":7,"bbox":{"l":0,"t":0,"r":1,"b":1}}]}`,
		"",
	)
	rawStore := &stubRawStore{
		getObjectFunc: func(context.Context, string) (io.ReadCloser, error) {
			return readCloserFromString(raw), nil
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

	job := newProcessParsedDocumentJob("parsed_output/doc.json")
	w := NewProcessParsedDocumentWorker(rawStore, documentService, newPassThroughChunkRepo())

	if err := w.Work(t.Context(), job); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotID != job.Args.DocumentID {
		t.Errorf("expected update for document %s, got %s", job.Args.DocumentID, gotID)
	}
	if gotPageCount != 7 {
		t.Errorf("expected page count 7, got %d", gotPageCount)
	}
	if gotTitle != "Docling's Title" {
		t.Errorf("expected blank title to be backfilled from Docling, got %q", gotTitle)
	}
}

func TestProcessParsedDocumentWorker_Work_PreservesExistingTitle(t *testing.T) {
	raw := rawDoclingJSON(
		"Docling's Title",
		`{"text":"body","label":"paragraph","prov":[{"page_no":7,"bbox":{"l":0,"t":0,"r":1,"b":1}}]}`,
		"",
	)
	rawStore := &stubRawStore{
		getObjectFunc: func(context.Context, string) (io.ReadCloser, error) {
			return readCloserFromString(raw), nil
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

	w := NewProcessParsedDocumentWorker(rawStore, documentService, newPassThroughChunkRepo())
	if err := w.Work(
		t.Context(),
		newProcessParsedDocumentJob("parsed_output/doc.json"),
	); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTitle != "User-Supplied Title" {
		t.Errorf("expected user-supplied title to be preserved, got %q", gotTitle)
	}
}

func TestProcessParsedDocumentWorker_Work_UpdateAfterParseError(t *testing.T) {
	raw := rawDoclingJSON("Doc", "", "")
	rawStore := &stubRawStore{
		getObjectFunc: func(context.Context, string) (io.ReadCloser, error) {
			return readCloserFromString(raw), nil
		},
	}

	documentRepo := &mockrepo.DocumentRepoerMock{}
	documentRepo.GetByIDFunc = func(_ context.Context, _ uuid.UUID) (*model.Document, error) {
		return nil, errors.New("db down")
	}
	documentService := service.NewDocumentService(documentRepo)

	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.CreateBatchFunc = func(context.Context, []model.DocumentChunk) error {
		t.Error("CreateBatch should not be called when UpdateAfterParse fails")
		return nil
	}

	w := NewProcessParsedDocumentWorker(rawStore, documentService, chunkRepo)
	if err := w.Work(
		t.Context(),
		newProcessParsedDocumentJob("parsed_output/doc.json"),
	); err == nil {
		t.Fatal("expected an error when UpdateAfterParse fails")
	}
}

func TestProcessParsedDocumentWorker_Work_ShouldBeStoredFiltersChunks(t *testing.T) {
	raw := rawDoclingJSON(
		"Doc",
		`{"text":"keep me","label":"paragraph","prov":[{"page_no":1,"bbox":{"l":0,"t":0,"r":1,"b":1}}]},`+
			`{"text":"drop me","label":"paragraph","prov":[{"page_no":1,"bbox":{"l":0,"t":0,"r":1,"b":1}}]},`+
			`{"text":"keep me too","label":"title","prov":[{"page_no":1,"bbox":{"l":0,"t":0,"r":1,"b":1}}]}`,
		"",
	)
	rawStore := &stubRawStore{
		getObjectFunc: func(context.Context, string) (io.ReadCloser, error) {
			return readCloserFromString(raw), nil
		},
	}

	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.WithTxFunc = func(_ context.Context, fn func(data.DocumentChunkRepoer) error) error {
		return fn(chunkRepo)
	}
	chunkRepo.ShouldBeStoredFunc = func(_ context.Context, _ model.Document, chunk model.DocumentChunk) (bool, error) {
		return chunk.Text != "drop me", nil
	}
	var stored []model.DocumentChunk
	chunkRepo.CreateBatchFunc = func(_ context.Context, chunks []model.DocumentChunk) error {
		stored = chunks
		return nil
	}
	chunkRepo.EnqueueFunc = func(context.Context, queue.BackgroundJob) error {
		return nil
	}

	w := NewProcessParsedDocumentWorker(rawStore, newPassThroughDocumentService(), chunkRepo)
	if err := w.Work(
		t.Context(),
		newProcessParsedDocumentJob("parsed_output/doc.json"),
	); err != nil {
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

func TestProcessParsedDocumentWorker_Work_CreateBatchError(t *testing.T) {
	raw := rawDoclingJSON(
		"Doc",
		`{"text":"body","label":"paragraph","prov":[{"page_no":1,"bbox":{"l":0,"t":0,"r":1,"b":1}}]}`,
		"",
	)
	rawStore := &stubRawStore{
		getObjectFunc: func(context.Context, string) (io.ReadCloser, error) {
			return readCloserFromString(raw), nil
		},
	}

	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.WithTxFunc = func(_ context.Context, fn func(data.DocumentChunkRepoer) error) error {
		return fn(chunkRepo)
	}
	chunkRepo.ShouldBeStoredFunc = func(_ context.Context, _ model.Document, _ model.DocumentChunk) (bool, error) {
		return true, nil
	}
	chunkRepo.CreateBatchFunc = func(_ context.Context, _ []model.DocumentChunk) error {
		return errors.New("db down")
	}

	w := NewProcessParsedDocumentWorker(rawStore, newPassThroughDocumentService(), chunkRepo)
	if err := w.Work(
		t.Context(),
		newProcessParsedDocumentJob("parsed_output/doc.json"),
	); err == nil {
		t.Fatal("expected an error when CreateBatch fails")
	}
}

func TestProcessParsedDocumentWorker_Work_NoChunksSurviveFilter(t *testing.T) {
	raw := rawDoclingJSON(
		"Doc",
		`{"text":"header","label":"page_header","prov":[{"page_no":1,"bbox":{"l":0,"t":0,"r":1,"b":1}}]},`+
			`{"text":"footer","label":"page_footer","prov":[{"page_no":1,"bbox":{"l":0,"t":0,"r":1,"b":1}}]}`,
		"",
	)
	rawStore := &stubRawStore{
		getObjectFunc: func(context.Context, string) (io.ReadCloser, error) {
			return readCloserFromString(raw), nil
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
	chunkRepo.EnqueueFunc = func(context.Context, queue.BackgroundJob) error {
		return nil
	}

	w := NewProcessParsedDocumentWorker(rawStore, newPassThroughDocumentService(), chunkRepo)
	if err := w.Work(
		t.Context(),
		newProcessParsedDocumentJob("parsed_output/doc.json"),
	); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !createBatchCalled {
		t.Fatal("expected CreateBatch to be called even with zero surviving chunks")
	}
	if receivedLen != 0 {
		t.Errorf("expected CreateBatch to receive an empty slice, got %d", receivedLen)
	}
}
