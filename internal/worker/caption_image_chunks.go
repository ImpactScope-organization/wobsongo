package worker

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/queue"
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
	// Captioner generates the caption text for an image.
	Captioner data.ImageCaptioner
}

// NewCaptionImageChunksWorker is a constructor for CaptionImageChunksWorker.
func NewCaptionImageChunksWorker(
	rawStore data.RawObjectStore,
	chunkRepo data.DocumentChunkRepoer,
	captioner data.ImageCaptioner,
) *CaptionImageChunksWorker {
	return &CaptionImageChunksWorker{
		RawStore:  rawStore,
		ChunkRepo: chunkRepo,
		Captioner: captioner,
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
	for _, chunkID := range job.Args.ChunkIDs {
		chunk, err := w.ChunkRepo.GetByID(ctx, chunkID)
		if err != nil {
			return fmt.Errorf("failed to fetch chunk %s: %w", chunkID, err)
		}
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

		caption, err := w.Captioner.Caption(ctx, imageBytes, contentTypeFromKey(chunk.AssetURL))
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

// contentTypeFromKey recovers an image's content type from its S3 key's extension.
func contentTypeFromKey(key string) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(key)), ".")
	return extensionContentTypes[ext]
}
