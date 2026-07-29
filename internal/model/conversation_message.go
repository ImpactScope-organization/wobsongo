package model

import (
	"time"

	"github.com/google/uuid"
)

// ConversationRole identifies who authored a conversation message.
type ConversationRole string

const (
	// ConversationRoleUser marks a message sent by the WhatsApp user.
	ConversationRoleUser ConversationRole = "user"

	// ConversationRoleAssistant marks a message sent by the agent back to the user.
	ConversationRoleAssistant ConversationRole = "assistant"

	// ConversationRoleSystem marks an internally-generated note injected into
	// the conversation (e.g. "video transcript is ready"), never shown to
	// the user directly but included in the LLM's context window.
	ConversationRoleSystem ConversationRole = "system"
)

// ConversationMessage represents one message in the agentic bot's chat
// history for a given WhatsApp jid, used to give the LLM conversational context
type ConversationMessage struct {
	// ID is the unique identifier for the message record.
	ID uuid.UUID `json:"id" format:"uuid"`

	// Jid is the WhatsApp chat identifier this message belongs to.
	Jid string `db:"jid" json:"jid"`

	// Role identifies who authored the message.
	Role ConversationRole `db:"role" json:"role"`

	// Content is the message text.
	Content string `db:"content" json:"content"`

	// PhoneNumber is the sender's phone number.
	PhoneNumber string `db:"phone_number" json:"phoneNumber,omitempty"`

	// CountryCode is the sender's country code.
	CountryCode string `db:"country_code" json:"countryCode,omitempty"`

	// CreatedAt is the timestamp when the message was stored.
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
