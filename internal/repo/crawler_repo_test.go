package repo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/jackc/pgx/v5"
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

	if _, err := tx.Exec(ctx, "delete from crawler_setting"); err != nil {
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
			got, err := repo.NewCrawlerRepo(q, pool).GetSetting(ctx)
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
			crawler := repo.NewCrawlerRepo(q, pool)
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
			crawler := repo.NewCrawlerRepo(q, pool)

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

			crawler := repo.NewCrawlerRepo(q, pool)
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
			_, err := repo.NewCrawlerRepo(q, pool).UpdateSetting(ctx, &model.AdminSetting{
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

// withCrawlTx runs fn in a rollback-only transaction with empty crawl_runs
// and discovery_keywords tables, keeping tests independent of local data.
func withCrawlTx(
	t *testing.T,
	pool *pgxpool.Pool,
	fn func(ctx context.Context, tx pgx.Tx, q *db.Queries),
) {
	t.Helper()

	ctx := t.Context()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("failed to begin test tx: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "delete from crawl_runs"); err != nil {
		t.Fatalf("failed to empty crawl_runs: %v", err)
	}
	if _, err := tx.Exec(ctx, "delete from discovery_keywords"); err != nil {
		t.Fatalf("failed to empty discovery_keywords: %v", err)
	}

	fn(ctx, tx, db.New(tx))
}

func insertKeyword(
	ctx context.Context,
	t *testing.T,
	tx pgx.Tx,
	keyword string,
	language model.Language,
	active bool,
) uuid.UUID {
	t.Helper()

	var id uuid.UUID
	err := tx.QueryRow(ctx,
		`insert into discovery_keywords (keyword, language, is_active)
		 values ($1, $2, $3) returning id`,
		keyword, int(language), active,
	).Scan(&id)
	if err != nil {
		t.Fatalf("failed to insert keyword %q: %v", keyword, err)
	}
	return id
}

// startRun creates a running run of kind through the repo.
func startRun(
	ctx context.Context,
	t *testing.T,
	crawler data.CrawlerRepoer,
	kind model.CrawlKind,
) *model.CrawlRun {
	t.Helper()

	run := &model.CrawlRun{Kind: kind, Trigger: model.CrawlTriggerScheduled}
	if err := crawler.CreateRun(ctx, run); err != nil {
		t.Fatalf("failed to create %s run: %v", kind, err)
	}
	return run
}

func setRunStartedAt(
	ctx context.Context,
	t *testing.T,
	tx pgx.Tx,
	id uuid.UUID,
	startedAt time.Time,
) {
	t.Helper()

	_, err := tx.Exec(ctx, "update crawl_runs set started_at = $2 where id = $1", id, startedAt)
	if err != nil {
		t.Fatalf("failed to set started_at: %v", err)
	}
}

type runRow struct {
	Status     string
	FinishedAt *time.Time
	VideoFound int
	VideoNew   int
	Error      *string
}

func readRun(ctx context.Context, t *testing.T, tx pgx.Tx, id uuid.UUID) runRow {
	t.Helper()

	var row runRow
	err := tx.QueryRow(ctx,
		`select status, finished_at, video_found, video_new, error
		 from crawl_runs where id = $1`,
		id,
	).Scan(&row.Status, &row.FinishedAt, &row.VideoFound, &row.VideoNew, &row.Error)
	if err != nil {
		t.Fatalf("failed to read run %s: %v", id, err)
	}
	return row
}

func TestCrawlerRepo_Keywords(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("ListActiveKeywords_ReturnsOnlyActiveSortedByKeyword", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			insertKeyword(ctx, t, tx, "zeta", model.LanguageEnglish, true)
			insertKeyword(ctx, t, tx, "alpha", model.LanguageFrench, true)
			insertKeyword(ctx, t, tx, "inactive", model.LanguageEnglish, false)

			got, err := repo.NewCrawlerRepo(q, pool).ListActiveKeywords(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != 2 {
				t.Fatalf("expected 2 active keywords, got %d: %+v", len(got), got)
			}
			if got[0].Keyword != "alpha" || got[1].Keyword != "zeta" {
				t.Errorf("expected alpha then zeta, got %q then %q", got[0].Keyword, got[1].Keyword)
			}
			if got[0].Language != model.LanguageFrench || got[1].Language != model.LanguageEnglish {
				t.Errorf("languages were not mapped: %v, %v", got[0].Language, got[1].Language)
			}
			if !got[0].IsActive || got[0].LastRunAt != nil {
				t.Errorf("expected an active keyword that never ran, got %+v", got[0])
			}
			if got[0].ID == uuid.Nil || got[0].CreatedAt.IsZero() || got[0].UpdatedAt.IsZero() {
				t.Errorf("expected id and timestamps to be set, got %+v", got[0])
			}
		})
	})

	t.Run("ListActiveKeywords_NoKeywords", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			got, err := repo.NewCrawlerRepo(q, pool).ListActiveKeywords(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != 0 {
				t.Errorf("expected no keywords, got %+v", got)
			}
		})
	})

	t.Run("MarkKeywordRun_StampsOnlyThatKeyword", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			first := insertKeyword(ctx, t, tx, "a-keyword", model.LanguageEnglish, true)
			insertKeyword(ctx, t, tx, "b-keyword", model.LanguageEnglish, true)
			crawler := repo.NewCrawlerRepo(q, pool)

			if err := crawler.MarkKeywordRun(ctx, first); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got, err := crawler.ListActiveKeywords(ctx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != 2 {
				t.Fatalf("expected 2 keywords, got %d", len(got))
			}
			if got[0].LastRunAt == nil {
				t.Error("expected last_run_at to be set on the marked keyword")
			}
			if got[1].LastRunAt != nil {
				t.Errorf("expected the other keyword untouched, got %v", got[1].LastRunAt)
			}
		})
	})

	t.Run("MarkKeywordRun_UnknownIDReturnsNotFound", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			err := repo.NewCrawlerRepo(q, pool).MarkKeywordRun(ctx, uuid.New())
			if !errors.Is(err, data.ErrNotFound) {
				t.Errorf("expected data.ErrNotFound, got %v", err)
			}
		})
	})
}

func TestCrawlerRepo_Runs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("CreateRun_FillsRunAndStoresIt", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			run := &model.CrawlRun{
				Kind:    model.CrawlKindMonitor,
				Trigger: model.CrawlTriggerManual,
				Params:  []byte(`{"max":5}`),
			}
			if err := repo.NewCrawlerRepo(q, pool).CreateRun(ctx, run); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if run.ID == uuid.Nil || run.StartedAt.IsZero() {
				t.Errorf("expected ID and StartedAt to be filled, got %+v", run)
			}
			if run.Status != model.CrawlStatusRunning {
				t.Errorf("expected status running, got %q", run.Status)
			}

			var kind, trigger, maxParam string
			var triggeredBy *uuid.UUID
			err := tx.QueryRow(ctx,
				`select kind, trigger, triggered_by, params->>'max'
				 from crawl_runs where id = $1`,
				run.ID,
			).Scan(&kind, &trigger, &triggeredBy, &maxParam)
			if err != nil {
				t.Fatalf("failed to read run: %v", err)
			}
			if kind != "monitor" || trigger != "manual" || maxParam != "5" {
				t.Errorf("unexpected stored run: %q %q %q", kind, trigger, maxParam)
			}
			if triggeredBy != nil {
				t.Errorf("expected no triggered_by, got %v", triggeredBy)
			}
		})
	})

	t.Run("CreateRun_EmptyParamsDefaultsToEmptyObject", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			run := startRun(ctx, t, repo.NewCrawlerRepo(q, pool), model.CrawlKindDiscover)

			var params string
			err := tx.QueryRow(ctx, "select params::text from crawl_runs where id = $1", run.ID).
				Scan(&params)
			if err != nil {
				t.Fatalf("failed to read params: %v", err)
			}
			if params != "{}" {
				t.Errorf("expected {}, got %s", params)
			}
		})
	})

	t.Run("CreateRun_RecordsTriggeredBy", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			user := newTestUser(uuid.NewString() + "@example.com")
			if err := repo.NewUserRepo(q, pool).Create(ctx, user); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			run := &model.CrawlRun{
				Kind:        model.CrawlKindDiscover,
				Trigger:     model.CrawlTriggerManual,
				TriggeredBy: &user.ID,
			}
			if err := repo.NewCrawlerRepo(q, pool).CreateRun(ctx, run); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var triggeredBy *uuid.UUID
			err := tx.QueryRow(ctx, "select triggered_by FROM crawl_runs where id = $1", run.ID).
				Scan(&triggeredBy)
			if err != nil {
				t.Fatalf("failed to read run: %v", err)
			}
			if triggeredBy == nil || *triggeredBy != user.ID {
				t.Errorf("expected triggered_by %s, got %v", user.ID, triggeredBy)
			}
		})
	})

	t.Run("CreateRun_AllowsRunningRunsOfDifferentKinds", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			crawler := repo.NewCrawlerRepo(q, pool)
			startRun(ctx, t, crawler, model.CrawlKindDiscover)
			startRun(ctx, t, crawler, model.CrawlKindMonitor)
		})
	})

	t.Run("CreateRun_AllowsNewRunAfterFinish", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			crawler := repo.NewCrawlerRepo(q, pool)
			first := startRun(ctx, t, crawler, model.CrawlKindDiscover)
			first.Status = model.CrawlStatusSucceeded
			if err := crawler.FinishRun(ctx, first); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			startRun(ctx, t, crawler, model.CrawlKindDiscover)
		})
	})

	t.Run("CreateRun_RejectsSecondRunningRunOfSameKind", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			crawler := repo.NewCrawlerRepo(q, pool)
			startRun(ctx, t, crawler, model.CrawlKindDiscover)

			// Must stay the last database call: the failed insert aborts the tx.
			err := crawler.CreateRun(ctx, &model.CrawlRun{
				Kind:    model.CrawlKindDiscover,
				Trigger: model.CrawlTriggerManual,
			})
			if !errors.Is(err, data.ErrConflict) {
				t.Errorf("expected data.ErrConflict, got %v", err)
			}
		})
	})

	t.Run("CreateRun_RejectsUnknownKind", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			err := repo.NewCrawlerRepo(q, pool).CreateRun(ctx, &model.CrawlRun{
				Kind:    model.CrawlKind("bogus"), // violates CHECK (kind IN (...))
				Trigger: model.CrawlTriggerManual,
			})
			if !errors.Is(err, data.ErrInternal) {
				t.Errorf("expected data.ErrInternal, got %v", err)
			}
		})
	})

	t.Run("FinishRun_SavesStatusAndCounts", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			crawler := repo.NewCrawlerRepo(q, pool)
			run := startRun(ctx, t, crawler, model.CrawlKindDiscover)
			run.Status = model.CrawlStatusSucceeded
			run.VideoFound = 12
			run.VideoNew = 5

			if err := crawler.FinishRun(ctx, run); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := readRun(ctx, t, tx, run.ID)
			if got.Status != "succeeded" || got.VideoFound != 12 || got.VideoNew != 5 {
				t.Errorf("unexpected finished run: %+v", got)
			}
			if got.FinishedAt == nil {
				t.Error("expected finished_at to be set")
			}
			if got.Error != nil {
				t.Errorf("expected no error text, got %q", *got.Error)
			}
		})
	})

	t.Run("FinishRun_FailedStoresError", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			crawler := repo.NewCrawlerRepo(q, pool)
			run := startRun(ctx, t, crawler, model.CrawlKindMonitor)
			msg := "apify timed out"
			run.Status = model.CrawlStatusFailed
			run.Error = &msg

			if err := crawler.FinishRun(ctx, run); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := readRun(ctx, t, tx, run.ID)
			if got.Status != "failed" {
				t.Errorf("expected status failed, got %q", got.Status)
			}
			if got.Error == nil || *got.Error != msg {
				t.Errorf("expected error %q, got %v", msg, got.Error)
			}
		})
	})

	t.Run("GetLastRunStartedAt_NoRunReturnsNotFound", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			got, err := repo.NewCrawlerRepo(q, pool).
				GetLastRunStartedAt(ctx, model.CrawlKindDiscover)
			if !errors.Is(err, data.ErrNotFound) {
				t.Fatalf("expected data.ErrNotFound, got %v", err)
			}
			if got != nil {
				t.Errorf("expected no time on error, got %v", got)
			}
		})
	})

	t.Run("GetLastRunStartedAt_ReturnsLatestRunOfKind", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			crawler := repo.NewCrawlerRepo(q, pool)
			base := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)

			older := startRun(ctx, t, crawler, model.CrawlKindDiscover)
			setRunStartedAt(ctx, t, tx, older.ID, base)
			older.Status = model.CrawlStatusSucceeded
			if err := crawler.FinishRun(ctx, older); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			newer := startRun(ctx, t, crawler, model.CrawlKindDiscover)
			setRunStartedAt(ctx, t, tx, newer.ID, base.Add(time.Hour))

			// A later run of another kind must not count.
			other := startRun(ctx, t, crawler, model.CrawlKindMonitor)
			setRunStartedAt(ctx, t, tx, other.ID, base.Add(2*time.Hour))

			got, err := crawler.GetLastRunStartedAt(ctx, model.CrawlKindDiscover)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil || !got.Equal(base.Add(time.Hour)) {
				t.Errorf("expected %v, got %v", base.Add(time.Hour), got)
			}
		})
	})

	t.Run("FailStaleRuns_FailsOnlyOldRunsOfKind", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			crawler := repo.NewCrawlerRepo(q, pool)
			twoHoursAgo := time.Now().Add(-2 * time.Hour)

			staleDiscover := startRun(ctx, t, crawler, model.CrawlKindDiscover)
			setRunStartedAt(ctx, t, tx, staleDiscover.ID, twoHoursAgo)
			staleMonitor := startRun(ctx, t, crawler, model.CrawlKindMonitor)
			setRunStartedAt(ctx, t, tx, staleMonitor.ID, twoHoursAgo)

			if err := crawler.FailStaleRuns(ctx, model.CrawlKindDiscover, time.Hour); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			failed := readRun(ctx, t, tx, staleDiscover.ID)
			if failed.Status != "failed" || failed.FinishedAt == nil {
				t.Errorf("expected the stale discover run to be failed and finished: %+v", failed)
			}
			if failed.Error == nil || *failed.Error == "" {
				t.Error("expected an error text on the failed run")
			}

			untouched := readRun(ctx, t, tx, staleMonitor.ID)
			if untouched.Status != "running" || untouched.FinishedAt != nil {
				t.Errorf("expected the monitor run to stay running: %+v", untouched)
			}

			// The point of the cleanup: a new run of that kind can start again.
			startRun(ctx, t, crawler, model.CrawlKindDiscover)
		})
	})

	t.Run("FailStaleRuns_KeepsRecentRuns", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			crawler := repo.NewCrawlerRepo(q, pool)
			recent := startRun(ctx, t, crawler, model.CrawlKindDiscover)

			if err := crawler.FailStaleRuns(ctx, model.CrawlKindDiscover, time.Hour); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got := readRun(ctx, t, tx, recent.ID); got.Status != "running" {
				t.Errorf("expected the recent run to stay running, got %q", got.Status)
			}
		})
	})

	t.Run("FailStaleRuns_LeavesFinishedRunsAlone", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			crawler := repo.NewCrawlerRepo(q, pool)
			run := startRun(ctx, t, crawler, model.CrawlKindDiscover)
			run.Status = model.CrawlStatusSucceeded
			if err := crawler.FinishRun(ctx, run); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			setRunStartedAt(ctx, t, tx, run.ID, time.Now().Add(-2*time.Hour))

			if err := crawler.FailStaleRuns(ctx, model.CrawlKindDiscover, time.Hour); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := readRun(ctx, t, tx, run.ID)
			if got.Status != "succeeded" || got.Error != nil {
				t.Errorf("expected the finished run to be untouched: %+v", got)
			}
		})
	})
}
