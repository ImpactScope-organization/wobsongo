package external

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTTSTestClient starts a fake Modal TTS endpoint and returns a client
// pointed at it.
func newTTSTestClient(t *testing.T, handler http.HandlerFunc) *TTSClient {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewTTSClient(srv.URL)
}

func TestTTSClient_Synthesize_Success(t *testing.T) {
	var (
		gotMethod string
		gotType   string
		gotBody   ttsRequest
	)
	client := newTTSTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotType = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte("OggS-audio"))
	})

	audio, err := client.Synthesize(context.Background(), "Bonjour", "fr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(audio) != "OggS-audio" {
		t.Errorf("unexpected audio %q", audio)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotType != "application/json" {
		t.Errorf("expected a JSON content type, got %q", gotType)
	}
	if gotBody.Text != "Bonjour" || gotBody.Language != "FR" || gotBody.Format != "ogg" {
		t.Errorf("unexpected request body %+v", gotBody)
	}
}

func TestTTSClient_Synthesize_UppercasesLanguage(t *testing.T) {
	var gotBody ttsRequest
	client := newTTSTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = w.Write([]byte("audio"))
	})

	if _, err := client.Synthesize(context.Background(), "Hello", "en"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBody.Language != "EN" {
		t.Errorf("expected language EN, got %q", gotBody.Language)
	}
}

func TestTTSClient_Synthesize_NonOKStatus(t *testing.T) {
	client := newTTSTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(strings.Repeat("x", 2000)))
	})

	_, err := client.Synthesize(context.Background(), "Bonjour", "fr")

	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "502") {
		t.Errorf("error should mention the status code, got %q", err.Error())
	}
	// The error body is capped so a large HTML error page cannot flood the logs.
	if len(err.Error()) > ttsErrorSnippetBytes+100 {
		t.Errorf("error message is not truncated, len=%d", len(err.Error()))
	}
}

func TestTTSClient_Synthesize_EmptyAudio(t *testing.T) {
	client := newTTSTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	_, err := client.Synthesize(context.Background(), "Bonjour", "fr")

	if err == nil || !strings.Contains(err.Error(), "empty audio") {
		t.Fatalf("expected an empty audio error, got %v", err)
	}
}

func TestTTSClient_Synthesize_AudioTooLarge(t *testing.T) {
	tooBig := bytes.Repeat([]byte("a"), ttsMaxAudioBytes+1)
	client := newTTSTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(tooBig)
	})

	_, err := client.Synthesize(context.Background(), "Bonjour", "fr")

	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("expected a size limit error, got %v", err)
	}
}

func TestTTSClient_Synthesize_AudioAtLimitIsAccepted(t *testing.T) {
	atLimit := bytes.Repeat([]byte("a"), ttsMaxAudioBytes)
	client := newTTSTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(atLimit)
	})

	audio, err := client.Synthesize(context.Background(), "Bonjour", "fr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(audio) != ttsMaxAudioBytes {
		t.Errorf("expected %d bytes, got %d", ttsMaxAudioBytes, len(audio))
	}
}

func TestTTSClient_Synthesize_ServerUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	client := NewTTSClient(srv.URL)
	srv.Close()

	_, err := client.Synthesize(context.Background(), "Bonjour", "fr")

	if err == nil || !strings.Contains(err.Error(), "tts request failed") {
		t.Fatalf("expected a request failure, got %v", err)
	}
}

func TestTTSClient_Synthesize_CanceledContext(t *testing.T) {
	client := newTTSTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("audio"))
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.Synthesize(ctx, "Bonjour", "fr")

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestMockTTSClient_Synthesize(t *testing.T) {
	audio, err := NewMockTTSClient().Synthesize(context.Background(), "Bonjour", "fr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(audio) == 0 {
		t.Error("the mock should return a non-empty placeholder payload")
	}
}
