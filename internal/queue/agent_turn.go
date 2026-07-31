package queue

import "github.com/riverqueue/river"

// AgentTurnJob runs one conversational turn of the agentic bot workflow for
// a given WhatsApp jid: it loads recent conversation history, calls the
// agent LLM (with tools), executes any tool calls, and notifies the bot
// with the final natural-language reply via the existing
// BotClient.NotifyExtractDone callback
type AgentTurnJob struct {
	// Jid is the WhatsApp chat identifier this turn belongs to.
	Jid string `json:"jid"`

	// ExtractionID is the correlation ID used by the bot to wait for
	// NotifyExtractDone. For continuation turns, a new ID is generated
	// and returned in the {status:"processing", jobId} response.
	ExtractionID string `json:"extraction_id"`

	// UserText is the raw user message, already persisted by the caller.
	// Empty for system-triggered continuation turns.
	UserText string `json:"user_text,omitempty"`

	// SystemNote is used for system-triggered turns (e.g. "video transcript
	// is ready") instead of a user message. It is stored as a system-role
	// entry before the turn runs.
	SystemNote string `json:"system_note,omitempty"`
}

// CurrentTurnInput returns whichever of UserText/SystemNote is set for this turn
func (j AgentTurnJob) CurrentTurnInput() string {
	if j.UserText != "" {
		return j.UserText
	}
	return j.SystemNote
}

// Kind implements queue.BackgroundJob and river.JobArgs.
func (AgentTurnJob) Kind() string { return string(JobTypeAgentTurn) }

// InsertOpts implements river.JobArgsWithInsertOpts. Reuses the
// media-processing queue since agent turns are triggered by the same video/claim pipeline
func (AgentTurnJob) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueMediaProcessing}
}
