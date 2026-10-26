package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

var (
	// ErrInvalidBotSetting is returned when a tone or length is not one of the
	// known values.
	ErrInvalidBotSetting = errors.New("invalid bot setting")

	// ErrEmptyBotSettings is returned when an update names neither a tone nor
	// a length.
	ErrEmptyBotSettings = errors.New("nothing to update: tone and length are both empty")
)

// BotSettingsService wraps the bot's per-jid response preferences repo.
type BotSettingsService struct {
	botSettingsRepo data.BotSettingsRepoer
}

// NewBotSettingsService creates a new BotSettingsService.
func NewBotSettingsService(botSettingsRepo data.BotSettingsRepoer) *BotSettingsService {
	return &BotSettingsService{botSettingsRepo: botSettingsRepo}
}

// Get returns the settings for the given jid, or data.ErrNotFound when the
// user never set any.
func (s *BotSettingsService) Get(ctx context.Context, jid string) (*model.BotSettings, error) {
	return s.botSettingsRepo.Get(ctx, jid)
}

// Update saves the non-empty tone/length for the given jid and leaves the
// empty one unchanged.
func (s *BotSettingsService) Update(
	ctx context.Context,
	jid string,
	tone model.BotTone,
	length model.BotLength,
) (*model.BotSettings, error) {
	if tone == "" && length == "" {
		return nil, ErrEmptyBotSettings
	}
	if tone != "" && !tone.Valid() {
		return nil, fmt.Errorf("%w: tone %q", ErrInvalidBotSetting, tone)
	}
	if length != "" && !length.Valid() {
		return nil, fmt.Errorf("%w: length %q", ErrInvalidBotSetting, length)
	}

	return s.botSettingsRepo.Upsert(ctx, &model.BotSettings{
		JID:    jid,
		Tone:   tone,
		Length: length,
	})
}
