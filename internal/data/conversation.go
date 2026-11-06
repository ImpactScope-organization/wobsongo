package data

import (
	"context"
	"time"

	"github.com/google/uuid"
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
		content string,
	) error

	// AppendMessageWithID is AppendMessage that also returns the new
	// message's id.
	AppendMessageWithID(
		ctx context.Context,
		jid string,
		role model.ConversationRole,
		content string,
	) (uuid.UUID, error)

	// RecentMessages returns up to limit most-recent messages for the
	// given jid, in chronological order (oldest first) — ready to feed
	// straight into an LLM messages array.
	RecentMessages(ctx context.Context, jid string, limit int) ([]model.ConversationMessage, error)

	// EnqueueAgentTurn adds a new agent-turn job to the River queue —
	// used both for a fresh user message and for a system-triggered
	// continuation turn (e.g. once a video transcript is ready).
	EnqueueAgentTurn(ctx context.Context, payload queue.AgentTurnJob) error

	// EnqueueVoiceTranscription adds a new voice-transcription job to the River queue.
	EnqueueVoiceTranscription(ctx context.Context, payload *queue.VoiceTranscriptionJob) error

	// UserActivity counts the distinct users and the messages they sent,
	// over the last filter.Days days or over all time.
	UserActivity(ctx context.Context, filter UserActivityFilter) (*model.UserActivity, error)

	// MessageVolume counts the messages users sent in the current window and
	// in the window before it, and the current window's volume per bucket.
	MessageVolume(ctx context.Context, filter MessageVolumeFilter) (*model.MessageVolume, error)
}

// UserActivityFilter narrows a user activity count. A nil field is not filtered.
type UserActivityFilter struct {
	// Days counts only the last Days days. Nil counts all time.
	Days *int32
}

// MessageVolumeFilter selects the windows of a message volume count. The
// current window ends now, the previous one is the same length right before it.
type MessageVolumeFilter struct {
	// Window is the length of one window.
	Window time.Duration

	// Bucket is the size of one trend bucket. Window must be a multiple of it.
	Bucket time.Duration
}
