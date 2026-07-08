package worker

import (
	"context"
	"fmt"
	"log"
	"time"

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
}

// NewParseDocumentWorker is a constructor for ParseDocumentWorker.
func NewParseDocumentWorker(
	processor data.DocumentProcessor,
	mediaService *service.MediaService,
) *ParseDocumentWorker {
	return &ParseDocumentWorker{
		Processor:    processor,
		MediaService: mediaService,
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

	kept, dropped := filterNoiseChunks(result.Chunks)
	log.Printf(
		"[ParseDocumentWorker] document=%s title=%q page_count=%d chunks_kept=%d chunks_dropped=%d",
		job.Args.DocumentID, result.Title, result.PageCount, len(kept), dropped,
	)

	// TODO(document-ingestion pipeline, future sub-task): persist `kept` as
	// model.DocumentChunk rows (SequenceNumber = index in `kept`) once a real
	// DocumentChunkRepoer exists, then enqueue Job 2 (knowledge extraction)
	// inside the same transaction.
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
