package data

import (
	"context"

	"github.com/impactscope-organization/wobsongo/internal/model"
)

// BotSettingsRepoer defines the data operations for the bot's per-jid
// response preferences (tone and length).
type BotSettingsRepoer interface {
	// Get returns the settings for the given jid, or ErrNotFound when the
	// user never set any.
	Get(ctx context.Context, jid string) (*model.BotSettings, error)

	// Upsert saves the non-empty fields of settings for settings.JID and
	// leaves the empty ones unchanged. It returns the settings as stored.
	Upsert(ctx context.Context, settings *model.BotSettings) (*model.BotSettings, error)
}
