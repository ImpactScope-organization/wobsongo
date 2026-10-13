package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

const (
	// maxConcurrentSpeechJobs bounds how many syntheses run at once, so a burst
	// of voice messages cannot pile unlimited requests onto the GPU endpoint.
	maxConcurrentSpeechJobs = 4

	// speechJobTimeout bounds one background synthesis (translation + TTS).
	speechJobTimeout = 10 * time.Minute

	// speechNotifyTimeout bounds the callback that delivers the result to the bot.
	speechNotifyTimeout = 30 * time.Second

	speechStatusCompleted = "completed"
	speechStatusFailed    = "failed"
)

var (
	// ErrSpeechNotConfigured is returned when no synthesizer is available.
	ErrSpeechNotConfigured = errors.New("speech synthesis is not configured")

	// ErrSpeechEmptyText is returned when nothing is left to speak after cleaning.
	ErrSpeechEmptyText = errors.New("nothing to speak after cleaning the text")
)

// SpeechNotifier reports the result of a background speech job to the bot.
type SpeechNotifier interface {
	NotifyTTSDone(ctx context.Context, jobID, status string, audio []byte) error
}

type SpeechService struct {
	translator  data.Translator
	synthesizer data.SpeechSynthesizer
	notifier    SpeechNotifier
	slots       chan struct{}
}

func NewSpeechService(
	translator data.Translator,
	synthesizer data.SpeechSynthesizer,
	notifier SpeechNotifier,
) *SpeechService {
	return &SpeechService{
		translator:  translator,
		synthesizer: synthesizer,
		notifier:    notifier,
		slots:       make(chan struct{}, maxConcurrentSpeechJobs),
	}
}

// SynthesizeAsync starts synthesizing text in the background and returns
// immediately. When the job finishes (or fails) the bot is notified through
// SpeechNotifier under jobID.
func (s *SpeechService) SynthesizeAsync(jobID, text, language string) error {
	if s.synthesizer == nil {
		return ErrSpeechNotConfigured
	}
	if CleanTextForSpeech(text) == "" {
		return ErrSpeechEmptyText
	}

	go s.runSpeechJob(jobID, text, language)
	return nil
}

func (s *SpeechService) runSpeechJob(jobID, text, language string) {
	s.slots <- struct{}{}
	defer func() { <-s.slots }()

	audio, err := s.synthesizeSafely(text, language)
	if err != nil {
		log.Printf("[SpeechService] job %s failed: %v", jobID, err)
		s.notifyBot(jobID, speechStatusFailed, nil)
		return
	}

	log.Printf("[SpeechService] job %s completed (%d bytes)", jobID, len(audio))
	s.notifyBot(jobID, speechStatusCompleted, audio)
}

// notifyBot reports the job result to the bot.
func (s *SpeechService) notifyBot(jobID, status string, audio []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), speechNotifyTimeout)
	defer cancel()

	if err := s.notifier.NotifyTTSDone(ctx, jobID, status, audio); err != nil {
		log.Printf("[SpeechService] failed to notify bot for job %s: %v", jobID, err)
	}
}

// synthesizeSafely runs Synthesize with a timeout and turns a panic into an error.
func (s *SpeechService) synthesizeSafely(text, language string) (audio []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during speech synthesis: %v", r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), speechJobTimeout)
	defer cancel()
	return s.Synthesize(ctx, text, language)
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
