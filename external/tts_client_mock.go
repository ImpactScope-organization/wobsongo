package external

import (
	"context"
	"log"
)

// MockTTSClient is a stub data.SpeechSynthesizer used when MODAL_TTS_ENDPOINT
// starts with mock:// (local development and tests only).
type MockTTSClient struct{}

// NewMockTTSClient creates a MockTTSClient.
func NewMockTTSClient() *MockTTSClient {
	return &MockTTSClient{}
}

// mockAudio is a placeholder payload, not playable audio.
var mockAudio = []byte("mock-tts-audio")

// Synthesize implements data.SpeechSynthesizer.
func (c *MockTTSClient) Synthesize(_ context.Context, text, language string) ([]byte, error) {
	log.Printf("[MockTTSClient] skipping real synthesis (lang=%s, %d chars)", language, len(text))
	return mockAudio, nil
}
