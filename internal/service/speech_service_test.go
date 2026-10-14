package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
)

// speechTestSynth is a hand-rolled data.SpeechSynthesizer that records its input.
type speechTestSynth struct {
	audio   []byte
	err     error
	calls   int
	gotText string
	gotLang string
}

func (s *speechTestSynth) Synthesize(_ context.Context, text, language string) ([]byte, error) {
	s.calls++
	s.gotText = text
	s.gotLang = language
	return s.audio, s.err
}

// speechTestTranslator is a hand-rolled data.Translator that records its input.
type speechTestTranslator struct {
	out       string
	err       error
	calls     int
	gotText   string
	gotSource model.Language
}

func (s *speechTestTranslator) Translate(
	_ context.Context,
	text string,
	sourceLanguage model.Language,
) (string, error) {
	s.calls++
	s.gotText = text
	s.gotSource = sourceLanguage
	return s.out, s.err
}

func TestCleanTextForSpeech(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain sentence gets a final period", "Bonjour tout le monde", "Bonjour tout le monde."},
		{"existing punctuation is kept", "Salut !", "Salut !"},
		{
			"markdown and emoji are stripped",
			"Bonjour **tout le monde** 👋",
			"Bonjour tout le monde.",
		},
		{"heading marker is stripped", "# Titre", "Titre."},
		{"urls are removed", "Voir https://example.com/x pour plus", "Voir pour plus."},
		{"citations are removed", "Fait avéré [1]", "Fait avéré."},
		{"dash bullets are removed", "- un\n- deux", "un. deux."},
		{"dot bullets become sentences", "• a\n• b", "a. b."},
		{"lines are joined with pauses", "Ligne un\nLigne deux", "Ligne un. Ligne deux."},
		{"degree sign is kept", "Il fait 30° ce jour", "Il fait 30° ce jour."},
		{"only emoji and a url leaves nothing", "🎉 https://example.com", ""},
		{"empty input", "", ""},
		{"whitespace only", "  \n\t ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CleanTextForSpeech(tt.in); got != tt.want {
				t.Errorf("CleanTextForSpeech(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSpeechService_Synthesize_French(t *testing.T) {
	synth := &speechTestSynth{audio: []byte("OggS")}
	translator := &speechTestTranslator{}
	svc := NewSpeechService(translator, synth, nil)

	audio, err := svc.Synthesize(context.Background(), "Bonjour **tout le monde** 👋", "fr")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(audio) != "OggS" {
		t.Errorf("unexpected audio %q", audio)
	}
	if synth.gotText != "Bonjour tout le monde." || synth.gotLang != "fr" {
		t.Errorf("unexpected synthesizer input %q (%s)", synth.gotText, synth.gotLang)
	}
	if translator.calls != 0 {
		t.Errorf("French replies must not be translated, got %d calls", translator.calls)
	}
}

func TestSpeechService_Synthesize_EnglishIsTranslatedFirst(t *testing.T) {
	synth := &speechTestSynth{audio: []byte("OggS")}
	translator := &speechTestTranslator{out: "Hello **everyone** 🎉"}
	svc := NewSpeechService(translator, synth, nil)

	if _, err := svc.Synthesize(context.Background(), "Bonjour tout le monde", "en"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if translator.gotText != "Bonjour tout le monde." ||
		translator.gotSource != model.LanguageFrench {
		t.Errorf("unexpected translator input %q (%v)", translator.gotText, translator.gotSource)
	}
	if synth.gotText != "Hello everyone." || synth.gotLang != "en" {
		t.Errorf("unexpected synthesizer input %q (%s)", synth.gotText, synth.gotLang)
	}
}

func TestSpeechService_Synthesize_Errors(t *testing.T) {
	translateErr := errors.New("translation api is down")
	synthErr := errors.New("modal is cold")

	tests := []struct {
		name     string
		svc      func() *SpeechService
		text     string
		language string
		wantIs   error // nil means any error is acceptable
	}{
		{
			name:     "synthesizer not configured",
			svc:      func() *SpeechService { return NewSpeechService(nil, nil, nil) },
			text:     "Bonjour",
			language: "fr",
			wantIs:   ErrSpeechNotConfigured,
		},
		{
			name: "nothing left to speak",
			svc: func() *SpeechService {
				return NewSpeechService(nil, &speechTestSynth{}, nil)
			},
			text:     "🎉 https://example.com",
			language: "fr",
			wantIs:   ErrSpeechEmptyText,
		},
		{
			name: "english without a translator",
			svc: func() *SpeechService {
				return NewSpeechService(nil, &speechTestSynth{}, nil)
			},
			text:     "Bonjour",
			language: "en",
		},
		{
			name: "translation fails",
			svc: func() *SpeechService {
				return NewSpeechService(
					&speechTestTranslator{err: translateErr},
					&speechTestSynth{},
					nil,
				)
			},
			text:     "Bonjour",
			language: "en",
			wantIs:   translateErr,
		},
		{
			name: "translation is empty after cleaning",
			svc: func() *SpeechService {
				return NewSpeechService(&speechTestTranslator{out: "🎉"}, &speechTestSynth{}, nil)
			},
			text:     "Bonjour",
			language: "en",
			wantIs:   ErrSpeechEmptyText,
		},
		{
			name: "synthesizer fails",
			svc: func() *SpeechService {
				return NewSpeechService(nil, &speechTestSynth{err: synthErr}, nil)
			},
			text:     "Bonjour",
			language: "fr",
			wantIs:   synthErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.svc().Synthesize(context.Background(), tt.text, tt.language)

			if err == nil {
				t.Fatal("expected an error")
			}
			if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
				t.Errorf("expected error matching %v, got %v", tt.wantIs, err)
			}
		})
	}
}

func TestSpeechService_Synthesize_NothingIsSynthesizedWhenTextIsEmpty(t *testing.T) {
	synth := &speechTestSynth{}
	svc := NewSpeechService(nil, synth, nil)

	_, err := svc.Synthesize(context.Background(), "🎉", "fr")

	if !errors.Is(err, ErrSpeechEmptyText) {
		t.Fatalf("expected ErrSpeechEmptyText, got %v", err)
	}
	if synth.calls != 0 {
		t.Errorf("the synthesizer must not be called, got %d calls", synth.calls)
	}
}

func TestSpeechService_EnqueueSpeech_Success(t *testing.T) {
	repo := &mockrepo.SpeechRepoerMock{
		EnqueueSpeechFunc: func(context.Context, queue.SpeechJob) error { return nil },
	}
	svc := NewSpeechService(nil, &speechTestSynth{}, repo)

	err := svc.EnqueueSpeech(context.Background(), "job-1", "Bonjour **monde** 👋", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := repo.EnqueueSpeechCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 enqueued job, got %d", len(calls))
	}
	// The raw text is queued; cleaning and translation happen when the job runs.
	want := queue.SpeechJob{JobID: "job-1", Text: "Bonjour **monde** 👋", Language: "en"}
	if calls[0].Payload != want {
		t.Errorf("unexpected payload %+v, want %+v", calls[0].Payload, want)
	}
}

func TestSpeechService_EnqueueSpeech_Rejected(t *testing.T) {
	tests := []struct {
		name  string
		synth data.SpeechSynthesizer
		text  string
		want  error
	}{
		{"synthesizer not configured", nil, "Bonjour", ErrSpeechNotConfigured},
		{"nothing to speak", &speechTestSynth{}, "🎉 https://example.com", ErrSpeechEmptyText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockrepo.SpeechRepoerMock{
				EnqueueSpeechFunc: func(context.Context, queue.SpeechJob) error { return nil },
			}
			svc := NewSpeechService(nil, tt.synth, repo)

			err := svc.EnqueueSpeech(context.Background(), "job-1", tt.text, "fr")

			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
			if n := len(repo.EnqueueSpeechCalls()); n != 0 {
				t.Errorf("expected no job to be enqueued, got %d", n)
			}
		})
	}
}

func TestSpeechService_EnqueueSpeech_RepoErrorsPropagate(t *testing.T) {
	queueFull := fmt.Errorf("%w: 18 speech jobs already waiting", data.ErrQueueFull)
	repo := &mockrepo.SpeechRepoerMock{
		EnqueueSpeechFunc: func(context.Context, queue.SpeechJob) error { return queueFull },
	}
	svc := NewSpeechService(nil, &speechTestSynth{}, repo)

	err := svc.EnqueueSpeech(context.Background(), "job-1", "Bonjour", "fr")

	if !errors.Is(err, data.ErrQueueFull) {
		t.Fatalf("expected data.ErrQueueFull, got %v", err)
	}
}
