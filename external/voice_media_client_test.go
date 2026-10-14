package external

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/impactscope-organization/wobsongo/internal/data"
)

func TestValidateVoiceMediaRef(t *testing.T) {
	good := "https://api.twilio.com/2010-04-01/Accounts/AC1/Messages/MM1/Media/ME1"
	if err := data.ValidateVoiceMediaRef(data.VoiceChannelWhatsApp, good); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		"http://api.twilio.com/x",
		"https://evil.com/x",
		"https://api.twilio.com.evil.com/x",
		"https://u:p@api.twilio.com/x",
		"::",
	} {
		err := data.ValidateVoiceMediaRef(data.VoiceChannelWhatsApp, bad)
		if !errors.Is(err, data.ErrInvalidVoiceMediaRef) {
			t.Fatalf("%q accepted: %v", bad, err)
		}
	}
	if err := data.ValidateVoiceMediaRef(
		data.VoiceChannelTelegram,
		"AwACAgQAAxkBAAI_-x",
	); err != nil {
		t.Fatal(err)
	}
	if err := data.ValidateVoiceMediaRef(data.VoiceChannelTelegram, "../../x"); err == nil {
		t.Fatal("bad file id accepted")
	}
	if f, ok := data.VoiceFormatFromMime("audio/ogg; codecs=opus"); !ok || f != "ogg" {
		t.Fatal(f, ok)
	}
	if _, ok := data.VoiceFormatFromMime("video/mp4"); ok {
		t.Fatal("video accepted")
	}
}

func TestVoiceMediaClientTelegramAndLimits(t *testing.T) {
	big := strings.Repeat("a", int(data.MaxVoiceBytes)+10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/getFile"):
			fmt.Fprint(w, `{"ok":true,"result":{"file_path":"voice/f.oga","file_size":10}}`)
		case strings.HasSuffix(r.URL.Path, "/voice/f.oga"):
			fmt.Fprint(w, "OggS-data")
		case r.URL.Path == "/big":
			fmt.Fprint(w, big)
		}
	}))
	defer srv.Close()

	c := NewVoiceMediaClient("", "", "TOKEN")
	c.telegramBaseURL = srv.URL

	got, err := c.Download(context.Background(), data.VoiceChannelTelegram, "fileid123")
	if err != nil || string(got) != "OggS-data" {
		t.Fatal(string(got), err)
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/big", nil)
	if _, err := c.fetch(req); !errors.Is(err, data.ErrVoiceTooLarge) {
		t.Fatalf("expected ErrVoiceTooLarge, got %v", err)
	}
	req, _ = http.NewRequest(http.MethodGet, srv.URL+"/empty", nil)
	if _, err := c.fetch(req); !errors.Is(err, data.ErrVoiceEmpty) {
		t.Fatalf("expected ErrVoiceEmpty, got %v", err)
	}

	// A transport error must not leak the bot token.
	c.telegramBaseURL = "http://127.0.0.1:1"
	_, err = c.Download(context.Background(), data.VoiceChannelTelegram, "fileid123")
	if err == nil || strings.Contains(err.Error(), "TOKEN") {
		t.Fatalf("token leaked or no error: %v", err)
	}

	// Missing credentials fail loudly.
	_, err = NewVoiceMediaClient("", "", "").Download(
		context.Background(), data.VoiceChannelWhatsApp, "https://api.twilio.com/x",
	)
	if err == nil {
		t.Fatal("expected credentials error")
	}
}
