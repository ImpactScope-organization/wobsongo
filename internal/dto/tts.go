package dto

// TTSRequest is the payload wa-bot sends to turn a bot reply into speech.
type TTSRequest struct {
	// JobID is the bot's correlation ID (its pending job). The finished audio
	// is reported back to the bot under this ID.
	JobID string `json:"jobId" validate:"required"`

	// Text is the bot reply to speak. It is French. For Language "en" it is
	// translated before synthesis.
	Text string `json:"text" validate:"required"`

	// Language is the language the audio is spoken in: "fr" or "en".
	Language string `json:"language" validate:"required,oneof=fr en"`
}

// TTSResponse acknowledges that synthesis has started. The audio itself is
// delivered later through the bot's /callback/tts-done endpoint.
type TTSResponse struct {
	Status string `json:"status"`
	JobID  string `json:"jobId"`
}
