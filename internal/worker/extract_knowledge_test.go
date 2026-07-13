package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

// newDocumentServiceWithGetByIDError returns a *service.DocumentService whose
// GetByID always fails with err.
func newDocumentServiceWithGetByIDError(err error) *service.DocumentService {
	repo := &mockrepo.DocumentRepoerMock{}
	repo.GetByIDFunc = func(_ context.Context, _ uuid.UUID) (*model.Document, error) {
		return nil, err
	}
	return service.NewDocumentService(repo)
}

// stubExtractor is a hand-rolled data.KnowledgeExtractor for testing without
// a real LLM provider.
type stubExtractor struct {
	extract func(req *data.ExtractionRequest) ([]data.ExtractedFact, error)
	facts   []data.ExtractedFact
	err     error
}

func (s *stubExtractor) Extract(
	_ context.Context,
	req *data.ExtractionRequest,
) ([]data.ExtractedFact, error) {
	if s.extract != nil {
		return s.extract(req)
	}
	return s.facts, s.err
}

func newExtractJob(documentID uuid.UUID) *river.Job[queue.ExtractKnowledgeDTO] {
	return &river.Job[queue.ExtractKnowledgeDTO]{
		Args: queue.ExtractKnowledgeDTO{DocumentID: documentID},
	}
}

// newAtomicKnowledgeRepoWithTx returns a mockrepo.AtomicKnowledgeRepoerMock
// wired so WithTx calls back into itself (the established pattern, mirroring
// newPassThroughChunkRepo in process_parsed_document_test.go).
func newAtomicKnowledgeRepoWithTx() *mockrepo.AtomicKnowledgeRepoerMock {
	repo := &mockrepo.AtomicKnowledgeRepoerMock{}
	repo.WithTxFunc = func(_ context.Context, fn func(data.AtomicKnowledgeRepoer) error) error {
		return fn(repo)
	}
	return repo
}

func TestExtractKnowledgeWorker_Work_Success(t *testing.T) {
	chunk1 := model.DocumentChunk{ID: uuid.New(), ParsedChunk: model.ParsedChunk{Text: "Acme was founded by Alice."}}
	chunk2 := model.DocumentChunk{ID: uuid.New(), ParsedChunk: model.ParsedChunk{Text: "The sky is blue."}}

	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.ListChunksNeedingKnowledgeExtractionFunc = func(_ context.Context, _ uuid.UUID) ([]model.DocumentChunk, error) {
		return []model.DocumentChunk{chunk1, chunk2}, nil
	}

	documentService := newDocumentServiceWithTitle("Company History")

	knowledgeRepo := newAtomicKnowledgeRepoWithTx()
	var created []model.AtomicKnowledge
	knowledgeRepo.CreateBatchFunc = func(_ context.Context, facts []model.AtomicKnowledge) error {
		created = append(created, facts...)
		return nil
	}
	var markedChunkIDs []uuid.UUID
	knowledgeRepo.MarkChunkKnowledgeExtractedFunc = func(_ context.Context, chunkID uuid.UUID) error {
		markedChunkIDs = append(markedChunkIDs, chunkID)
		return nil
	}

	extractor := &stubExtractor{
		extract: func(req *data.ExtractionRequest) ([]data.ExtractedFact, error) {
			if req.Text == chunk1.Text {
				return []data.ExtractedFact{
					{Subject: "Alice", Predicate: "founded", Object: "Acme", TruthTier: model.TruthTierAxiomatic},
				}, nil
			}
			return nil, nil // chunk2 yields zero facts
		},
	}

	w := NewExtractKnowledgeWorker(chunkRepo, knowledgeRepo, documentService, extractor)
	if err := w.Work(t.Context(), newExtractJob(uuid.New())); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(created) != 1 {
		t.Fatalf("expected 1 fact created, got %d", len(created))
	}
	if created[0].Subject != "Alice" || created[0].DocumentChunkID != chunk1.ID {
		t.Errorf("unexpected created fact: %+v", created[0])
	}
	if len(markedChunkIDs) != 2 {
		t.Fatalf("expected both chunks marked extracted, got %d", len(markedChunkIDs))
	}
	if markedChunkIDs[0] != chunk1.ID || markedChunkIDs[1] != chunk2.ID {
		t.Errorf("expected chunks marked in order [chunk1 chunk2], got %v", markedChunkIDs)
	}
}

func TestExtractKnowledgeWorker_Work_NoChunksNeedingExtraction_NoOp(t *testing.T) {
	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.ListChunksNeedingKnowledgeExtractionFunc = func(_ context.Context, _ uuid.UUID) ([]model.DocumentChunk, error) {
		return nil, nil
	}

	knowledgeRepo := newAtomicKnowledgeRepoWithTx()
	knowledgeRepo.CreateBatchFunc = func(context.Context, []model.AtomicKnowledge) error {
		t.Error("CreateBatch should not be called when there are no chunks needing extraction")
		return nil
	}

	extractor := &stubExtractor{
		extract: func(*data.ExtractionRequest) ([]data.ExtractedFact, error) {
			t.Error("Extract should not be called when there are no chunks needing extraction")
			return nil, nil
		},
	}

	w := NewExtractKnowledgeWorker(chunkRepo, knowledgeRepo, newDocumentServiceWithTitle("Doc"), extractor)
	if err := w.Work(t.Context(), newExtractJob(uuid.New())); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExtractKnowledgeWorker_Work_ExtractorError_ChunkNotMarked(t *testing.T) {
	chunk := model.DocumentChunk{ID: uuid.New(), ParsedChunk: model.ParsedChunk{Text: "text"}}
	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.ListChunksNeedingKnowledgeExtractionFunc = func(_ context.Context, _ uuid.UUID) ([]model.DocumentChunk, error) {
		return []model.DocumentChunk{chunk}, nil
	}

	knowledgeRepo := newAtomicKnowledgeRepoWithTx()
	knowledgeRepo.MarkChunkKnowledgeExtractedFunc = func(context.Context, uuid.UUID) error {
		t.Error("MarkChunkKnowledgeExtracted should not be called when Extract fails")
		return nil
	}

	extractor := &stubExtractor{err: errors.New("llm endpoint down")}

	w := NewExtractKnowledgeWorker(chunkRepo, knowledgeRepo, newDocumentServiceWithTitle("Doc"), extractor)
	if err := w.Work(t.Context(), newExtractJob(uuid.New())); err == nil {
		t.Fatal("expected an error when the extractor fails")
	}
}

func TestExtractKnowledgeWorker_Work_CreateBatchError(t *testing.T) {
	chunk := model.DocumentChunk{ID: uuid.New(), ParsedChunk: model.ParsedChunk{Text: "text"}}
	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.ListChunksNeedingKnowledgeExtractionFunc = func(_ context.Context, _ uuid.UUID) ([]model.DocumentChunk, error) {
		return []model.DocumentChunk{chunk}, nil
	}

	knowledgeRepo := newAtomicKnowledgeRepoWithTx()
	knowledgeRepo.CreateBatchFunc = func(context.Context, []model.AtomicKnowledge) error {
		return errors.New("db down")
	}
	knowledgeRepo.MarkChunkKnowledgeExtractedFunc = func(context.Context, uuid.UUID) error {
		t.Error("MarkChunkKnowledgeExtracted should not be called when CreateBatch fails")
		return nil
	}

	extractor := &stubExtractor{
		facts: []data.ExtractedFact{{Subject: "a", Predicate: "b", Object: "c"}},
	}

	w := NewExtractKnowledgeWorker(chunkRepo, knowledgeRepo, newDocumentServiceWithTitle("Doc"), extractor)
	if err := w.Work(t.Context(), newExtractJob(uuid.New())); err == nil {
		t.Fatal("expected an error when CreateBatch fails")
	}
}

func TestExtractKnowledgeWorker_Work_DocumentServiceError(t *testing.T) {
	chunkRepo := &mockrepo.DocumentChunkRepoerMock{}
	chunkRepo.ListChunksNeedingKnowledgeExtractionFunc = func(context.Context, uuid.UUID) ([]model.DocumentChunk, error) {
		t.Error("ListChunksNeedingKnowledgeExtraction should not be called when fetching the document fails")
		return nil, nil
	}

	knowledgeRepo := newAtomicKnowledgeRepoWithTx()
	extractor := &stubExtractor{}

	documentService := newDocumentServiceWithGetByIDError(errors.New("db down"))
	w := NewExtractKnowledgeWorker(chunkRepo, knowledgeRepo, documentService, extractor)
	if err := w.Work(t.Context(), newExtractJob(uuid.New())); err == nil {
		t.Fatal("expected an error when fetching the document fails")
	}
}
