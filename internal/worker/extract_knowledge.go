package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

// extractKnowledgeTimeout bounds how long the worker waits across all of a
// document's chunks. Generous relative to a single extraction call since a
// document can carry many chunks, each requiring its own LLM call.
const extractKnowledgeTimeout = 15 * time.Minute

// ExtractKnowledgeWorker is a River worker that extracts atomic knowledge
// facts (subject-predicate-object, with a truth-tier classification) from a
// document's text-bearing, not-yet-extracted chunks. Kept as its own job
// (not inline in chunk storage/captioning) so a slow/costly/rate-limited LLM
// call never fails or retries those steps themselves.
type ExtractKnowledgeWorker struct {
	// Embedding River's default worker behavior for the specific DTO.
	river.WorkerDefaults[queue.ExtractKnowledgeDTO]
	// ChunkRepo lists chunks needing extraction.
	ChunkRepo data.DocumentChunkRepoer
	// KnowledgeRepo stores extracted facts and marks chunks as processed.
	KnowledgeRepo data.AtomicKnowledgeRepoer
	// DocumentService fetches the parent document's title, for grounding.
	DocumentService *service.DocumentService
	// Extractor extracts facts from a chunk's text.
	Extractor data.KnowledgeExtractor
}

// NewExtractKnowledgeWorker is a constructor for ExtractKnowledgeWorker.
func NewExtractKnowledgeWorker(
	chunkRepo data.DocumentChunkRepoer,
	knowledgeRepo data.AtomicKnowledgeRepoer,
	documentService *service.DocumentService,
	extractor data.KnowledgeExtractor,
) *ExtractKnowledgeWorker {
	return &ExtractKnowledgeWorker{
		ChunkRepo:       chunkRepo,
		KnowledgeRepo:   knowledgeRepo,
		DocumentService: documentService,
		Extractor:       extractor,
	}
}

// Timeout overrides River's default job timeout.
func (w *ExtractKnowledgeWorker) Timeout(*river.Job[queue.ExtractKnowledgeDTO]) time.Duration {
	return extractKnowledgeTimeout
}

// Work is the main method that gets called when a job is dequeued.
func (w *ExtractKnowledgeWorker) Work(
	ctx context.Context,
	job *river.Job[queue.ExtractKnowledgeDTO],
) error {
	doc, err := w.DocumentService.GetByID(ctx, job.Args.DocumentID)
	if err != nil {
		return fmt.Errorf("failed to fetch document %s: %w", job.Args.DocumentID, err)
	}

	chunks, err := w.ChunkRepo.ListChunksNeedingKnowledgeExtraction(ctx, job.Args.DocumentID)
	if err != nil {
		return fmt.Errorf(
			"failed to list chunks needing knowledge extraction for document %s: %w",
			job.Args.DocumentID,
			err,
		)
	}

	for i := range chunks {
		chunk := &chunks[i]

		extracted, err := w.Extractor.Extract(ctx, &data.ExtractionRequest{
			Text:          chunk.Text,
			DocumentTitle: doc.Title,
		})
		if err != nil {
			return fmt.Errorf("failed to extract knowledge for chunk %s: %w", chunk.ID, err)
		}

		now := time.Now()
		facts := make([]model.AtomicKnowledge, len(extracted))
		for j := range extracted {
			facts[j] = model.AtomicKnowledge{
				ID:              uuid.New(),
				CreatedAt:       now,
				UpdatedAt:       now,
				DocumentID:      job.Args.DocumentID,
				DocumentChunkID: chunk.ID,
				TruthTier:       extracted[j].TruthTier,
				Topics:          extracted[j].Topics,
				Subject:         extracted[j].Subject,
				Predicate:       extracted[j].Predicate,
				Object:          extracted[j].Object,
				Note:            extracted[j].Note,
			}
		}

		err = w.KnowledgeRepo.WithTx(ctx, func(tx data.AtomicKnowledgeRepoer) error {
			if err := tx.CreateBatch(ctx, facts); err != nil {
				return fmt.Errorf("failed to store extracted facts: %w", err)
			}
			return tx.MarkChunkKnowledgeExtracted(ctx, chunk.ID)
		})
		if err != nil {
			return fmt.Errorf(
				"failed to commit extraction results for chunk %s: %w",
				chunk.ID,
				err,
			)
		}
	}
	return nil
}
