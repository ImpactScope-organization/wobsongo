package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

var keywordRepoErrors = map[string]error{
	"NotFound": data.ErrNotFound,
	"Conflict": data.ErrConflict,
	"Internal": data.ErrInternal,
}

func TestAdminKeywordService_List(t *testing.T) {
	t.Run("PassesPagingAndFilterToRepo", func(t *testing.T) {
		active := true
		page := &dto.PaginationResults[model.DiscoveryKeyword]{Page: 2, PerPage: 5, TotalItems: 7}
		var gotQuery data.SupportsPagination
		var gotFilter data.AdminListFilter
		repo := &mockrepo.KeywordRepoerMock{
			PaginateFunc: func(
				_ context.Context,
				q data.SupportsPagination,
				f data.AdminListFilter,
			) (*dto.PaginationResults[model.DiscoveryKeyword], error) {
				gotQuery, gotFilter = q, f
				return page, nil
			},
		}
		query := &dto.KeywordListDTO{
			PaginationDTO: dto.PaginationDTO{Page: 2, PerPage: 5},
			IsActive:      &active,
			Search:        "vac",
		}

		got, err := NewAdminKeywordService(repo).List(t.Context(), query)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != page {
			t.Errorf("expected the repo page, got %+v", got)
		}
		if gotQuery.Limit() != 5 || gotQuery.Offset() != 5 {
			t.Errorf(
				"expected limit 5 / offset 5, got %d / %d",
				gotQuery.Limit(),
				gotQuery.Offset(),
			)
		}
		if gotFilter.IsActive == nil || !*gotFilter.IsActive || gotFilter.Search != "vac" {
			t.Errorf("expected the filters to reach the repo, got %+v", gotFilter)
		}
	})

	t.Run("PassesThroughRepoError", func(t *testing.T) {
		repo := &mockrepo.KeywordRepoerMock{
			PaginateFunc: func(
				context.Context, data.SupportsPagination, data.AdminListFilter,
			) (*dto.PaginationResults[model.DiscoveryKeyword], error) {
				return nil, data.ErrInternal
			},
		}

		_, err := NewAdminKeywordService(repo).List(t.Context(), &dto.KeywordListDTO{})
		if !errors.Is(err, data.ErrInternal) {
			t.Errorf("expected data.ErrInternal, got %v", err)
		}
	})
}

func TestAdminKeywordService_Get(t *testing.T) {
	id := uuid.New()

	t.Run("ReturnsKeyword", func(t *testing.T) {
		stored := &model.DiscoveryKeyword{ID: id, Keyword: "vaccin"}
		var gotID uuid.UUID
		repo := &mockrepo.KeywordRepoerMock{
			GetByIDFunc: func(_ context.Context, id uuid.UUID) (*model.DiscoveryKeyword, error) {
				gotID = id
				return stored, nil
			},
		}

		got, err := NewAdminKeywordService(repo).Get(t.Context(), id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != stored || gotID != id {
			t.Errorf("expected keyword %s from the repo, got %+v (asked for %s)", id, got, gotID)
		}
	})

	t.Run("PassesThroughNotFound", func(t *testing.T) {
		repo := &mockrepo.KeywordRepoerMock{
			GetByIDFunc: func(context.Context, uuid.UUID) (*model.DiscoveryKeyword, error) {
				return nil, data.ErrNotFound
			},
		}

		_, err := NewAdminKeywordService(repo).Get(t.Context(), id)
		if !errors.Is(err, data.ErrNotFound) {
			t.Errorf("expected data.ErrNotFound, got %v", err)
		}
	})
}

func TestAdminKeywordService_Create(t *testing.T) {
	yes, no := true, false

	tests := []struct {
		name         string
		req          dto.CreateKeywordRequest
		wantKeyword  string
		wantLanguage model.Language
		wantActive   bool
	}{
		{
			"NormalizesKeyword",
			dto.CreateKeywordRequest{Keyword: "  Vaccin   COVID ", Language: "fr"},
			"vaccin covid", model.LanguageFrench, true,
		},
		{
			"English",
			dto.CreateKeywordRequest{Keyword: "flu", Language: "en"},
			"flu", model.LanguageEnglish, true,
		},
		{
			"ExplicitInactive",
			dto.CreateKeywordRequest{Keyword: "flu", Language: "en", IsActive: &no},
			"flu", model.LanguageEnglish, false,
		},
		{
			"ExplicitActive",
			dto.CreateKeywordRequest{Keyword: "flu", Language: "en", IsActive: &yes},
			"flu", model.LanguageEnglish, true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var saved *model.DiscoveryKeyword
			repo := &mockrepo.KeywordRepoerMock{
				CreateFunc: func(_ context.Context, k *model.DiscoveryKeyword) error {
					saved = k
					k.ID = uuid.New()
					return nil
				},
			}

			got, err := NewAdminKeywordService(repo).Create(t.Context(), &tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != saved || got.ID == uuid.Nil {
				t.Errorf("expected the stored keyword with its ID, got %+v", got)
			}
			if got.Keyword != tt.wantKeyword || got.Language != tt.wantLanguage ||
				got.IsActive != tt.wantActive {
				t.Errorf("unexpected keyword: %+v", got)
			}
		})
	}

	t.Run("InvalidLanguage_DoesNotReachRepo", func(t *testing.T) {
		repo := &mockrepo.KeywordRepoerMock{}

		got, err := NewAdminKeywordService(repo).Create(
			t.Context(), &dto.CreateKeywordRequest{Keyword: "flu", Language: "de"},
		)
		if err == nil || got != nil {
			t.Fatalf("expected an error for an unknown language, got %+v, %v", got, err)
		}
		if n := len(repo.CreateCalls()); n != 0 {
			t.Errorf("expected the repo not to be called, got %d calls", n)
		}
	})

	for name, repoErr := range keywordRepoErrors {
		t.Run("RepoError_"+name, func(t *testing.T) {
			repo := &mockrepo.KeywordRepoerMock{
				CreateFunc: func(context.Context, *model.DiscoveryKeyword) error { return repoErr },
			}

			got, err := NewAdminKeywordService(repo).Create(
				t.Context(), &dto.CreateKeywordRequest{Keyword: "flu", Language: "en"},
			)
			if !errors.Is(err, repoErr) || got != nil {
				t.Errorf("expected %v and no keyword, got %+v, %v", repoErr, got, err)
			}
		})
	}
}

func TestAdminKeywordService_Update(t *testing.T) {
	id := uuid.New()
	inactive := false

	t.Run("ReplacesAllFields", func(t *testing.T) {
		var saved *model.DiscoveryKeyword
		repo := &mockrepo.KeywordRepoerMock{
			UpdateFunc: func(_ context.Context, k *model.DiscoveryKeyword) error {
				saved = k
				return nil
			},
		}

		got, err := NewAdminKeywordService(repo).Update(t.Context(), id, &dto.UpdateKeywordRequest{
			Keyword: " Grippe   A ", Language: "fr", IsActive: &inactive,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != saved || got.ID != id || got.Keyword != "grippe a" ||
			got.Language != model.LanguageFrench || got.IsActive {
			t.Errorf("unexpected keyword: %+v", got)
		}
	})

	t.Run("InvalidLanguage_DoesNotReachRepo", func(t *testing.T) {
		repo := &mockrepo.KeywordRepoerMock{}

		_, err := NewAdminKeywordService(repo).Update(t.Context(), id, &dto.UpdateKeywordRequest{
			Keyword: "flu", Language: "de", IsActive: &inactive,
		})
		if err == nil {
			t.Fatal("expected an error for an unknown language")
		}
		if n := len(repo.UpdateCalls()); n != 0 {
			t.Errorf("expected the repo not to be called, got %d calls", n)
		}
	})

	for name, repoErr := range keywordRepoErrors {
		t.Run("RepoError_"+name, func(t *testing.T) {
			repo := &mockrepo.KeywordRepoerMock{
				UpdateFunc: func(context.Context, *model.DiscoveryKeyword) error { return repoErr },
			}

			got, err := NewAdminKeywordService(
				repo,
			).Update(t.Context(), id, &dto.UpdateKeywordRequest{
				Keyword: "flu", Language: "en", IsActive: &inactive,
			})
			if !errors.Is(err, repoErr) || got != nil {
				t.Errorf("expected %v and no keyword, got %+v, %v", repoErr, got, err)
			}
		})
	}
}

func TestAdminKeywordService_Delete(t *testing.T) {
	id := uuid.New()

	t.Run("DeletesByID", func(t *testing.T) {
		var gotID uuid.UUID
		repo := &mockrepo.KeywordRepoerMock{
			DeleteFunc: func(_ context.Context, id uuid.UUID) error {
				gotID = id
				return nil
			},
		}

		if err := NewAdminKeywordService(repo).Delete(t.Context(), id); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotID != id {
			t.Errorf("expected keyword %s to be deleted, got %s", id, gotID)
		}
	})

	t.Run("PassesThroughNotFound", func(t *testing.T) {
		repo := &mockrepo.KeywordRepoerMock{
			DeleteFunc: func(context.Context, uuid.UUID) error { return data.ErrNotFound },
		}

		err := NewAdminKeywordService(repo).Delete(t.Context(), id)
		if !errors.Is(err, data.ErrNotFound) {
			t.Errorf("expected data.ErrNotFound, got %v", err)
		}
	})
}

func TestNormalizeKeyword(t *testing.T) {
	tests := map[string]struct{ in, want string }{
		"Lowercases":          {"Vaccin", "vaccin"},
		"Trims":               {"  vaccin  ", "vaccin"},
		"CollapsesInnerSpace": {"vaccin   covid", "vaccin covid"},
		"HandlesTabsNewlines": {"vaccin\t\ncovid", "vaccin covid"},
		"KeepsAccents":        {"  Épidémie ", "épidémie"},
		"BlankBecomesEmpty":   {"   ", ""},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := normalizeKeyword(tt.in); got != tt.want {
				t.Errorf("normalizeKeyword(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
