package data

import "context"

// SpeechSynthesizer converts text into spoken audio.
type SpeechSynthesizer interface {
	// Synthesize returns OGG/Opus audio of text spoken in language ("fr" or "en").
	Synthesize(ctx context.Context, text, language string) ([]byte, error)
}
