package repo_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/jackc/pgx/v5/pgxpool"
)

// withEmptySettingTx runs fn in a transaction that is always rolled back and
// starts with an empty crawler_setting table, so the tests do not depend on
// whatever a developer saved in the local database.
func withEmptySettingTx(
	t *testing.T,
	pool *pgxpool.Pool,
	fn func(ctx context.Context, q *db.Queries),
) {
	t.Helper()

	ctx := t.Context()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin test tx: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "DELETE FROM crawler_setting"); err != nil {
		t.Fatalf("failed to empty crawler_setting: %v", err)
	}

	fn(ctx, db.New(tx))
}

func TestCrawlerRepo_Setting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("GetSetting_CreatesDefaultRow", func(t *testing.T) {
		withEmptySettingTx(t, pool, func(ctx context.Context, q *db.Queries) {
			got, err := repo.NewCrawlerRepo(q).GetSetting(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.DiscoveryIntervalHours != 24 || got.MaxResultsKeyword != 20 ||
				got.MaxResultsInfluencer != 10 || got.MaxCostPerRun != 5 {
				t.Errorf("unexpected defaults: %+v", got)
			}
			if got.UpdatedBy != nil {
				t.Errorf("expected no updater on the default row, got %v", got.UpdatedBy)
			}
			if got.UpdatedAt.IsZero() {
				t.Error("expected updated_at to be set")
			}
		})
	})

	t.Run("UpdateSetting_SavesValuesAndReturnsPreviousInterval", func(t *testing.T) {
		withEmptySettingTx(t, pool, func(ctx context.Context, q *db.Queries) {
			crawler := repo.NewCrawlerRepo(q)
			if _, err := crawler.GetSetting(ctx); err != nil { // creates the 24h default
				t.Fatalf("unexpected error: %v", err)
			}

			setting := &model.AdminSetting{
				DiscoveryIntervalHours: 6,
				MaxResultsKeyword:      30,
				MaxResultsInfluencer:   15,
				MaxCostPerRun:          12.5,
			}
			previous, err := crawler.UpdateSetting(ctx, setting)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if previous != 24 {
				t.Errorf("expected previous interval 24, got %d", previous)
			}
			if setting.UpdatedAt.IsZero() {
				t.Error("expected UpdateSetting to fill UpdatedAt")
			}

			got, err := crawler.GetSetting(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.DiscoveryIntervalHours != 6 || got.MaxResultsKeyword != 30 ||
				got.MaxResultsInfluencer != 15 || got.MaxCostPerRun != 12.5 {
				t.Errorf("saved values were not stored: %+v", got)
			}
		})
	})

	t.Run("UpdateSetting_FirstSaveReturnsZeroPreviousInterval", func(t *testing.T) {
		withEmptySettingTx(t, pool, func(ctx context.Context, q *db.Queries) {
			crawler := repo.NewCrawlerRepo(q)

			previous, err := crawler.UpdateSetting(ctx, &model.AdminSetting{
				DiscoveryIntervalHours: 12,
				MaxResultsKeyword:      20,
				MaxResultsInfluencer:   10,
				MaxCostPerRun:          5,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if previous != 0 {
				t.Errorf("expected 0 when no row existed, got %d", previous)
			}

			got, err := crawler.GetSetting(ctx)
			if err != nil || got.DiscoveryIntervalHours != 12 {
				t.Fatalf("expected the first save to be stored, got %+v, %v", got, err)
			}
		})
	})

	t.Run("UpdateSetting_RecordsUpdatedBy", func(t *testing.T) {
		withEmptySettingTx(t, pool, func(ctx context.Context, q *db.Queries) {
			users := repo.NewUserRepo(q, pool)
			user := newTestUser(uuid.NewString() + "@example.com")
			if err := users.Create(ctx, user); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			crawler := repo.NewCrawlerRepo(q)
			_, err := crawler.UpdateSetting(ctx, &model.AdminSetting{
				DiscoveryIntervalHours: 6,
				MaxResultsKeyword:      30,
				MaxResultsInfluencer:   15,
				MaxCostPerRun:          12.5,
				UpdatedBy:              &user.ID,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, err := crawler.GetSetting(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.UpdatedBy == nil || *got.UpdatedBy != user.ID {
				t.Errorf("expected updated_by %s, got %v", user.ID, got.UpdatedBy)
			}
		})
	})

	t.Run("UpdateSetting_RejectsNonPositiveLimits", func(t *testing.T) {
		withEmptySettingTx(t, pool, func(ctx context.Context, q *db.Queries) {
			_, err := repo.NewCrawlerRepo(q).UpdateSetting(ctx, &model.AdminSetting{
				DiscoveryIntervalHours: 6,
				MaxResultsKeyword:      0, // violates CHECK (max_results_keyword > 0)
				MaxResultsInfluencer:   15,
				MaxCostPerRun:          12.5,
			})
			if err == nil {
				t.Fatal("expected the CHECK constraint to reject a zero limit")
			}
		})
	})
}
