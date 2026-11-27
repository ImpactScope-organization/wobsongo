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

func TestBotSettingsService_GetEffective(t *testing.T) {
	const (
		defTone   = model.BotToneFriendly
		defLength = model.BotLengthLong
	)

	tests := []struct {
		name       string
		stored     *model.BotSettings
		storedErr  error
		wantTone   model.BotTone
		wantLength model.BotLength
	}{
		{
			name:       "no row uses both defaults",
			storedErr:  data.ErrNotFound,
			wantTone:   defTone,
			wantLength: defLength,
		},
		{
			name:       "row without any value uses both defaults",
			stored:     &model.BotSettings{JID: "jid-1"},
			wantTone:   defTone,
			wantLength: defLength,
		},
		{
			name:       "only tone set: length falls back to default",
			stored:     &model.BotSettings{JID: "jid-1", Tone: model.BotToneAcademic},
			wantTone:   model.BotToneAcademic,
			wantLength: defLength,
		},
		{
			name:       "only length set: tone falls back to default",
			stored:     &model.BotSettings{JID: "jid-1", Length: model.BotLengthShort},
			wantTone:   defTone,
			wantLength: model.BotLengthShort,
		},
		{
			name: "user's own values always win",
			stored: &model.BotSettings{
				JID: "jid-1", Tone: model.BotToneFormal, Length: model.BotLengthExpert,
			},
			wantTone:   model.BotToneFormal,
			wantLength: model.BotLengthExpert,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockrepo.BotSettingsRepoerMock{
				GetFunc: func(context.Context, string) (*model.BotSettings, error) {
					return tt.stored, tt.storedErr
				},
			}
			svc := NewBotSettingsService(repo).WithDefaults(defTone, defLength)

			got, err := svc.GetEffective(t.Context(), "jid-1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.JID != "jid-1" || got.Tone != tt.wantTone || got.Length != tt.wantLength {
				t.Errorf("got %+v, want jid-1/%s/%s", got, tt.wantTone, tt.wantLength)
			}
		})
	}
}

func TestBotSettingsService_GetEffective_NoDefaultsKeepsBaseline(t *testing.T) {
	repo := &mockrepo.BotSettingsRepoerMock{
		GetFunc: func(context.Context, string) (*model.BotSettings, error) {
			return nil, data.ErrNotFound
		},
	}

	got, err := NewBotSettingsService(repo).GetEffective(t.Context(), "jid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Tone != "" || got.Length != "" {
		t.Errorf("expected empty tone/length (baseline), got %+v", got)
	}
}

func TestBotSettingsService_GetEffective_PropagatesRepoError(t *testing.T) {
	boom := errors.New("db down")
	repo := &mockrepo.BotSettingsRepoerMock{
		GetFunc: func(context.Context, string) (*model.BotSettings, error) {
			return nil, boom
		},
	}
	svc := NewBotSettingsService(repo).WithDefaults(model.BotToneFriendly, model.BotLengthShort)

	got, err := svc.GetEffective(t.Context(), "jid-1")
	if !errors.Is(err, boom) {
		t.Errorf("expected the repo error, got %v", err)
	}
	if got != nil {
		t.Errorf("expected no settings alongside an error, got %+v", got)
	}
}

func TestBotSettingsService_GetEffective_DoesNotMutateStoredRow(t *testing.T) {
	stored := &model.BotSettings{JID: "jid-1"}
	repo := &mockrepo.BotSettingsRepoerMock{
		GetFunc: func(context.Context, string) (*model.BotSettings, error) {
			return stored, nil
		},
	}
	svc := NewBotSettingsService(repo).WithDefaults(model.BotToneFriendly, "")

	if _, err := svc.GetEffective(t.Context(), "jid-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.Tone != "" {
		t.Errorf("defaults must not leak into the repo's row, got %+v", stored)
	}
}

func TestBotSettingsService_Defaults(t *testing.T) {
	tests := []struct {
		name       string
		tone       model.BotTone
		length     model.BotLength
		wantTone   model.BotTone
		wantLength model.BotLength
	}{
		{"no defaults is the baseline", "", "", "", ""},
		{
			"configured defaults",
			model.BotToneFriendly, model.BotLengthShort,
			model.BotToneFriendly, model.BotLengthShort,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewBotSettingsService(&mockrepo.BotSettingsRepoerMock{}).
				WithDefaults(tt.tone, tt.length)

			got := svc.Defaults("jid-1")
			if got.JID != "jid-1" || got.Tone != tt.wantTone || got.Length != tt.wantLength {
				t.Errorf("got %+v, want jid-1/%s/%s", got, tt.wantTone, tt.wantLength)
			}
		})
	}
}
