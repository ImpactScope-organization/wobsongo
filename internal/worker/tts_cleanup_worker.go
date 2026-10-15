package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/riverqueue/river"
)

// ttsCleanupTimeout bounds one cleanup run.
const ttsCleanupTimeout = 5 * time.Minute

// TTSAudioCleanupWorker deletes synthesized voice replies older than
// queue.TTSAudioRetention from S3.
type TTSAudioCleanupWorker struct {
	river.WorkerDefaults[queue.TTSAudioCleanupJob]
	storage data.MediaStorageAdmin
	now     func() time.Time
}

// NewTTSAudioCleanupWorker creates a new TTSAudioCleanupWorker.
func NewTTSAudioCleanupWorker(storage data.MediaStorageAdmin) *TTSAudioCleanupWorker {
	return &TTSAudioCleanupWorker{storage: storage, now: time.Now}
}

func (w *TTSAudioCleanupWorker) Timeout(_ *river.Job[queue.TTSAudioCleanupJob]) time.Duration {
	return ttsCleanupTimeout
}

// Work lists the objects under the TTS prefix and removes the expired ones.
func (w *TTSAudioCleanupWorker) Work(
	ctx context.Context,
	_ *river.Job[queue.TTSAudioCleanupJob],
) error {
	prefix := data.ObjectPrefixForIntent(data.TTSAudioUploadIntent)
	objects, err := w.storage.ListObjects(ctx, prefix)
	if err != nil {
		return fmt.Errorf("failed to list synthesized audio: %w", err)
	}

	cutoff := w.now().Add(-queue.TTSAudioRetention)
	deleted, failed := 0, 0
	for _, obj := range objects {
		if !obj.LastModified.Before(cutoff) {
			continue
		}
		if err := w.storage.DeleteObject(ctx, obj.Key); err != nil {
			failed++
			log.Printf("[TTSAudioCleanupWorker] failed to delete %q: %v", obj.Key, err)
			continue
		}
		deleted++
	}

	log.Printf(
		"[TTSAudioCleanupWorker] scanned=%d deleted=%d failed=%d",
		len(objects), deleted, failed,
	)
	if failed > 0 {
		return fmt.Errorf("failed to delete %d of %d expired audio files", failed, deleted+failed)
	}
	return nil
}
