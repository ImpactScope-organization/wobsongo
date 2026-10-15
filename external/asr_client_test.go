package external

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newASRTestClient starts a fake Modal ASR endpoint and returns a client
// pointed at it.
func newASRTestClient(t *testing.T, handler http.HandlerFunc) *ASRClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewASRClient(srv.URL, "Whisper Small", "french")
}

func TestASRClient_TranscribeURL_Success(t *testing.T) {
	var (
		gotMethod string
		gotType   string
		gotBody   asrRequest
	)
	client := newASRTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"transcript":"  bonjour tout le monde  "}`))
	})

	transcript, err := client.TranscribeURL(
		context.Background(), "https://s3.test/voice_notes/a.ogg?sig=1", "ogg",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if transcript != "bonjour tout le monde" {
		t.Errorf("expected a trimmed transcript, got %q", transcript)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotType != "application/json" {
		t.Errorf("expected a JSON content type, got %q", gotType)
	}
	want := asrRequest{
		Model:       "Whisper Small",
		SourceLang:  "french",
		AudioURL:    "https://s3.test/voice_notes/a.ogg?sig=1",
		AudioFormat: "ogg",
	}
	if gotBody != want {
		t.Errorf("unexpected request body %+v, want %+v", gotBody, want)
	}
}

func TestASRClient_TranscribeURL_EmptyTranscriptIsNotAnError(t *testing.T) {
	client := newASRTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"transcript":"   "}`))
	})

	transcript, err := client.TranscribeURL(context.Background(), "https://s3.test/a.ogg", "ogg")
	if err != nil {
		t.Fatalf("no speech recognized is not an error, got %v", err)
	}
	if transcript != "" {
		t.Errorf("expected an empty transcript, got %q", transcript)
	}
}

func TestASRClient_TranscribeURL_ApplicationError(t *testing.T) {
	client := newASRTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":"unsupported audio"}`))
	})

	_, err := client.TranscribeURL(context.Background(), "https://s3.test/a.ogg", "ogg")

	if err == nil || !strings.Contains(err.Error(), "unsupported audio") {
		t.Fatalf("expected the application error to surface, got %v", err)
	}
}

func TestASRClient_TranscribeURL_NonOKStatus(t *testing.T) {
	client := newASRTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(strings.Repeat("x", 2000)))
	})

	_, err := client.TranscribeURL(context.Background(), "https://s3.test/a.ogg", "ogg")

	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention the status code, got %q", err.Error())
	}
	if len(err.Error()) > asrErrorSnippetBytes+100 {
		t.Errorf("error message is not truncated, len=%d", len(err.Error()))
	}
}

func TestASRClient_TranscribeURL_InvalidJSON(t *testing.T) {
	client := newASRTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>not json</html>"))
	})

	_, err := client.TranscribeURL(context.Background(), "https://s3.test/a.ogg", "ogg")

	if err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("expected a decode error, got %v", err)
	}
}

func TestASRClient_TranscribeURL_ServerUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	client := NewASRClient(srv.URL, "Whisper Small", "auto")
	srv.Close()

	_, err := client.TranscribeURL(context.Background(), "https://s3.test/a.ogg", "ogg")

	if err == nil || !strings.Contains(err.Error(), "asr request failed") {
		t.Fatalf("expected a request failure, got %v", err)
	}
}
