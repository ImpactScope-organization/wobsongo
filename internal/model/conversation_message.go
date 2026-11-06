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

	// JID is the WhatsApp chat identifier this message belongs to.
	JID string `db:"jid" json:"jid"`

	// Role identifies who authored the message.
	Role ConversationRole `db:"role" json:"role"`

	// Content is the message text.
	Content string `db:"content" json:"content"`

	// CreatedAt is the timestamp when the message was stored.
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// UserActivity counts what users sent in one window.
type UserActivity struct {
	// Users is the number of distinct jids that sent at least one message.
	Users int64 `json:"users"`

	// Messages is the number of messages users sent.
	Messages int64 `json:"messages"`
}

// MessageVolume counts the messages users sent in one window, next to the
// window right before it.
type MessageVolume struct {
	// Current is the number of messages in the selected window.
	Current int64 `json:"current"`

	// Previous is the number of messages in the window before it.
	Previous int64 `json:"previous"`

	// Trend is the current window's volume per bucket, oldest first: hourly
	// for 24h, daily for 7d and 30d.
	Trend []int64 `json:"trend"`
}
