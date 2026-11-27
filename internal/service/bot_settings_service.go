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
	defaultTone     model.BotTone
	defaultLength   model.BotLength
}

// NewBotSettingsService creates a new BotSettingsService with no defaults
// (the baseline reply applies to chats that never set anything).
func NewBotSettingsService(botSettingsRepo data.BotSettingsRepoer) *BotSettingsService {
	return &BotSettingsService{botSettingsRepo: botSettingsRepo}
}

// WithDefaults sets the tone/length used for chats that never chose their own.
// An empty value keeps the baseline for that field. 
func (s *BotSettingsService) WithDefaults(
	tone model.BotTone,
	length model.BotLength,
) *BotSettingsService {
	s.defaultTone = tone
	s.defaultLength = length
	return s
}

// Defaults returns the settings of a chat that never set anything.
func (s *BotSettingsService) Defaults(jid string) model.BotSettings {
	return model.BotSettings{JID: jid, Tone: s.defaultTone, Length: s.defaultLength}
}

// GetEffective returns the settings to apply for the given jid: the user's own
// tone/length, with the configured default filling in any field the user never
// set (or every field when the jid has no row).
func (s *BotSettingsService) GetEffective(
	ctx context.Context,
	jid string,
) (*model.BotSettings, error) {
	settings, err := s.botSettingsRepo.Get(ctx, jid)
	if err != nil {
		if !errors.Is(err, data.ErrNotFound) {
			return nil, err
		}
		// The chat never set anything: only the defaults apply.
		settings = &model.BotSettings{JID: jid}
	}

	effective := *settings
	if effective.Tone == "" {
		effective.Tone = s.defaultTone
	}
	if effective.Length == "" {
		effective.Length = s.defaultLength
	}
	return &effective, nil
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
