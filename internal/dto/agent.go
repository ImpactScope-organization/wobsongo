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

	// JobID is the correlation ID the bot should register as a pending
	// job — the eventual /callback/extract-done call will carry this
	// same ID.
	JobID string `json:"jobId"`
}
