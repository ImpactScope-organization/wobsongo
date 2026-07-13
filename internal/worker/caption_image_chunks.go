package worker

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
)

// captionImageChunksTimeout bounds how long the worker waits across all of
// a job's images. Generous relative to a single VLM call since a document
// can carry many images and calls run sequentially.
const captionImageChunksTimeout = 10 * time.Minute

// extensionContentTypes recovers an image's content type from its stored S3
// key's extension — the reverse of imageContentTypeExtensions
// (process_parsed_document.go), which chose that extension in the first place.
var extensionContentTypes = map[string]string{
	"png":  "image/png",
	"jpeg": contentTypeImageJPEG,
	"jpg":  contentTypeImageJPEG,
	"webp": "image/webp",
	"avif": "image/avif",
}

// contextualLayoutTypes are chunk types worth including as grounding text
// when captioning a nearby image — the parts of a document a human would
// actually read for context, excluding noise, tables, and other images.
var contextualLayoutTypes = map[model.LayoutType]bool{
	model.LayoutTypeTitle:         true,
	model.LayoutTypeSectionHeader: true,
	model.LayoutTypeParagraph:     true,
	model.LayoutTypeListItem:      true,
	model.LayoutTypeCaption:       true, // Docling's own figure/table caption — strongest signal
}

const (
	// surroundingPageWindow bounds how many pages away from an image a
	// context chunk may be and still count as "surrounding."
	surroundingPageWindow = 1

	// surroundingTextBudget caps how many characters of gathered context
	// are sent per image, to bound prompt size/cost.
	surroundingTextBudget = 2000
)

// CaptionImageChunksWorker is a River worker that generates and stores a
// caption for each image/chart chunk enqueued by ProcessParsedDocumentWorker.
// Kept as its own job (not inline in that worker) so a slow/costly/
// rate-limited VLM call never fails or retries chunk storage itself.
type CaptionImageChunksWorker struct {
	// Embedding River's default worker behavior for the specific DTO.
	river.WorkerDefaults[queue.CaptionImageChunksDTO]
	// RawStore fetches each chunk's image bytes back from S3.
	RawStore data.RawObjectStore
	// ChunkRepo reads chunks and saves their generated captions.
	ChunkRepo data.DocumentChunkRepoer
	// DocumentService fetches the parent document's title, for grounding.
	DocumentService *service.DocumentService
	// Captioner generates the caption text for an image.
	Captioner data.ImageCaptioner
}

// NewCaptionImageChunksWorker is a constructor for CaptionImageChunksWorker.
func NewCaptionImageChunksWorker(
	rawStore data.RawObjectStore,
	chunkRepo data.DocumentChunkRepoer,
	documentService *service.DocumentService,
	captioner data.ImageCaptioner,
) *CaptionImageChunksWorker {
	return &CaptionImageChunksWorker{
		RawStore:        rawStore,
		ChunkRepo:       chunkRepo,
		DocumentService: documentService,
		Captioner:       captioner,
	}
}

// Timeout overrides River's default job timeout.
func (w *CaptionImageChunksWorker) Timeout(*river.Job[queue.CaptionImageChunksDTO]) time.Duration {
	return captionImageChunksTimeout
}

// Work is the main method that gets called when a job is dequeued.
func (w *CaptionImageChunksWorker) Work(
	ctx context.Context,
	job *river.Job[queue.CaptionImageChunksDTO],
) error {
	doc, err := w.DocumentService.GetByID(ctx, job.Args.DocumentID)
	if err != nil {
		return fmt.Errorf("failed to fetch document %s: %w", job.Args.DocumentID, err)
	}

	chunks, err := w.ChunkRepo.ListByDocumentID(ctx, job.Args.DocumentID)
	if err != nil {
		return fmt.Errorf(
			"failed to list chunks for document %s: %w",
			job.Args.DocumentID,
			err,
		)
	}

	for _, chunkID := range job.Args.ChunkIDs {
		idx := indexOfChunk(chunks, chunkID)
		if idx < 0 {
			return fmt.Errorf("chunk %s not found for document %s", chunkID, job.Args.DocumentID)
		}
		chunk := &chunks[idx]

		if chunk.Text != "" {
			// Already captioned — a prior attempt at this job got this far
			// before failing on a later chunk; don't re-pay for the VLM call.
			continue
		}

		rc, err := w.RawStore.GetObject(ctx, chunk.AssetURL)
		if err != nil {
			return fmt.Errorf("failed to fetch image for chunk %s: %w", chunkID, err)
		}
		imageBytes, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return fmt.Errorf("failed to read image for chunk %s: %w", chunkID, err)
		}

		caption, err := w.Captioner.Caption(ctx, &data.CaptionRequest{
			ImageBytes:      imageBytes,
			ContentType:     contentTypeFromKey(chunk.AssetURL),
			DocumentTitle:   doc.Title,
			Page:            chunk.Page,
			SurroundingText: gatherSurroundingText(chunks, chunk),
		})
		if err != nil {
			return fmt.Errorf("failed to caption image for chunk %s: %w", chunkID, err)
		}

		chunk.Text = caption
		chunk.UpdatedAt = time.Now()
		if err := w.ChunkRepo.Update(ctx, chunk); err != nil {
			return fmt.Errorf("failed to save caption for chunk %s: %w", chunkID, err)
		}
	}
	return nil
}

// indexOfChunk returns the index of the chunk with the given ID in chunks,
// or -1 if not found.
func indexOfChunk(chunks []model.DocumentChunk, id uuid.UUID) int {
	for i := range chunks {
		if chunks[i].ID == id {
			return i
		}
	}
	return -1
}

// gatherSurroundingText collects nearby body text/caption chunks (within
// +/-surroundingPageWindow pages of target) to ground a VLM caption request.
//
// Windows by Page, not SequenceNumber adjacency: mapDoclingDocument
// concatenates all texts, then all tables, then all pictures (a documented
// simplification, not true reading order), so a chunk's sequence-neighbors
// are typically other images/tables, not nearby body text — Page is the
// reliable signal instead. Docling's own LayoutTypeCaption chunks (if any)
// are gathered first, since they're the strongest available signal — this
// way they survive the character budget even if they sit late in `all`'s
// concatenated order.
func gatherSurroundingText(all []model.DocumentChunk, target *model.DocumentChunk) string {
	inWindow := func(c *model.DocumentChunk) bool {
		return c.ID != target.ID && contextualLayoutTypes[c.LayoutType] && c.Text != "" &&
			c.Page >= target.Page-surroundingPageWindow && c.Page <= target.Page+surroundingPageWindow
	}

	var b strings.Builder
	write := func(text string) bool {
		if b.Len()+len(text)+1 > surroundingTextBudget {
			return false
		}
		b.WriteString(text)
		b.WriteString("\n")
		return true
	}

	for i := range all {
		c := &all[i]
		if inWindow(c) && c.LayoutType == model.LayoutTypeCaption {
			if !write(c.Text) {
				return strings.TrimSpace(b.String())
			}
		}
	}
	for i := range all {
		c := &all[i]
		if inWindow(c) && c.LayoutType != model.LayoutTypeCaption {
			if !write(c.Text) {
				break
			}
		}
	}
	return strings.TrimSpace(b.String())
}

// contentTypeFromKey recovers an image's content type from its S3 key's extension.
func contentTypeFromKey(key string) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(key)), ".")
	return extensionContentTypes[ext]
}
