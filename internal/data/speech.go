package data

import (
	"context"

	"github.com/impactscope-organization/wobsongo/internal/queue"
)

// SpeechSynthesizer converts text into spoken audio.
type SpeechSynthesizer interface {
	// Synthesize returns OGG/Opus audio of text spoken in language ("fr" or "en").
	Synthesize(ctx context.Context, text, language string) ([]byte, error)
}

// SpeechRepoer enqueues background speech-synthesis jobs.
type SpeechRepoer interface {
	// EnqueueSpeech adds a speech job to the River queue. It returns
	// ErrQueueFull when too many speech jobs are already waiting.
	EnqueueSpeech(ctx context.Context, payload queue.SpeechJob) error
}

// SpeechTranscriber converts spoken audio into text.
type SpeechTranscriber interface {
	// TranscribeURL does the same for audio the ASR service downloads itself from audioURL.
	TranscribeURL(ctx context.Context, audioURL, format string) (string, error)
}
