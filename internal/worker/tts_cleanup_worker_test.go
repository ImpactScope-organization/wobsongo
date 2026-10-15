package worker

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/riverqueue/river"
)

// cleanupNow is the fixed "current time" the cleanup tests run at.
var cleanupNow = time.Date(2026, time.October, 15, 12, 0, 0, 0, time.UTC)

// stubCleanupStorage is a hand-rolled data.MediaStorageAdmin.
type stubCleanupStorage struct {
	objects   []data.S3ObjectInfo
	listErr   error
	deleteErr map[string]error
	gotPrefix string
	deleted   []string
}

func (s *stubCleanupStorage) ListObjects(
	_ context.Context, prefix string,
) ([]data.S3ObjectInfo, error) {
	s.gotPrefix = prefix
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.objects, nil
}

func (s *stubCleanupStorage) DeleteObject(_ context.Context, key string) error {
	if err := s.deleteErr[key]; err != nil {
		return err
	}
	s.deleted = append(s.deleted, key)
	return nil
}

func newCleanupWorker(storage *stubCleanupStorage) *TTSAudioCleanupWorker {
	w := NewTTSAudioCleanupWorker(storage)
	w.now = func() time.Time { return cleanupNow }
	return w
}

// ttsObject returns a stored reply that was last modified age ago.
func ttsObject(name string, age time.Duration) data.S3ObjectInfo {
	return data.S3ObjectInfo{
		Key:          "tts_audios/" + name + ".ogg",
		LastModified: cleanupNow.Add(-age),
	}
}

func runCleanup(w *TTSAudioCleanupWorker) error {
	return w.Work(context.Background(), &river.Job[queue.TTSAudioCleanupJob]{})
}

func TestTTSAudioCleanupWorker_Timeout(t *testing.T) {
	w := newCleanupWorker(&stubCleanupStorage{})

	if got := w.Timeout(nil); got != ttsCleanupTimeout {
		t.Errorf("expected timeout %s, got %s", ttsCleanupTimeout, got)
	}
}

func TestTTSAudioCleanupWorker_Work_DeletesOnlyExpiredAudio(t *testing.T) {
	storage := &stubCleanupStorage{
		objects: []data.S3ObjectInfo{
			ttsObject("old-1", queue.TTSAudioRetention+time.Hour),
			ttsObject("fresh", time.Hour),
			ttsObject("old-2", 3*queue.TTSAudioRetention),
			// Exactly at the retention limit is not expired yet.
			ttsObject("boundary", queue.TTSAudioRetention),
		},
	}

	if err := runCleanup(newCleanupWorker(storage)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"tts_audios/old-1.ogg", "tts_audios/old-2.ogg"}
	if !slices.Equal(storage.deleted, want) {
		t.Errorf("expected %v to be deleted, got %v", want, storage.deleted)
	}
}

func TestTTSAudioCleanupWorker_Work_OnlyScansTTSPrefix(t *testing.T) {
	storage := &stubCleanupStorage{}

	if err := runCleanup(newCleanupWorker(storage)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := data.ObjectPrefixForIntent(data.TTSAudioUploadIntent)
	if storage.gotPrefix != want {
		t.Errorf("expected the %q prefix to be listed, got %q", want, storage.gotPrefix)
	}
}

func TestTTSAudioCleanupWorker_Work_NothingExpired(t *testing.T) {
	storage := &stubCleanupStorage{
		objects: []data.S3ObjectInfo{ttsObject("fresh", time.Minute)},
	}

	if err := runCleanup(newCleanupWorker(storage)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(storage.deleted) != 0 {
		t.Errorf("nothing should be deleted, got %v", storage.deleted)
	}
}

func TestTTSAudioCleanupWorker_Work_ListFailure(t *testing.T) {
	listErr := errors.New("bucket unavailable")
	storage := &stubCleanupStorage{listErr: listErr}

	err := runCleanup(newCleanupWorker(storage))

	if !errors.Is(err, listErr) {
		t.Fatalf("expected the list error to be returned, got %v", err)
	}
	if len(storage.deleted) != 0 {
		t.Errorf("nothing should be deleted, got %v", storage.deleted)
	}
}

// One object that cannot be deleted must not stop the rest of the run.
func TestTTSAudioCleanupWorker_Work_DeleteFailureDoesNotStopTheRun(t *testing.T) {
	old := queue.TTSAudioRetention + time.Hour
	storage := &stubCleanupStorage{
		objects: []data.S3ObjectInfo{ttsObject("stuck", old), ttsObject("ok", old)},
		deleteErr: map[string]error{
			"tts_audios/stuck.ogg": errors.New("access denied"),
		},
	}

	err := runCleanup(newCleanupWorker(storage))

	if err == nil {
		t.Fatal("expected an error because one delete failed")
	}
	want := []string{"tts_audios/ok.ogg"}
	if !slices.Equal(storage.deleted, want) {
		t.Errorf("expected %v to be deleted, got %v", want, storage.deleted)
	}
}
