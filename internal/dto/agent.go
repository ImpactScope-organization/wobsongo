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

// AgentInboundVoiceRequest holds the form fields wa-bot sends along with the
// uploaded audio file (multipart field "audio").
type AgentInboundVoiceRequest struct {
	// Jid is the chat identifier this voice message came from.
	Jid string `form:"jid" validate:"required"`

	// Format is the audio format hint.
	Format string `form:"format" validate:"required,alphanum,max=8"`
}
