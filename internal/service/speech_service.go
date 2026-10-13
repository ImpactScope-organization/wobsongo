package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
)

var (
	// ErrSpeechNotConfigured is returned when no synthesizer is available.
	ErrSpeechNotConfigured = errors.New("speech synthesis is not configured")

	// ErrSpeechEmptyText is returned when nothing is left to speak after cleaning.
	ErrSpeechEmptyText = errors.New("nothing to speak after cleaning the text")
)

// SpeechService turns bot replies into speech: it enqueues the background
// job (EnqueueSpeech) and does the actual work when the job runs (Synthesize).
type SpeechService struct {
	translator  data.Translator
	synthesizer data.SpeechSynthesizer
	speechRepo  data.SpeechRepoer
}

// NewSpeechService creates a SpeechService. translator and synthesizer may be
// nil when their services are not configured.
func NewSpeechService(
	translator data.Translator,
	synthesizer data.SpeechSynthesizer,
	speechRepo data.SpeechRepoer,
) *SpeechService {
	return &SpeechService{
		translator:  translator,
		synthesizer: synthesizer,
		speechRepo:  speechRepo,
	}
}

// EnqueueSpeech validates the request and enqueues a background speech job,
// returning as soon as the job is stored. When the job runs (see
// worker.SpeechWorker) the bot is notified of the result under jobID.
func (s *SpeechService) EnqueueSpeech(
	ctx context.Context,
	jobID, text, language string,
) error {
	if s.synthesizer == nil {
		return ErrSpeechNotConfigured
	}
	if CleanTextForSpeech(text) == "" {
		return ErrSpeechEmptyText
	}

	return s.speechRepo.EnqueueSpeech(ctx, queue.SpeechJob{
		JobID:    jobID,
		Text:     text,
		Language: language,
	})
}

// Synthesize speaks text in language ("fr" or "en"). text is expected to be
// the bot's reply, which is always French; for "en" it is translated first.
// The full text is spoken, nothing is truncated.
func (s *SpeechService) Synthesize(ctx context.Context, text, language string) ([]byte, error) {
	if s.synthesizer == nil {
		return nil, ErrSpeechNotConfigured
	}

	spoken := CleanTextForSpeech(text)
	if spoken == "" {
		return nil, ErrSpeechEmptyText
	}

	if language == "en" {
		if s.translator == nil {
			return nil, errors.New("translator is not configured")
		}
		translated, err := s.translator.Translate(ctx, spoken, model.LanguageFrench)
		if err != nil {
			return nil, fmt.Errorf("failed to translate reply for speech: %w", err)
		}
		spoken = CleanTextForSpeech(translated)
		if spoken == "" {
			return nil, ErrSpeechEmptyText
		}
	}

	return s.synthesizer.Synthesize(ctx, spoken, language)
}

var (
	reURL      = regexp.MustCompile(`https?://\S+`)
	reCitation = regexp.MustCompile(`\[\d+(?:\s*[,;-]\s*\d+)*\]`)
	reMarkdown = regexp.MustCompile("[*_#>`~|]+")
	reBullet   = regexp.MustCompile(`^\s*[-•·▪●]\s+`)
	reSpaces   = regexp.MustCompile(`[ \t]+`)

	speechReplacer = strings.NewReplacer("→", ", ", "➡", ", ", "•", "\n")
)

// CleanTextForSpeech strips what a TTS engine should not read aloud (emoji,
// markdown, URLs, [N] citations, bullets) and makes sure every line ends with
// sentence punctuation so the engine pauses between lines.
func CleanTextForSpeech(text string) string {
	text = reURL.ReplaceAllString(text, "")
	text = reCitation.ReplaceAllString(text, "")
	text = reMarkdown.ReplaceAllString(text, "")
	text = speechReplacer.Replace(text)
	text = strings.Map(stripSpeechRune, text)

	lines := strings.Split(text, "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		line = reBullet.ReplaceAllString(line, "")
		line = strings.TrimSpace(reSpaces.ReplaceAllString(line, " "))
		if line == "" {
			continue
		}
		if !endsWithSentencePunctuation(line) {
			line += "."
		}
		parts = append(parts, line)
	}
	return strings.Join(parts, " ")
}

// stripSpeechRune drops emoji and emoji glue characters, keeps newlines and
// the degree sign, and turns other control characters into spaces.
func stripSpeechRune(r rune) rune {
	switch {
	case r == '\n', r == '°':
		return r
	case r == 0xFE0F, r == 0x200D, r == 0x20E3:
		return -1
	case unicode.Is(unicode.So, r), unicode.Is(unicode.Sk, r):
		return -1
	case unicode.IsControl(r):
		return ' '
	}
	return r
}

func endsWithSentencePunctuation(line string) bool {
	for _, mark := range []string{".", "!", "?", "…", ":", ";"} {
		if strings.HasSuffix(line, mark) {
			return true
		}
	}
	return false
}
