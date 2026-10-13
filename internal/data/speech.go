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
	// EnqueueSpeech adds a speech job to the River queue.
	EnqueueSpeech(ctx context.Context, payload queue.SpeechJob) error
}

// SpeechTranscriber converts spoken audio into text.
type SpeechTranscriber interface {
	// Transcribe returns the transcript of audio 
	// The transcript is empty when no speech was recognized.
	Transcribe(ctx context.Context, audio []byte, format string) (string, error)

	// TranscribeURL does the same for audio the ASR service downloads itself from audioURL.
	TranscribeURL(ctx context.Context, audioURL, format string) (string, error)
}
