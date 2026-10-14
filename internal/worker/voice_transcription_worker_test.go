package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/impactscope-organization/wobsongo/internal/validation"
	"github.com/riverqueue/river"
)

// stubVoiceStore is a hand-rolled data.VoiceObjectStore.
type stubVoiceStore struct {
	puts    map[string][]byte
	putErr  error
	deleted []string
}

func (s *stubVoiceStore) PutObject(
	_ context.Context, key string, r io.Reader, _ int64, _ string,
) error {
	if s.putErr != nil {
		return s.putErr
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.puts[key] = body
	return nil
}

func (s *stubVoiceStore) GetObject(context.Context, string) (io.ReadCloser, error) {
	return nil, errors.New("GetObject is not used by the voice worker")
}

func (s *stubVoiceStore) DeleteObject(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return nil
}

// stubPresigner is a hand-rolled presigner returning a fake URL for a key.
type stubPresigner struct {
	keys []string
	err  error
}

func (s *stubPresigner) GetPresignedGETURL(_ context.Context, key string, _ int64) (string, error) {
	s.keys = append(s.keys, key)
	if s.err != nil {
		return "", s.err
	}
	return "https://s3.test/" + key + "?sig=1", nil
}

// stubTranscriber is a hand-rolled data.SpeechTranscriber.
type stubTranscriber struct {
	transcript string
	err        error
	gotURL     string
	gotFormat  string
}

func (s *stubTranscriber) TranscribeURL(
	_ context.Context,
	audioURL, format string,
) (string, error) {
	s.gotURL = audioURL
	s.gotFormat = format
	return s.transcript, s.err
}

// botCallback is the part of the extract-done callback the tests check.
type botCallback struct {
	JobID  string `json:"jobId"`
	Status string `json:"status"`
	Error  string `json:"error"`
}

type botCallbacks struct {
	mu    sync.Mutex
	calls []botCallback
}

func (b *botCallbacks) snapshot() []botCallback {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]botCallback(nil), b.calls...)
}

// newBotRecorder starts a fake wa-bot and returns a BotClient pointed at it.
func newBotRecorder(t *testing.T) (*external.BotClient, *botCallbacks) {
	t.Helper()
	rec := &botCallbacks{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cb botCallback
		if err := json.NewDecoder(r.Body).Decode(&cb); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		rec.mu.Lock()
		rec.calls = append(rec.calls, cb)
		rec.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)
	return external.NewBotClient(srv.URL, "callback-psk", "control-psk"), rec
}

type voiceFixture struct {
	worker      *VoiceTranscriptionWorker
	downloader  *external.MockVoiceMediaClient
	store       *stubVoiceStore
	presigner   *stubPresigner
	transcriber *stubTranscriber
	bot         *botCallbacks
	appended    []string
	turns       []queue.AgentTurnJob
}

func newVoiceFixture(t *testing.T) *voiceFixture {
	t.Helper()
	f := &voiceFixture{
		downloader:  external.NewMockVoiceMediaClient(),
		store:       &stubVoiceStore{puts: map[string][]byte{}},
		presigner:   &stubPresigner{},
		transcriber: &stubTranscriber{transcript: "bonjour"},
	}
	f.downloader.Audio = []byte("OggS-test-audio")

	repo := &mockrepo.ConversationRepoerMock{}
	repo.AppendMessageFunc = func(
		_ context.Context, jid string, role model.ConversationRole, content string,
	) error {
		f.appended = append(f.appended, fmt.Sprintf("%s|%v|%s", jid, role, content))
		return nil
	}
	repo.EnqueueAgentTurnFunc = func(_ context.Context, payload queue.AgentTurnJob) error {
		f.turns = append(f.turns, payload)
		return nil
	}

	botClient, bot := newBotRecorder(t)
	f.bot = bot
	f.worker = NewVoiceTranscriptionWorker(
		f.downloader,
		f.store,
		f.presigner,
		f.transcriber,
		service.NewConversationService(repo),
		botClient,
	)
	return f
}

func newVoiceJob(mimeType string) *river.Job[queue.VoiceTranscriptionJob] {
	return &river.Job[queue.VoiceTranscriptionJob]{
		Args: queue.VoiceTranscriptionJob{
			ExtractionID: "job-1",
			JID:          "22500000000",
			Channel:      "whatsapp",
			MediaRef:     "https://api.twilio.com/media/1",
			MimeType:     mimeType,
		},
	}
}

// requireSingleFailure checks that the bot was told, once, that job-1 failed
// with the given user-facing message.
func requireSingleFailure(t *testing.T, bot *botCallbacks, wantMessage string) {
	t.Helper()
	calls := bot.snapshot()
	if len(calls) != 1 {
		t.Fatalf("expected 1 bot callback, got %d: %+v", len(calls), calls)
	}
	if calls[0].JobID != "job-1" || calls[0].Status != "failed" || calls[0].Error != wantMessage {
		t.Errorf("unexpected bot callback %+v, want failed with %q", calls[0], wantMessage)
	}
}

func TestVoiceTranscriptionWorker_Work_Success(t *testing.T) {
	f := newVoiceFixture(t)

	if err := f.worker.Work(t.Context(), newVoiceJob("audio/ogg; codecs=opus")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(f.store.puts) != 1 {
		t.Fatalf("expected 1 stored object, got %d", len(f.store.puts))
	}
	var key string
	for k, body := range f.store.puts {
		key = k
		if string(body) != "OggS-test-audio" {
			t.Errorf("stored audio = %q, want the downloaded bytes", body)
		}
	}
	if !strings.HasPrefix(key, "voice_notes/") || !strings.HasSuffix(key, ".ogg") {
		t.Errorf("unexpected object key %q", key)
	}
	if len(f.store.deleted) != 1 || f.store.deleted[0] != key {
		t.Errorf("expected the temporary audio %q to be deleted, got %v", key, f.store.deleted)
	}
	if !strings.Contains(f.transcriber.gotURL, key) || f.transcriber.gotFormat != "ogg" {
		t.Errorf("ASR got url %q format %q", f.transcriber.gotURL, f.transcriber.gotFormat)
	}

	wantMessage := fmt.Sprintf("22500000000|%v|bonjour", model.ConversationRoleUser)
	if len(f.appended) != 1 || f.appended[0] != wantMessage {
		t.Errorf("stored messages = %v, want [%s]", f.appended, wantMessage)
	}
	if len(f.turns) != 1 ||
		f.turns[0].JID != "22500000000" ||
		f.turns[0].ExtractionID != "job-1" ||
		f.turns[0].UserText != "bonjour" {
		t.Errorf("unexpected agent turns: %+v", f.turns)
	}
	if calls := f.bot.snapshot(); len(calls) != 0 {
		t.Errorf("expected no bot callback on success, got %+v", calls)
	}
}

func TestVoiceTranscriptionWorker_Work_DownloadFailures(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantMessage string
	}{
		{
			"too large",
			fmt.Errorf("failed: %w", data.ErrVoiceTooLarge),
			voiceTooLargeMessage,
		},
		{"empty", data.ErrVoiceEmpty, voiceUnclearMessage},
		{"other error", errors.New("twilio returned 500"), voiceFailureMessage},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newVoiceFixture(t)
			f.downloader.Err = tt.err

			if err := f.worker.Work(t.Context(), newVoiceJob("audio/ogg")); err == nil {
				t.Fatal("expected an error")
			}

			requireSingleFailure(t, f.bot, tt.wantMessage)
			if len(f.store.puts) != 0 {
				t.Errorf("nothing should be stored, got %d objects", len(f.store.puts))
			}
			if len(f.turns) != 0 {
				t.Errorf("no agent turn should start, got %+v", f.turns)
			}
		})
	}
}

func TestVoiceTranscriptionWorker_Work_UnsupportedMimeType(t *testing.T) {
	f := newVoiceFixture(t)

	if err := f.worker.Work(t.Context(), newVoiceJob("video/mp4")); err == nil {
		t.Fatal("expected an error")
	}

	requireSingleFailure(t, f.bot, voiceFailureMessage)
	if len(f.store.puts) != 0 {
		t.Errorf("nothing should be stored, got %d objects", len(f.store.puts))
	}
}

func TestVoiceTranscriptionWorker_Work_StoreFailure(t *testing.T) {
	f := newVoiceFixture(t)
	f.store.putErr = errors.New("bucket unavailable")

	if err := f.worker.Work(t.Context(), newVoiceJob("audio/ogg")); err == nil {
		t.Fatal("expected an error")
	}

	requireSingleFailure(t, f.bot, voiceFailureMessage)
	if len(f.store.deleted) != 0 {
		t.Errorf("nothing was stored, so nothing should be deleted, got %v", f.store.deleted)
	}
}

func TestVoiceTranscriptionWorker_Work_ASRFailureStillDeletesAudio(t *testing.T) {
	f := newVoiceFixture(t)
	f.transcriber.err = errors.New("asr endpoint returned 500")

	if err := f.worker.Work(t.Context(), newVoiceJob("audio/ogg")); err == nil {
		t.Fatal("expected an error")
	}

	requireSingleFailure(t, f.bot, voiceFailureMessage)
	if len(f.store.puts) != 1 || len(f.store.deleted) != 1 {
		t.Errorf("expected the stored audio to be cleaned up, puts=%d deleted=%v",
			len(f.store.puts), f.store.deleted)
	}
	if len(f.appended) != 0 || len(f.turns) != 0 {
		t.Errorf("no message or agent turn expected, got %v %+v", f.appended, f.turns)
	}
}

func TestVoiceTranscriptionWorker_Work_EmptyTranscript(t *testing.T) {
	f := newVoiceFixture(t)
	f.transcriber.transcript = ""

	if err := f.worker.Work(t.Context(), newVoiceJob("audio/ogg")); err != nil {
		t.Fatalf("an unclear voice message is not a job error, got %v", err)
	}

	requireSingleFailure(t, f.bot, voiceUnclearMessage)
	if len(f.store.deleted) != 1 {
		t.Errorf("expected the stored audio to be cleaned up, got %v", f.store.deleted)
	}
	if len(f.appended) != 0 || len(f.turns) != 0 {
		t.Errorf("no message or agent turn expected, got %v %+v", f.appended, f.turns)
	}
}

// The presigner only accepts <prefix>/<sha256>.<ext> keys, so every key the
// workers generate must pass the same validation.
func TestObjectKeys_PassS3Validation(t *testing.T) {
	keys := []string{ttsObjectKey("job-1")}
	for _, mimeType := range []string{
		"audio/ogg", "audio/mpeg", "audio/mp4", "audio/aac",
		"audio/amr", "audio/webm", "audio/wav",
	} {
		format, ok := data.VoiceFormatFromMime(mimeType)
		if !ok {
			t.Fatalf("%s should be an accepted voice mime type", mimeType)
		}
		keys = append(keys, voiceObjectKey("job-1", format))
	}

	for _, key := range keys {
		if !validation.ValidateS3PrefixAndFile(key) {
			t.Errorf("key %q is rejected by validation.ValidateS3PrefixAndFile", key)
		}
	}
}
