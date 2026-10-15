package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/impactscope-organization/wobsongo/internal"
	"github.com/impactscope-organization/wobsongo/internal/core"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/labstack/echo/v4"
)

const (
	mediaUploadPath   = "/api/v1/media/upload"
	mediaPresignPath  = "/api/v1/media/presigned-url"
	mediaWrongPSK     = "PSK definitely-not-the-key"
	mediaTestFileHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

// stubMediaProvider is a hand-rolled data.MediaUploadProvider.
type stubMediaProvider struct {
	postIntents []data.MediaUploadIntent
	getKeys     []string
}

func (s *stubMediaProvider) GetPresignedPOSTURL(
	_ context.Context,
	intent data.MediaUploadIntent,
	filename, _ string,
) (*url.URL, map[string]string, error) {
	s.postIntents = append(s.postIntents, intent)
	u := &url.URL{Scheme: "https", Host: "s3.test", Path: "/bucket"}
	return u, map[string]string{"key": data.ObjectPrefixForIntent(intent) + filename}, nil
}

func (s *stubMediaProvider) GetPresignedGETURL(
	_ context.Context, key string, _ int64,
) (string, error) {
	s.getKeys = append(s.getKeys, key)
	return "https://s3.test/" + key + "?sig=1", nil
}

func (s *stubMediaProvider) GetPresignedGETURLs(
	context.Context, []string, int64,
) (map[string]string, error) {
	return nil, errors.New("GetPresignedGETURLs is not used by the media routes")
}

func newMediaTestApp(t *testing.T, provider data.MediaUploadProvider) *echo.Echo {
	t.Helper()
	// The agent is enabled by default and then needs an LLM endpoint to be
	// built. It is not under test here.
	t.Setenv("AGENT_ENABLED", "false")

	app := core.NewApp(
		testhelpers.NewEcho(),
		internal.NewConfig(),
		core.WithMediaProvider(provider),
	)
	return app.Echo()
}

func mediaUploadTarget(intent, filename string) string {
	q := url.Values{
		"intent":       {intent},
		"filename":     {filename},
		"content_type": {"audio/ogg"},
	}
	return mediaUploadPath + "?" + q.Encode()
}

func mediaPresignTarget(key string) string {
	return mediaPresignPath + "?" + url.Values{"s3_key": {key}}.Encode()
}

func mediaGet(app *echo.Echo, target, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if auth != "" {
		req.Header.Set(echo.HeaderAuthorization, auth)
	}
	rec := httptest.NewRecorder()
	app.ServeHTTP(rec, req)
	return rec
}

// Both routes hand out access to every intent's objects (including voice
// notes and synthesized replies), so they must only answer to the bot.
func TestMediaRoutes_RequirePSK(t *testing.T) {
	voiceKey := data.ObjectPrefixForIntent(data.TTSAudioUploadIntent) + mediaTestFileHash + ".ogg"
	targets := []struct {
		name   string
		target string
	}{
		{
			"upload",
			mediaUploadTarget(string(data.VoiceNoteUploadIntent), mediaTestFileHash+".ogg"),
		},
		{"presigned url", mediaPresignTarget(voiceKey)},
	}
	auths := []struct {
		name string
		auth string
	}{
		{"missing PSK", ""},
		{"wrong PSK", mediaWrongPSK},
	}

	for _, tg := range targets {
		for _, a := range auths {
			t.Run(tg.name+"/"+a.name, func(t *testing.T) {
				provider := &stubMediaProvider{}
				app := newMediaTestApp(t, provider)

				rec := mediaGet(app, tg.target, a.auth)

				if rec.Code != http.StatusUnauthorized {
					t.Fatalf("expected status %d, got %d: %s",
						http.StatusUnauthorized, rec.Code, rec.Body.String())
				}
				if len(provider.postIntents) != 0 || len(provider.getKeys) != 0 {
					t.Errorf("the provider must not be reached without a valid PSK, got %+v",
						provider)
				}
			})
		}
	}
}

func TestMediaUploadRoute_AcceptsVoiceIntentsWithPSK(t *testing.T) {
	tests := []data.MediaUploadIntent{
		data.VoiceNoteUploadIntent,
		data.TTSAudioUploadIntent,
	}

	for _, intent := range tests {
		t.Run(string(intent), func(t *testing.T) {
			provider := &stubMediaProvider{}
			app := newMediaTestApp(t, provider)

			rec := mediaGet(
				app,
				mediaUploadTarget(string(intent), mediaTestFileHash+".ogg"),
				speechAuth(),
			)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d: %s",
					http.StatusOK, rec.Code, rec.Body.String())
			}
			var resp testhelpers.APIResponse[model.POSTUploadPolicy]
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to unmarshal response body: %v", err)
			}
			if want := data.ObjectPrefixForIntent(intent); resp.Data.Prefix != want {
				t.Errorf("expected prefix %q, got %q", want, resp.Data.Prefix)
			}
			if len(provider.postIntents) != 1 || provider.postIntents[0] != intent {
				t.Errorf("expected one upload for intent %q, got %v", intent, provider.postIntents)
			}
		})
	}
}

func TestMediaUploadRoute_RejectsUnknownIntentWithPSK(t *testing.T) {
	provider := &stubMediaProvider{}
	app := newMediaTestApp(t, provider)

	rec := mediaGet(app, mediaUploadTarget("avatar", mediaTestFileHash+".ogg"), speechAuth())

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s",
			http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
	if len(provider.postIntents) != 0 {
		t.Errorf("the provider must not be reached for an unknown intent")
	}
}

func TestMediaPresignedURLRoute_WithPSK(t *testing.T) {
	key := data.ObjectPrefixForIntent(data.TTSAudioUploadIntent) + mediaTestFileHash + ".ogg"
	provider := &stubMediaProvider{}
	app := newMediaTestApp(t, provider)

	rec := mediaGet(app, mediaPresignTarget(key), speechAuth())

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s",
			http.StatusOK, rec.Code, rec.Body.String())
	}
	var resp testhelpers.APIResponse[dto.PresignedURL]
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}
	if !strings.Contains(resp.Data.PresignedURL, key) {
		t.Errorf("expected a presigned URL for %q, got %q", key, resp.Data.PresignedURL)
	}
	if resp.Data.TTL != 900 {
		t.Errorf("expected the default TTL of 900s, got %d", resp.Data.TTL)
	}
	if len(provider.getKeys) != 1 || provider.getKeys[0] != key {
		t.Errorf("expected %q to be presigned once, got %v", key, provider.getKeys)
	}
}

func TestMediaPresignedURLRoute_RejectsInvalidKeyWithPSK(t *testing.T) {
	provider := &stubMediaProvider{}
	app := newMediaTestApp(t, provider)

	rec := mediaGet(app, mediaPresignTarget("../etc/passwd"), speechAuth())

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s",
			http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	}
	if len(provider.getKeys) != 0 {
		t.Errorf("the provider must not be reached for an invalid key")
	}
}
