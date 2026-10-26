package internal

import (
	"strings"
	"testing"
	"time"
)

const (
	testModalTTSEndpoint = "https://acme--melotts-web.modal.run"
	testModalASREndpoint = "https://acme--asr-web.modal.run"
)

// requireErrContains fails unless err is nil when want is empty, or contains
// want otherwise.
func requireErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		return
	}
	if err == nil {
		t.Fatalf("expected an error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected an error containing %q, got %q", want, err.Error())
	}
}

func validTTSConfig() *TTSConfig {
	return &TTSConfig{
		ModalMeloTTSConfig: ModalMeloTTSConfig{Endpoint: testModalTTSEndpoint},
		MaxWorkers:         4,
		MaxPendingJobs:     18,
	}
}

func validASRConfig() *ASRConfig {
	return &ASRConfig{
		Endpoint:   testModalASREndpoint,
		Model:      "Omnilingual ASR",
		SourceLang: "auto",
	}
}

func validVoiceMediaConfig() *VoiceMediaConfig {
	return &VoiceMediaConfig{TelegramBotToken: "telegram-token"}
}

// validConfig returns a Config that passes IsOK. Tests break one field at a
// time to prove the server refuses to start.
func validConfig() *Config {
	return &Config{
		PostgresURI:      "postgres://localhost/test",
		JWTSecret:        "secret",
		JWTAccessTTL:     time.Minute,
		JWTRefreshTTL:    time.Hour,
		Port:             8000,
		EmailConfig:      &EmailConfig{},
		ApifyConfig:      &ApifyConfig{Token: "apify-token"},
		ASRConfig:        validASRConfig(),
		TTSConfig:        validTTSConfig(),
		VoiceMediaConfig: validVoiceMediaConfig(),
	}
}

func TestValidateModalEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr string
	}{
		{"valid modal endpoint", testModalTTSEndpoint, ""},
		{"valid with path", testModalTTSEndpoint + "/synthesize", ""},
		{"empty", "", "is not set"},
		{"malformed url", "http://[::1", "not a valid URL"},
		{"http scheme", "http://acme--melotts-web.modal.run", "must use https"},
		{"mock scheme", "mock://tts", "must use https"},
		{"no scheme", "acme--melotts-web.modal.run", "must use https"},
		{"other host", "https://example.com", "not an allowed modal.run domain"},
		{"lookalike host", "https://acme.modal.run.evil.com", "not an allowed modal.run domain"},
		{"bare modal.run", "https://modal.run", "not an allowed modal.run domain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireErrContains(t, validateModalEndpoint("MODAL_TEST_ENDPOINT", tt.url), tt.wantErr)
		})
	}
}

func TestValidateModalEndpoint_NamesTheVariable(t *testing.T) {
	err := validateModalEndpoint("MODAL_TTS_ENDPOINT", "http://example.com")
	requireErrContains(t, err, "MODAL_TTS_ENDPOINT")
}

func TestModalMeloTTSConfig_UseMock(t *testing.T) {
	tests := []struct {
		endpoint string
		want     bool
	}{
		{"mock://tts", true},
		{MockTTSEndpointPrefix, true},
		{testModalTTSEndpoint, false},
		{"", false},
		{"http://mock://tts", false},
		{"MOCK://tts", false},
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			got := ModalMeloTTSConfig{Endpoint: tt.endpoint}.UseMock()
			if got != tt.want {
				t.Errorf("UseMock(%q) = %v, want %v", tt.endpoint, got, tt.want)
			}
		})
	}
}

func TestConfig_ttsOK(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr string
	}{
		{"valid real endpoint", func(*Config) {}, ""},
		{
			"explicit mock endpoint",
			func(c *Config) { c.TTSConfig.ModalMeloTTSConfig.Endpoint = "mock://tts" },
			"",
		},
		{"nil config", func(c *Config) { c.TTSConfig = nil }, "TTSConfig is not set"},
		{
			"empty endpoint must not fall back to the mock",
			func(c *Config) { c.TTSConfig.ModalMeloTTSConfig.Endpoint = "" },
			"MODAL_TTS_ENDPOINT is not set",
		},
		{
			"http endpoint",
			func(c *Config) { c.TTSConfig.ModalMeloTTSConfig.Endpoint = "http://x.modal.run" },
			"must use https",
		},
		{
			"non-modal host",
			func(c *Config) { c.TTSConfig.ModalMeloTTSConfig.Endpoint = "https://example.com" },
			"not an allowed modal.run domain",
		},
		{
			"malformed endpoint",
			func(c *Config) { c.TTSConfig.ModalMeloTTSConfig.Endpoint = "http://[::1" },
			"not a valid URL",
		},
		{"zero workers", func(c *Config) { c.TTSConfig.MaxWorkers = 0 }, "TTS_MAX_WORKERS"},
		{
			"negative pending jobs",
			func(c *Config) { c.TTSConfig.MaxPendingJobs = -1 },
			"TTS_MAX_PENDING_JOBS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validConfig()
			tt.mutate(c)
			requireErrContains(t, c.ttsOK(), tt.wantErr)
		})
	}
}

func TestConfig_ttsOK_EmptyEndpointHintsAtMock(t *testing.T) {
	c := validConfig()
	c.TTSConfig.ModalMeloTTSConfig.Endpoint = ""

	requireErrContains(t, c.ttsOK(), MockTTSEndpointPrefix)
}

func TestConfig_asrOK(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr string
	}{
		{"valid", func(*Config) {}, ""},
		{
			"nil config",
			func(c *Config) { c.ASRConfig = nil },
			"MODAL_ASR_ENDPOINT config is not set",
		},
		{"empty endpoint", func(c *Config) { c.ASRConfig.Endpoint = "" }, "MODAL_ASR_ENDPOINT"},
		{
			"non-modal host",
			func(c *Config) { c.ASRConfig.Endpoint = "https://example.com" },
			"not an allowed modal.run domain",
		},
		{
			"mock is not allowed for ASR",
			func(c *Config) { c.ASRConfig.Endpoint = "mock://asr" },
			"must use https",
		},
		{"unknown model", func(c *Config) { c.ASRConfig.Model = "nope" }, "MODAL_ASR_MODEL"},
		{
			"unknown source language",
			func(c *Config) { c.ASRConfig.SourceLang = "klingon" },
			"MODAL_ASR_SOURCE_LANG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validConfig()
			tt.mutate(c)
			requireErrContains(t, c.asrOK(), tt.wantErr)
		})
	}
}

func TestVoiceMediaConfig_IsOK(t *testing.T) {
	tests := []struct {
		name    string
		cfg     VoiceMediaConfig
		wantErr string
	}{
		{"telegram only", VoiceMediaConfig{TelegramBotToken: "t"}, ""},
		{"twilio only", VoiceMediaConfig{TwilioAccountSID: "AC1", TwilioAuthToken: "tok"}, ""},
		{
			"both platforms",
			VoiceMediaConfig{
				TwilioAccountSID: "AC1",
				TwilioAuthToken:  "tok",
				TelegramBotToken: "t",
			},
			"",
		},
		{"nothing set", VoiceMediaConfig{}, "to download voice messages"},
		{
			"sid without token",
			VoiceMediaConfig{TwilioAccountSID: "AC1", TelegramBotToken: "t"},
			"must be set together",
		},
		{
			"token without sid",
			VoiceMediaConfig{TwilioAuthToken: "tok", TelegramBotToken: "t"},
			"must be set together",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireErrContains(t, tt.cfg.IsOK(), tt.wantErr)
		})
	}
}

func TestConfig_voiceMediaOK_NilConfig(t *testing.T) {
	c := validConfig()
	c.VoiceMediaConfig = nil

	requireErrContains(t, c.voiceMediaOK(), "VoiceMediaConfig is not set")
}

// IsOK is what `serve` runs before anything else, and os.Exit(1)s on error.
// These cases prove an invalid voice configuration stops the app from starting.
func TestConfig_IsOK_VoiceSettings(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr string
	}{
		{"fully valid config", func(*Config) {}, ""},
		{
			"mock TTS endpoint is accepted",
			func(c *Config) { c.TTSConfig.ModalMeloTTSConfig.Endpoint = "mock://tts" },
			"",
		},
		{
			"empty TTS endpoint",
			func(c *Config) { c.TTSConfig.ModalMeloTTSConfig.Endpoint = "" },
			"MODAL_TTS_ENDPOINT is not set",
		},
		{
			"invalid TTS endpoint",
			func(c *Config) { c.TTSConfig.ModalMeloTTSConfig.Endpoint = "https://example.com" },
			"MODAL_TTS_ENDPOINT",
		},
		{
			"invalid ASR endpoint",
			func(c *Config) { c.ASRConfig.Endpoint = "http://x.modal.run" },
			"MODAL_ASR_ENDPOINT",
		},
		{
			"missing voice download credentials",
			func(c *Config) { c.VoiceMediaConfig = &VoiceMediaConfig{} },
			"to download voice messages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validConfig()
			tt.mutate(c)
			requireErrContains(t, c.IsOK(), tt.wantErr)
		})
	}
}
