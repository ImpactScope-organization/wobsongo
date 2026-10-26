package repo

import (
	"context"
	"fmt"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

// botSettingsRepo implements data.BotSettingsRepoer.
type botSettingsRepo struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

// NewBotSettingsRepo creates a new repository for the bot's per-jid
// response preferences.
func NewBotSettingsRepo(q *db.Queries, pool *pgxpool.Pool) data.BotSettingsRepoer {
	return &botSettingsRepo{q: q, pool: pool}
}

// Get returns the settings for the given jid, or data.ErrNotFound.
func (r *botSettingsRepo) Get(ctx context.Context, jid string) (*model.BotSettings, error) {
	row, err := r.q.GetBotSettings(ctx, jid)
	if err != nil {
		return nil, fmt.Errorf("failed to get bot settings: %w", mapPostgresError(err))
	}
	return toModelBotSettings(&row), nil
}

// Upsert saves the non-empty fields of settings; empty ones stay unchanged.
func (r *botSettingsRepo) Upsert(
	ctx context.Context,
	settings *model.BotSettings,
) (*model.BotSettings, error) {
	row, err := r.q.UpsertBotSettings(ctx, db.UpsertBotSettingsParams{
		Jid:    settings.JID,
		Tone:   toPgText(string(settings.Tone)),
		Length: toPgText(string(settings.Length)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert bot settings: %w", mapPostgresError(err))
	}
	return toModelBotSettings(&row), nil
}

func toModelBotSettings(row *db.BotSetting) *model.BotSettings {
	return &model.BotSettings{
		JID:       row.Jid,
		Tone:      model.BotTone(row.Tone.String),
		Length:    model.BotLength(row.Length.String),
		UpdatedAt: row.UpdatedAt,
	}
}
