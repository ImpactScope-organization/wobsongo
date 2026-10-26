package model

import "time"

// BotTone is the writing style of the bot's replies.
type BotTone string

const (
	// BotToneFormal is neutral and structured, like the baseline reply.
	BotToneFormal BotTone = "formal"

	// BotToneAcademic uses formal, objective language.
	BotToneAcademic BotTone = "academic"

	// BotToneFriendly is warm and conversational.
	BotToneFriendly BotTone = "friendly"
)

// Valid reports whether t is one of the known tones.
func (t BotTone) Valid() bool {
	switch t {
	case BotToneFormal, BotToneAcademic, BotToneFriendly:
		return true
	default:
		return false
	}
}

// BotLength controls how much of a verdict is shown.
type BotLength string

const (
	// BotLengthShort shows the verdict and one sentence, without the header.
	BotLengthShort BotLength = "short"

	// BotLengthMedium is the baseline: header plus one sentence per claim.
	BotLengthMedium BotLength = "medium"

	// BotLengthLong shows the full reasoning per claim, without citations.
	BotLengthLong BotLength = "long"

	// BotLengthExpert shows the full reasoning plus the cited evidence.
	BotLengthExpert BotLength = "expert"
)

// Valid reports whether l is one of the known lengths.
func (l BotLength) Valid() bool {
	switch l {
	case BotLengthShort, BotLengthMedium, BotLengthLong, BotLengthExpert:
		return true
	default:
		return false
	}
}

// BotSettings holds one WhatsApp chat's response preferences. An empty Tone or
// Length means the user never set it, so the baseline reply applies.
type BotSettings struct {
	// JID is the WhatsApp chat identifier the settings belong to.
	JID string `db:"jid" json:"jid"`

	// Tone is empty when the user never chose one.
	Tone BotTone `db:"tone" json:"tone,omitempty"`

	// Length is empty when the user never chose one.
	Length BotLength `db:"length" json:"length,omitempty"`

	// UpdatedAt is the timestamp when the settings last changed.
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
