package data

import (
	"context"

	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
)

// ConversationRepoer defines the data operations for the agentic bot's
// per-jid conversation history.
type ConversationRepoer interface {
	// AppendMessage stores one message (user, assistant, or system) for
	// the given jid.
	AppendMessage(
		ctx context.Context,
		jid string,
		role model.ConversationRole,
		content, phoneNumber, countryCode string,
	) error

	// RecentMessages returns up to limit most-recent messages for the
	// given jid, in chronological order (oldest first) — ready to feed
	// straight into an LLM messages array.
	RecentMessages(ctx context.Context, jid string, limit int) ([]model.ConversationMessage, error)

	// EnqueueAgentTurn adds a new agent-turn job to the River queue —
	// used both for a fresh user message and for a system-triggered
	// continuation turn (e.g. once a video transcript is ready).
	EnqueueAgentTurn(ctx context.Context, payload queue.AgentTurnJob) error
}
