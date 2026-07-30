package dto

// AgentInboundRequest is the payload wa-bot sends for every inbound WhatsApp message
type AgentInboundRequest struct {
	// Jid is the WhatsApp chat identifier this message came from.
	Jid string `json:"jid" validate:"required"`

	// Text is the raw inbound message text.
	Text string `json:"text" validate:"required"`

	// PhoneNumber is the sender's phone number.
	PhoneNumber string `json:"phoneNumber,omitempty"`

	// CountryCode is the sender's phone country code.
	CountryCode string `json:"countryCode,omitempty"`
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
