package dto

// AgentInboundRequest is the payload wa-bot sends for every inbound WhatsApp message
type AgentInboundRequest struct {
	// Jid is the WhatsApp chat identifier this message came from.
	Jid string `json:"jid" validate:"required"`

	// Text is the raw inbound message text.
	Text string `json:"text" validate:"required"`
}

// AgentInboundResponse mirrors ExtractResponse's shape so
// wa-bot's existing pending-job
type AgentInboundResponse struct {
	// Status indicates the current state of the agent turn.
	Status ExtractStatus `json:"status"`

	// JobID is the job identifier returned to the bot.
	JobID string `json:"jobId"`

	// Message contains the agent's response when the turn completes.
	Message string `json:"message,omitempty"`
}

// AgentInboundVoiceRequest is the payload wa-bot sends for an inbound voice
// message. It carries a reference to the media, not the audio itself.
type AgentInboundVoiceRequest struct {
	// JID is the chat identifier this voice message came from.
	JID string `json:"jid" validate:"required"`

	// Channel is the messaging platform the message came from.
	Channel string `json:"channel" validate:"required,oneof=whatsapp telegram"`

	// MediaRef is a Twilio media URL (whatsapp) or a file_id (telegram).
	MediaRef string `json:"mediaRef" validate:"required,max=2048"`

	// MimeType is the audio MIME type reported by the platform.
	MimeType string `json:"mimeType" validate:"required,max=100"`
}
