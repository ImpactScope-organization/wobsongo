package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/core"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/labstack/echo/v4"
)

const (
	inboundVoicePath = "/api/agent/inbound-voice"
	ttsPath          = "/api/tts"

	twilioMediaURL = "https://api.twilio.com/2010-04-01/Accounts/AC1/Messages/MM1/Media/ME1"
)

func newSpeechTestApp(
	t *testing.T,
	conversationRepo data.ConversationRepoer,
	speechRepo data.SpeechRepoer,
	synthesizer data.SpeechSynthesizer,
) *echo.Echo {
	t.Helper()
	// The agent is enabled by default and then needs an LLM endpoint to be
	// built. It is not under test here.
	t.Setenv("AGENT_ENABLED", "false")

	app := core.NewApp(
		testhelpers.NewEcho(),
		internal.NewConfig(),
		core.WithConversationRepo(conversationRepo),
		core.WithSpeechRepo(speechRepo),
		core.WithSpeechSynthesizer(synthesizer),
	)
	return app.Echo()
}

// speechAuth returns a valid Authorization header for the bot-facing routes.
func speechAuth() string {
	return "PSK " + internal.NewConfig().BotExtractPSK
}

func speechJSON(t *testing.T, v any) []byte {
	t.Helper()
	body, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}
	return body
}

func speechPost(
	app *echo.Echo,
	path string,
	body []byte,
	auth string,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if auth != "" {
		req.Header.Set(echo.HeaderAuthorization, auth)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

func validVoiceRequest() map[string]any {
	return map[string]any{
		"jid":      "22500000000",
		"channel":  "whatsapp",
		"mediaRef": twilioMediaURL,
		"mimeType": "audio/ogg; codecs=opus",
	}
}

func TestInboundVoiceHandler_Accepted(t *testing.T) {
	tests := []struct {
		name     string
		channel  string
		mediaRef string
		mimeType string
	}{
		{"whatsapp ogg with codecs", "whatsapp", twilioMediaURL, "audio/ogg; codecs=opus"},
		{"whatsapp mp3", "whatsapp", twilioMediaURL, "audio/mpeg"},
		{"telegram voice", "telegram", "AwACAgQAAxkBAAI_-x", "audio/ogg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockrepo.ConversationRepoerMock{}
			repo.EnqueueVoiceTranscriptionFunc = func(
				context.Context, *queue.VoiceTranscriptionJob,
			) error {
				return nil
			}
			app := newSpeechTestApp(
				t, repo, &mockrepo.SpeechRepoerMock{}, external.NewMockTTSClient(),
			)

			body := speechJSON(t, map[string]any{
				"jid":      "22500000000",
				"channel":  tt.channel,
				"mediaRef": tt.mediaRef,
				"mimeType": tt.mimeType,
			})
			rec := speechPost(app, inboundVoicePath, body, speechAuth())

			if rec.Code != http.StatusAccepted {
				t.Fatalf("expected status %d, got %d: %s",
					http.StatusAccepted, rec.Code, rec.Body.String())
			}
			var resp testhelpers.APIResponse[dto.AgentInboundResponse]
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response body: %v", err)
			}
			if resp.Data.Status != dto.StatusProcessing || resp.Data.JobID == "" {
				t.Errorf("expected a processing job with an ID, got %+v", resp.Data)
			}

			calls := repo.EnqueueVoiceTranscriptionCalls()
			if len(calls) != 1 {
				t.Fatalf("expected 1 enqueued job, got %d", len(calls))
			}
			got := calls[0].Payload
			if got.ExtractionID != resp.Data.JobID {
				t.Errorf("job ID %q does not match response %q", got.ExtractionID, resp.Data.JobID)
			}
			if got.JID != "22500000000" || got.Channel != tt.channel ||
				got.MediaRef != tt.mediaRef || got.MimeType != tt.mimeType {
				t.Errorf("unexpected enqueued payload: %+v", got)
			}
		})
	}
}

func TestInboundVoiceHandler_ValidationErrors(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(body map[string]any)
		want   int
	}{
		{
			"missing jid",
			func(b map[string]any) { delete(b, "jid") },
			http.StatusUnprocessableEntity,
		},
		{
			"missing channel",
			func(b map[string]any) { delete(b, "channel") },
			http.StatusUnprocessableEntity,
		},
		{
			"unknown channel",
			func(b map[string]any) { b["channel"] = "sms" },
			http.StatusUnprocessableEntity,
		},
		{
			"missing mediaRef",
			func(b map[string]any) { delete(b, "mediaRef") },
			http.StatusUnprocessableEntity,
		},
		{
			"mediaRef too long",
			func(b map[string]any) { b["mediaRef"] = "https://api.twilio.com/" + strings.Repeat("a", 2048) },
			http.StatusUnprocessableEntity,
		},
		{
			"missing mimeType",
			func(b map[string]any) { delete(b, "mimeType") },
			http.StatusUnprocessableEntity,
		},
		{
			"untrusted host",
			func(b map[string]any) { b["mediaRef"] = "https://evil.com/x" },
			http.StatusUnprocessableEntity,
		},
		{
			"twilio over http",
			func(b map[string]any) { b["mediaRef"] = "http://api.twilio.com/x" },
			http.StatusUnprocessableEntity,
		},
		{
			"lookalike host",
			func(b map[string]any) { b["mediaRef"] = "https://api.twilio.com.evil.com/x" },
			http.StatusUnprocessableEntity,
		},
		{
			"malformed telegram file id",
			func(b map[string]any) {
				b["channel"] = "telegram"
				b["mediaRef"] = "../../x"
			},
			http.StatusUnprocessableEntity,
		},
		{
			"video mime type",
			func(b map[string]any) { b["mimeType"] = "video/mp4" },
			http.StatusUnsupportedMediaType,
		},
		{
			"text mime type",
			func(b map[string]any) { b["mimeType"] = "text/plain" },
			http.StatusUnsupportedMediaType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockrepo.ConversationRepoerMock{}
			app := newSpeechTestApp(
				t, repo, &mockrepo.SpeechRepoerMock{}, external.NewMockTTSClient(),
			)

			body := validVoiceRequest()
			tt.mutate(body)
			rec := speechPost(app, inboundVoicePath, speechJSON(t, body), speechAuth())

			if rec.Code != tt.want {
				t.Fatalf("expected status %d, got %d: %s", tt.want, rec.Code, rec.Body.String())
			}
			if n := len(repo.EnqueueVoiceTranscriptionCalls()); n != 0 {
				t.Errorf("expected no job to be enqueued, got %d", n)
			}
		})
	}
}

func TestInboundVoiceHandler_AuthAndBody(t *testing.T) {
	valid := speechJSON(t, validVoiceRequest())
	tests := []struct {
		name string
		auth string
		body []byte
		want int
	}{
		{"missing PSK", "", valid, http.StatusUnauthorized},
		{"wrong PSK", "PSK definitely-not-the-key", valid, http.StatusUnauthorized},
		{"malformed JSON", speechAuth(), []byte(`{"jid":`), http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockrepo.ConversationRepoerMock{}
			app := newSpeechTestApp(
				t, repo, &mockrepo.SpeechRepoerMock{}, external.NewMockTTSClient(),
			)

			rec := speechPost(app, inboundVoicePath, tt.body, tt.auth)

			if rec.Code != tt.want {
				t.Fatalf("expected status %d, got %d: %s", tt.want, rec.Code, rec.Body.String())
			}
			if n := len(repo.EnqueueVoiceTranscriptionCalls()); n != 0 {
				t.Errorf("expected no job to be enqueued, got %d", n)
			}
		})
	}
}

func TestInboundVoiceHandler_EnqueueFailure(t *testing.T) {
	repo := &mockrepo.ConversationRepoerMock{}
	repo.EnqueueVoiceTranscriptionFunc = func(
		context.Context, *queue.VoiceTranscriptionJob,
	) error {
		return errors.New("database is down")
	}
	app := newSpeechTestApp(t, repo, &mockrepo.SpeechRepoerMock{}, external.NewMockTTSClient())

	rec := speechPost(app, inboundVoicePath, speechJSON(t, validVoiceRequest()), speechAuth())

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d: %s",
			http.StatusInternalServerError, rec.Code, rec.Body.String())
	}
}

func TestTTSHandler_Accepted(t *testing.T) {
	for _, language := range []string{"fr", "en"} {
		t.Run(language, func(t *testing.T) {
			speechRepo := &mockrepo.SpeechRepoerMock{}
			speechRepo.EnqueueSpeechFunc = func(context.Context, queue.SpeechJob) error {
				return nil
			}
			app := newSpeechTestApp(
				t, &mockrepo.ConversationRepoerMock{}, speechRepo, external.NewMockTTSClient(),
			)

			body := speechJSON(t, map[string]any{
				"jobId":    "job-1",
				"text":     "Bonjour **tout le monde** 👋",
				"language": language,
			})
			rec := speechPost(app, ttsPath, body, speechAuth())

			if rec.Code != http.StatusAccepted {
				t.Fatalf("expected status %d, got %d: %s",
					http.StatusAccepted, rec.Code, rec.Body.String())
			}
			var resp testhelpers.APIResponse[dto.TTSResponse]
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response body: %v", err)
			}
			if resp.Data.Status != "processing" || resp.Data.JobID != "job-1" {
				t.Errorf("unexpected response: %+v", resp.Data)
			}

			calls := speechRepo.EnqueueSpeechCalls()
			if len(calls) != 1 {
				t.Fatalf("expected 1 enqueued speech job, got %d", len(calls))
			}
			got := calls[0].Payload
			if got.JobID != "job-1" || got.Language != language ||
				got.Text != "Bonjour **tout le monde** 👋" {
				t.Errorf("unexpected enqueued payload: %+v", got)
			}
		})
	}
}

func TestTTSHandler_Rejected(t *testing.T) {
	valid := func() map[string]any {
		return map[string]any{"jobId": "job-1", "text": "Bonjour", "language": "fr"}
	}
	tests := []struct {
		name   string
		mutate func(body map[string]any)
		want   int
	}{
		{
			"missing jobId",
			func(b map[string]any) { delete(b, "jobId") },
			http.StatusUnprocessableEntity,
		},
		{
			"missing text",
			func(b map[string]any) { delete(b, "text") },
			http.StatusUnprocessableEntity,
		},
		{
			"missing language",
			func(b map[string]any) { delete(b, "language") },
			http.StatusUnprocessableEntity,
		},
		{
			"unsupported language",
			func(b map[string]any) { b["language"] = "de" },
			http.StatusUnprocessableEntity,
		},
		{
			"nothing left to speak",
			func(b map[string]any) { b["text"] = "🎉 https://example.com" },
			http.StatusUnprocessableEntity,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			speechRepo := &mockrepo.SpeechRepoerMock{}
			app := newSpeechTestApp(
				t, &mockrepo.ConversationRepoerMock{}, speechRepo, external.NewMockTTSClient(),
			)

			body := valid()
			tt.mutate(body)
			rec := speechPost(app, ttsPath, speechJSON(t, body), speechAuth())

			if rec.Code != tt.want {
				t.Fatalf("expected status %d, got %d: %s", tt.want, rec.Code, rec.Body.String())
			}
			if n := len(speechRepo.EnqueueSpeechCalls()); n != 0 {
				t.Errorf("expected no job to be enqueued, got %d", n)
			}
		})
	}
}

func TestTTSHandler_AuthAndBody(t *testing.T) {
	valid := speechJSON(t, map[string]any{"jobId": "job-1", "text": "Bonjour", "language": "fr"})
	tests := []struct {
		name string
		auth string
		body []byte
		want int
	}{
		{"missing PSK", "", valid, http.StatusUnauthorized},
		{"wrong PSK", "PSK definitely-not-the-key", valid, http.StatusUnauthorized},
		{"malformed JSON", speechAuth(), []byte(`{"jobId":`), http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			speechRepo := &mockrepo.SpeechRepoerMock{}
			app := newSpeechTestApp(
				t, &mockrepo.ConversationRepoerMock{}, speechRepo, external.NewMockTTSClient(),
			)

			rec := speechPost(app, ttsPath, tt.body, tt.auth)

			if rec.Code != tt.want {
				t.Fatalf("expected status %d, got %d: %s", tt.want, rec.Code, rec.Body.String())
			}
			if n := len(speechRepo.EnqueueSpeechCalls()); n != 0 {
				t.Errorf("expected no job to be enqueued, got %d", n)
			}
		})
	}
}

func TestTTSHandler_EnqueueErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			"queue full",
			fmt.Errorf("%w: 18 speech jobs already waiting", data.ErrQueueFull),
			http.StatusServiceUnavailable,
		},
		{"unexpected failure", errors.New("database is down"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			speechRepo := &mockrepo.SpeechRepoerMock{}
			speechRepo.EnqueueSpeechFunc = func(context.Context, queue.SpeechJob) error {
				return tt.err
			}
			app := newSpeechTestApp(
				t, &mockrepo.ConversationRepoerMock{}, speechRepo, external.NewMockTTSClient(),
			)

			body := speechJSON(
				t,
				map[string]any{"jobId": "job-1", "text": "Bonjour", "language": "fr"},
			)
			rec := speechPost(app, ttsPath, body, speechAuth())

			if rec.Code != tt.want {
				t.Fatalf("expected status %d, got %d: %s", tt.want, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestTTSHandler_SynthesizerNotConfigured(t *testing.T) {
	speechRepo := &mockrepo.SpeechRepoerMock{}
	app := newSpeechTestApp(t, &mockrepo.ConversationRepoerMock{}, speechRepo, nil)

	body := speechJSON(t, map[string]any{"jobId": "job-1", "text": "Bonjour", "language": "fr"})
	rec := speechPost(app, ttsPath, body, speechAuth())

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d: %s",
			http.StatusServiceUnavailable, rec.Code, rec.Body.String())
	}
	if n := len(speechRepo.EnqueueSpeechCalls()); n != 0 {
		t.Errorf("expected no job to be enqueued, got %d", n)
	}
}
