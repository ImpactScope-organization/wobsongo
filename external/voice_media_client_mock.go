package external

import (
	"context"
	"log"

	"github.com/impactscope-organization/wobsongo/internal/data"
)

// MockVoiceMediaClient is a stub data.VoiceMediaDownloader for tests.
type MockVoiceMediaClient struct {
	// Audio is returned by Download; nil falls back to a placeholder payload.
	Audio []byte

	// Err, when set, is returned by Download instead of audio.
	Err error
}

// NewMockVoiceMediaClient creates a MockVoiceMediaClient.
func NewMockVoiceMediaClient() *MockVoiceMediaClient {
	return &MockVoiceMediaClient{}
}

// mockVoiceAudio is a placeholder payload, not playable audio.
var mockVoiceAudio = []byte("mock-voice-audio")

// Download implements data.VoiceMediaDownloader.
func (c *MockVoiceMediaClient) Download(
	_ context.Context,
	channel data.VoiceChannel,
	_ string,
) ([]byte, error) {
	log.Printf("[MockVoiceMediaClient] skipping real download (channel=%s)", channel)
	if c.Err != nil {
		return nil, c.Err
	}
	if c.Audio != nil {
		return c.Audio, nil
	}
	return mockVoiceAudio, nil
}
