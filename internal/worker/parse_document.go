package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

// parseDocumentTimeout bounds how long the worker waits for Docling to parse
// a document. Docling can take minutes on large PDFs; blocking a River
// worker goroutine for that long is cheap in Go.
const parseDocumentTimeout = 5 * time.Minute

// noiseLayoutTypes are Docling layout types dropped before persistence — never
// embedded, never shown to an LLM. Per the ingestion pipeline design: filter
// at ingestion time, not query time.
var noiseLayoutTypes = map[model.LayoutType]bool{
	model.LayoutTypePageHeader:    true,
	model.LayoutTypePageFooter:    true,
	model.LayoutTypeDocumentIndex: true,
}

// ParseDocumentWorker is a River worker that parses an ingested document via Docling.
type ParseDocumentWorker struct {
	// Embedding River's default worker behavior for the specific DTO.
	river.WorkerDefaults[queue.ParseDocumentDTO]
	// Processor calls out to the external document-parsing service (Docling).
	Processor data.DocumentProcessor
	// MediaService presigns the document's file key into a URL Docling can fetch.
	MediaService *service.MediaService
	// ChunkRepo stores the chunks that survive filtering.
	ChunkRepo data.DocumentChunkRepoer
	// DocumentService backfills the document's page count once Docling has
	// actually parsed it (the document is created with page_count=0 up front).
	DocumentService *service.DocumentService
}

// NewParseDocumentWorker is a constructor for ParseDocumentWorker.
func NewParseDocumentWorker(
	processor data.DocumentProcessor,
	mediaService *service.MediaService,
	chunkRepo data.DocumentChunkRepoer,
	documentService *service.DocumentService,
) *ParseDocumentWorker {
	return &ParseDocumentWorker{
		Processor:       processor,
		MediaService:    mediaService,
		ChunkRepo:       chunkRepo,
		DocumentService: documentService,
	}
}

// Timeout overrides River's default job timeout to accommodate slow Docling parses.
func (w *ParseDocumentWorker) Timeout(*river.Job[queue.ParseDocumentDTO]) time.Duration {
	return parseDocumentTimeout
}

// Work is the main method that gets called when a job is dequeued.
func (w *ParseDocumentWorker) Work(
	ctx context.Context,
	job *river.Job[queue.ParseDocumentDTO],
) error {
	// ttl=0 lets MediaService apply its own default (900s) — plenty, since
	// Docling fetches the file once up front, not throughout the whole parse.
	documentURL, err := w.MediaService.GetPresignedGETURL(ctx, job.Args.FileKey, 0)
	if err != nil {
		return fmt.Errorf("failed to presign document %s for docling: %w", job.Args.DocumentID, err)
	}

	result, err := w.Processor.ProcessFromURL(ctx, documentURL)
	if err != nil {
		return fmt.Errorf("failed to process document %s via docling: %w", job.Args.DocumentID, err)
	}

	if err := w.DocumentService.UpdateAfterParse(
		ctx,
		job.Args.DocumentID,
		result.PageCount,
		result.Title,
	); err != nil {
		return fmt.Errorf(
			"failed to update document %s after parsing: %w",
			job.Args.DocumentID,
			err,
		)
	}

	kept, dropped := filterNoiseChunks(result.Chunks)
	log.Printf(
		"[ParseDocumentWorker] document=%s title=%q page_count=%d chunks_kept=%d chunks_dropped=%d",
		job.Args.DocumentID, result.Title, result.PageCount, len(kept), dropped,
	)

	var stored int
	err = w.ChunkRepo.WithTx(ctx, func(tx data.DocumentChunkRepoer) error {
		now := time.Now()
		toStore := make([]model.DocumentChunk, 0, len(kept))
		for i, c := range kept {
			chunk := model.DocumentChunk{
				ID:             uuid.New(),
				CreatedAt:      now,
				UpdatedAt:      now,
				DocumentID:     job.Args.DocumentID,
				SequenceNumber: i,
				ParsedChunk:    c,
			}
			ok, err := tx.ShouldBeStored(ctx, chunk)
			if err != nil {
				return fmt.Errorf("failed to evaluate chunk %d for storage: %w", i, err)
			}
			if ok {
				toStore = append(toStore, chunk)
			}
		}
		stored = len(toStore)
		return tx.CreateBatch(ctx, toStore)
		// TODO(future sub-task): enqueue Job 2 (knowledge extraction) here once
		// its queue DTO exists, so it commits atomically with these chunks.
	})
	if err != nil {
		return fmt.Errorf("failed to store chunks for document %s: %w", job.Args.DocumentID, err)
	}

	log.Printf(
		"[ParseDocumentWorker] document=%s stored=%d/%d chunks",
		job.Args.DocumentID, stored, len(kept),
	)
	return nil
}

// filterNoiseChunks drops chunks whose layout type is considered noise
// (page headers/footers, table of contents), returning the kept chunks and
// a count of how many were dropped.
func filterNoiseChunks(chunks []model.ParsedChunk) (kept []model.ParsedChunk, dropped int) {
	for _, c := range chunks {
		if noiseLayoutTypes[c.LayoutType] {
			dropped++
			continue
		}
		kept = append(kept, c)
	}
	return kept, dropped
}
