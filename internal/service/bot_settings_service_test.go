package service

import (
	"context"
	"errors"
	"testing"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

func TestBotSettingsService_Get_PassesThroughNotFound(t *testing.T) {
	repo := &mockrepo.BotSettingsRepoerMock{
		GetFunc: func(context.Context, string) (*model.BotSettings, error) {
			return nil, data.ErrNotFound
		},
	}

	_, err := NewBotSettingsService(repo).Get(t.Context(), "jid-1")
	if !errors.Is(err, data.ErrNotFound) {
		t.Errorf("expected data.ErrNotFound, got %v", err)
	}
}

func TestBotSettingsService_Update_SavesNonEmptyFields(t *testing.T) {
	var saved *model.BotSettings
	repo := &mockrepo.BotSettingsRepoerMock{
		UpsertFunc: func(_ context.Context, s *model.BotSettings) (*model.BotSettings, error) {
			saved = s
			return s, nil
		},
	}

	got, err := NewBotSettingsService(repo).Update(
		t.Context(), "jid-1", model.BotToneFriendly, "",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saved == nil || saved.JID != "jid-1" || saved.Tone != model.BotToneFriendly ||
		saved.Length != "" {
		t.Errorf("expected repo to receive jid-1/friendly/empty length, got %+v", saved)
	}
	if got.Tone != model.BotToneFriendly {
		t.Errorf("expected the stored settings to be returned, got %+v", got)
	}
}

func TestBotSettingsService_Update_Validation(t *testing.T) {
	tests := []struct {
		name    string
		tone    model.BotTone
		length  model.BotLength
		wantErr error
	}{
		{"nothing to update", "", "", ErrEmptyBotSettings},
		{"unknown tone", "sarcastic", model.BotLengthShort, ErrInvalidBotSetting},
		{"unknown length", model.BotToneFriendly, "huge", ErrInvalidBotSetting},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockrepo.BotSettingsRepoerMock{}
			_, err := NewBotSettingsService(repo).Update(t.Context(), "jid-1", tt.tone, tt.length)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected %v, got %v", tt.wantErr, err)
			}
			if len(repo.UpsertCalls()) != 0 {
				t.Error("an invalid update must not reach the repo")
			}
		})
	}
}
