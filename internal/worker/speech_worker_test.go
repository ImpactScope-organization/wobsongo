package worker

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/service"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// speechStubSynth is a hand-rolled data.SpeechSynthesizer.
type speechStubSynth struct {
	audio   []byte
	err     error
	calls   int
	gotText string
	gotLang string
}

func (s *speechStubSynth) Synthesize(_ context.Context, text, language string) ([]byte, error) {
	s.calls++
	s.gotText = text
	s.gotLang = language
	return s.audio, s.err
}

// ttsBotCallback is the part of the tts-done callback the tests check.
type ttsBotCallback struct {
	JobID    string `json:"jobId"`
	Status   string `json:"status"`
	AudioURL string `json:"audioUrl"`
}

type ttsBotRecorder struct {
	mu    sync.Mutex
	calls []ttsBotCallback
	paths []string
}

func (b *ttsBotRecorder) snapshot() []ttsBotCallback {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]ttsBotCallback(nil), b.calls...)
}

// newTTSBotRecorder starts a fake wa-bot whose /callback/tts-done endpoint
// answers with status, and returns a BotClient pointed at it.
func newTTSBotRecorder(t *testing.T, status int) (*external.BotClient, *ttsBotRecorder) {
	t.Helper()
	rec := &ttsBotRecorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cb ttsBotCallback
		if err := json.NewDecoder(r.Body).Decode(&cb); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		rec.mu.Lock()
		rec.calls = append(rec.calls, cb)
		rec.paths = append(rec.paths, r.URL.Path)
		rec.mu.Unlock()
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return external.NewBotClient(srv.URL, "callback-psk", "control-psk"), rec
}

type speechFixture struct {
	worker    *SpeechWorker
	synth     *speechStubSynth
	store     *stubVoiceStore
	presigner *stubPresigner
	bot       *ttsBotRecorder
}

func newSpeechFixture(t *testing.T, botStatus int) *speechFixture {
	t.Helper()
	f := &speechFixture{
		synth:     &speechStubSynth{audio: []byte("OggS-reply")},
		store:     &stubVoiceStore{puts: map[string][]byte{}},
		presigner: &stubPresigner{},
	}
	botClient, bot := newTTSBotRecorder(t, botStatus)
	f.bot = bot
	f.worker = NewSpeechWorker(
		service.NewSpeechService(nil, f.synth, nil),
		botClient,
		f.store,
		f.presigner,
	)
	return f
}

func newSpeechJob(createdAt time.Time, text string) *river.Job[queue.SpeechJob] {
	return &river.Job[queue.SpeechJob]{
		JobRow: &rivertype.JobRow{CreatedAt: createdAt},
		Args:   queue.SpeechJob{JobID: "job-1", Text: text, Language: "fr"},
	}
}

// requireSingleTTSFailure checks that the bot was told, once, that job-1 failed.
func requireSingleTTSFailure(t *testing.T, bot *ttsBotRecorder) {
	t.Helper()
	calls := bot.snapshot()
	if len(calls) != 1 {
		t.Fatalf("expected 1 bot callback, got %d: %+v", len(calls), calls)
	}
	if calls[0].JobID != "job-1" || calls[0].Status != ttsStatusFailed || calls[0].AudioURL != "" {
		t.Errorf("unexpected bot callback %+v, want job-1 failed without audio", calls[0])
	}
}

func TestSpeechWorker_Timeout(t *testing.T) {
	f := newSpeechFixture(t, http.StatusNoContent)

	if got := f.worker.Timeout(nil); got != queue.SpeechJobTTL {
		t.Errorf("expected timeout %s, got %s", queue.SpeechJobTTL, got)
	}
}

func TestSpeechWorker_Work_Success(t *testing.T) {
	f := newSpeechFixture(t, http.StatusNoContent)

	if err := f.worker.Work(t.Context(), newSpeechJob(time.Now(), "Bonjour")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if f.synth.gotText != "Bonjour." || f.synth.gotLang != "fr" {
		t.Errorf("unexpected synthesizer input %q (%s)", f.synth.gotText, f.synth.gotLang)
	}

	key := ttsObjectKey("job-1")
	if !strings.HasPrefix(key, "tts_audios/") || !strings.HasSuffix(key, ".ogg") {
		t.Fatalf("unexpected object key %q", key)
	}
	if got := f.store.puts[key]; string(got) != "OggS-reply" || len(f.store.puts) != 1 {
		t.Errorf("expected the audio stored once under %q, got %v", key, f.store.puts)
	}
	if len(f.presigner.keys) != 1 || f.presigner.keys[0] != key {
		t.Errorf("expected %q to be presigned, got %v", key, f.presigner.keys)
	}

	calls := f.bot.snapshot()
	if len(calls) != 1 {
		t.Fatalf("expected 1 bot callback, got %d: %+v", len(calls), calls)
	}
	wantURL := "https://s3.test/" + key + "?sig=1"
	if calls[0].JobID != "job-1" || calls[0].Status != ttsStatusCompleted ||
		calls[0].AudioURL != wantURL {
		t.Errorf("unexpected bot callback %+v, want completed with %q", calls[0], wantURL)
	}
	if f.bot.paths[0] != "/callback/tts-done" {
		t.Errorf("expected the tts-done endpoint, got %q", f.bot.paths[0])
	}
}

func TestSpeechWorker_Work_ExpiredJobIsDroppedWithoutSynthesizing(t *testing.T) {
	f := newSpeechFixture(t, http.StatusNoContent)
	createdAt := time.Now().Add(-2 * queue.SpeechJobTTL)

	err := f.worker.Work(t.Context(), newSpeechJob(createdAt, "Bonjour"))

	if err == nil {
		t.Fatal("expected the expired job to be canceled with an error")
	}
	if f.synth.calls != 0 {
		t.Errorf("an expired job must not reach the synthesizer, got %d calls", f.synth.calls)
	}
	if len(f.store.puts) != 0 {
		t.Errorf("nothing should be stored, got %d objects", len(f.store.puts))
	}
	requireSingleTTSFailure(t, f.bot)
}

func TestSpeechWorker_Work_SynthesisFailure(t *testing.T) {
	f := newSpeechFixture(t, http.StatusNoContent)
	f.synth.err = errors.New("modal returned 500")

	err := f.worker.Work(t.Context(), newSpeechJob(time.Now(), "Bonjour"))

	if err == nil {
		t.Fatal("expected an error")
	}
	if len(f.store.puts) != 0 {
		t.Errorf("nothing should be stored, got %d objects", len(f.store.puts))
	}
	requireSingleTTSFailure(t, f.bot)
}

func TestSpeechWorker_Work_NothingToSpeak(t *testing.T) {
	f := newSpeechFixture(t, http.StatusNoContent)

	err := f.worker.Work(t.Context(), newSpeechJob(time.Now(), "🎉"))

	if !errors.Is(err, service.ErrSpeechEmptyText) {
		t.Fatalf("expected ErrSpeechEmptyText, got %v", err)
	}
	if f.synth.calls != 0 {
		t.Errorf("the synthesizer must not be called, got %d calls", f.synth.calls)
	}
	requireSingleTTSFailure(t, f.bot)
}

func TestSpeechWorker_Work_StoreFailure(t *testing.T) {
	f := newSpeechFixture(t, http.StatusNoContent)
	f.store.putErr = errors.New("bucket unavailable")

	err := f.worker.Work(t.Context(), newSpeechJob(time.Now(), "Bonjour"))

	if err == nil {
		t.Fatal("expected an error")
	}
	if len(f.presigner.keys) != 0 {
		t.Errorf("nothing was stored, so nothing should be presigned, got %v", f.presigner.keys)
	}
	requireSingleTTSFailure(t, f.bot)
}

func TestSpeechWorker_Work_PresignFailure(t *testing.T) {
	f := newSpeechFixture(t, http.StatusNoContent)
	f.presigner.err = errors.New("cannot sign")

	err := f.worker.Work(t.Context(), newSpeechJob(time.Now(), "Bonjour"))

	if err == nil {
		t.Fatal("expected an error")
	}
	requireSingleTTSFailure(t, f.bot)
}

// The audio is already stored when the callback fails, and the job's own
// result does not depend on wa-bot being reachable.
func TestSpeechWorker_Work_BotNotifyFailureDoesNotFailTheJob(t *testing.T) {
	f := newSpeechFixture(t, http.StatusInternalServerError)

	if err := f.worker.Work(t.Context(), newSpeechJob(time.Now(), "Bonjour")); err != nil {
		t.Fatalf("a failed callback must not fail the job, got %v", err)
	}

	calls := f.bot.snapshot()
	if len(calls) != 1 || calls[0].Status != ttsStatusCompleted {
		t.Errorf("expected one completed callback attempt, got %+v", calls)
	}
}

func TestTTSObjectKey(t *testing.T) {
	key := ttsObjectKey("job-1")

	if key != ttsObjectKey("job-1") {
		t.Error("the key must be deterministic for a given job")
	}
	if key == ttsObjectKey("job-2") {
		t.Error("different jobs must not share a key")
	}
	if !strings.HasPrefix(key, data.ObjectPrefixForIntent(data.TTSAudioUploadIntent)) {
		t.Errorf("key %q is not under the tts audio prefix", key)
	}
}
