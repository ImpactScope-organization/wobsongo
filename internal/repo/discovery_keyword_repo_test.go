package repo_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/repo"
	"github.com/impactscope-organization/wobsongo/internal/testhelpers"
	"github.com/jackc/pgx/v5"
)

func TestKeywordRepo(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	pool, _ := testhelpers.SetupTestDB(t)
	defer pool.Close()

	t.Run("Create_FillsStoredFields", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			keywords := repo.NewKeywordRepo(q)
			keyword := &model.DiscoveryKeyword{
				Keyword: "vaccin", Language: model.LanguageFrench, IsActive: true,
			}

			if err := keywords.Create(ctx, keyword); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if keyword.ID == uuid.Nil || keyword.LanguageCode != "fr" ||
				keyword.CreatedAt.IsZero() || keyword.UpdatedAt.IsZero() ||
				keyword.LastRunAt != nil {
				t.Errorf("unexpected stored fields: %+v", keyword)
			}

			got, err := keywords.GetByID(ctx, keyword.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Keyword != "vaccin" || got.Language != model.LanguageFrench || !got.IsActive {
				t.Errorf("unexpected keyword: %+v", got)
			}
		})
	})

	t.Run("Create_SameKeywordInOtherLanguage_IsAllowed_SameLanguage_Conflicts", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			keywords := repo.NewKeywordRepo(q)
			for _, language := range []model.Language{model.LanguageFrench, model.LanguageEnglish} {
				err := keywords.Create(ctx, &model.DiscoveryKeyword{
					Keyword: "vaccin", Language: language, IsActive: true,
				})
				if err != nil {
					t.Fatalf("unexpected error for %s: %v", language, err)
				}
			}

			// Keep this last: a unique violation aborts the transaction.
			err := keywords.Create(ctx, &model.DiscoveryKeyword{
				Keyword: "vaccin", Language: model.LanguageFrench, IsActive: true,
			})
			if !errors.Is(err, data.ErrConflict) {
				t.Errorf("expected ErrConflict, got %v", err)
			}
		})
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			_, err := repo.NewKeywordRepo(q).GetByID(ctx, uuid.New())
			if !errors.Is(err, data.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})
	})

	t.Run("Update_ReplacesEditableFields", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			id := insertKeyword(ctx, t, tx, "old", model.LanguageEnglish, true)
			keywords := repo.NewKeywordRepo(q)

			updated := &model.DiscoveryKeyword{
				ID: id, Keyword: "new", Language: model.LanguageFrench, IsActive: false,
			}
			if err := keywords.Update(ctx, updated); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if updated.Keyword != "new" || updated.LanguageCode != "fr" || updated.IsActive ||
				updated.CreatedAt.IsZero() {
				t.Errorf("expected the stored fields to be filled, got %+v", updated)
			}

			got, err := keywords.GetByID(ctx, id)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Keyword != "new" || got.Language != model.LanguageFrench || got.IsActive {
				t.Errorf("update was not stored: %+v", got)
			}
		})
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			err := repo.NewKeywordRepo(q).Update(ctx, &model.DiscoveryKeyword{
				ID: uuid.New(), Keyword: "new", Language: model.LanguageEnglish, IsActive: true,
			})
			if !errors.Is(err, data.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})
	})

	t.Run("Update_ConflictOnTakenKeyword", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			insertKeyword(ctx, t, tx, "taken", model.LanguageEnglish, true)
			id := insertKeyword(ctx, t, tx, "free", model.LanguageEnglish, true)

			err := repo.NewKeywordRepo(q).Update(ctx, &model.DiscoveryKeyword{
				ID: id, Keyword: "taken", Language: model.LanguageEnglish, IsActive: true,
			})
			if !errors.Is(err, data.ErrConflict) {
				t.Errorf("expected ErrConflict, got %v", err)
			}
		})
	})

	t.Run("Delete_RemovesKeyword", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			id := insertKeyword(ctx, t, tx, "gone", model.LanguageEnglish, true)
			keywords := repo.NewKeywordRepo(q)

			if err := keywords.Delete(ctx, id); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, err := keywords.GetByID(ctx, id); !errors.Is(err, data.ErrNotFound) {
				t.Errorf("expected the keyword to be gone, got %v", err)
			}
		})
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, _ pgx.Tx, q *db.Queries) {
			err := repo.NewKeywordRepo(q).Delete(ctx, uuid.New())
			if !errors.Is(err, data.ErrNotFound) {
				t.Errorf("expected ErrNotFound, got %v", err)
			}
		})
	})

	t.Run("Paginate_FiltersAndPages", func(t *testing.T) {
		withCrawlTx(t, pool, func(ctx context.Context, tx pgx.Tx, q *db.Queries) {
			// Ordered by keyword, then language (en before fr):
			// grippe, paludisme, vaccin/en, vaccin/fr.
			insertKeyword(ctx, t, tx, "vaccin", model.LanguageFrench, true)
			insertKeyword(ctx, t, tx, "vaccin", model.LanguageEnglish, true)
			insertKeyword(ctx, t, tx, "grippe", model.LanguageFrench, false)
			insertKeyword(ctx, t, tx, "paludisme", model.LanguageFrench, true)
			keywords := repo.NewKeywordRepo(q)

			list := func(page, perPage int32, filter data.AdminListFilter,
			) *dto.PaginationResults[model.DiscoveryKeyword] {
				t.Helper()
				got, err := keywords.Paginate(
					ctx, &dto.PaginationDTO{Page: page, PerPage: perPage}, filter,
				)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return got
			}
			names := func(p *dto.PaginationResults[model.DiscoveryKeyword]) []string {
				out := make([]string, 0, len(p.Items))
				for _, k := range p.Items {
					out = append(out, k.Keyword)
				}
				return out
			}
			active, inactive := true, false

			first := list(1, 3, data.AdminListFilter{})
			if first.Page != 1 || first.PerPage != 3 || first.TotalItems != 4 ||
				!slices.Equal(names(first), []string{"grippe", "paludisme", "vaccin"}) {
				t.Errorf("unexpected first page: %+v", first)
			}

			second := list(2, 3, data.AdminListFilter{})
			if second.Page != 2 || second.TotalItems != 4 || len(second.Items) != 1 ||
				second.Items[0].Keyword != "vaccin" ||
				second.Items[0].Language != model.LanguageFrench {
				t.Errorf("unexpected second page: %+v", second)
			}

			if got := list(1, 10, data.AdminListFilter{IsActive: &inactive}); got.TotalItems != 1 ||
				!slices.Equal(names(got), []string{"grippe"}) {
				t.Errorf("unexpected inactive filter result: %+v", got)
			}
			if got := list(1, 10, data.AdminListFilter{IsActive: &active}); got.TotalItems != 3 {
				t.Errorf("expected 3 active keywords, got %d", got.TotalItems)
			}

			if got := list(1, 10, data.AdminListFilter{Search: "VAC"}); got.TotalItems != 2 ||
				!slices.Equal(names(got), []string{"vaccin", "vaccin"}) {
				t.Errorf("expected the search to ignore case, got %+v", got)
			}
			if got := list(1, 10, data.AdminListFilter{Search: "  vac  "}); got.TotalItems != 2 {
				t.Errorf("expected the search to be trimmed, got %d", got.TotalItems)
			}

			if got := list(
				1,
				10,
				data.AdminListFilter{Search: "grip", IsActive: &inactive},
			); got.TotalItems != 1 {
				t.Errorf("expected search and active filter to combine, got %d", got.TotalItems)
			}
			if got := list(
				1,
				10,
				data.AdminListFilter{Search: "grip", IsActive: &active},
			); got.TotalItems != 0 {
				t.Errorf("expected no active match for grip, got %d", got.TotalItems)
			}

			none := list(1, 10, data.AdminListFilter{Search: "zzz"})
			if none.TotalItems != 0 || none.Items == nil || len(none.Items) != 0 {
				t.Errorf("expected an empty, non-nil page, got %+v", none)
			}
		})
	})
}
