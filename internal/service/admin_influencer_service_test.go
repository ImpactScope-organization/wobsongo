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

func TestAdminInfluencerService_List(t *testing.T) {
	t.Run("PassesPagingAndFiltersToRepo", func(t *testing.T) {
		page := &dto.PaginationResults[model.Influencer]{Page: 2, PerPage: 5, TotalItems: 7}
		var gotQuery data.SupportsPagination
		var gotFilter data.InfluencerListFilter
		repo := &mockrepo.InfluencerRepoerMock{
			PaginateFunc: func(
				_ context.Context,
				q data.SupportsPagination,
				f data.InfluencerListFilter,
			) (*dto.PaginationResults[model.Influencer], error) {
				gotQuery, gotFilter = q, f
				return page, nil
			},
		}
		query := &dto.InfluencerListDTO{
			PaginationDTO: dto.PaginationDTO{Page: 2, PerPage: 5},
			IsActive:      boolPtr(false),
			Source:        "discovered",
			Search:        "naf",
		}

		got, err := NewAdminInfluencerService(repo).List(t.Context(), query)
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
		if gotFilter.IsActive == nil || *gotFilter.IsActive ||
			gotFilter.Source != "discovered" || gotFilter.Search != "naf" {
			t.Errorf("expected the filters to reach the repo, got %+v", gotFilter)
		}
	})

	t.Run("PassesThroughRepoError", func(t *testing.T) {
		repo := &mockrepo.InfluencerRepoerMock{
			PaginateFunc: func(
				context.Context, data.SupportsPagination, data.InfluencerListFilter,
			) (*dto.PaginationResults[model.Influencer], error) {
				return nil, data.ErrInternal
			},
		}

		_, err := NewAdminInfluencerService(repo).List(t.Context(), &dto.InfluencerListDTO{})
		if !errors.Is(err, data.ErrInternal) {
			t.Errorf("expected data.ErrInternal, got %v", err)
		}
	})
}

func TestAdminInfluencerService_Get(t *testing.T) {
	id := uuid.New()

	t.Run("ReturnsInfluencer", func(t *testing.T) {
		repo := &mockrepo.InfluencerRepoerMock{
			GetByIDFunc: func(_ context.Context, id uuid.UUID) (*model.Influencer, error) {
				return &model.Influencer{ID: id, Username: "someone"}, nil
			},
		}

		got, err := NewAdminInfluencerService(repo).Get(t.Context(), id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != id || got.Username != "someone" {
			t.Errorf("unexpected influencer: %+v", got)
		}
	})

	t.Run("PassesThroughNotFound", func(t *testing.T) {
		repo := &mockrepo.InfluencerRepoerMock{
			GetByIDFunc: func(context.Context, uuid.UUID) (*model.Influencer, error) {
				return nil, data.ErrNotFound
			},
		}

		_, err := NewAdminInfluencerService(repo).Get(t.Context(), id)
		if !errors.Is(err, data.ErrNotFound) {
			t.Errorf("expected data.ErrNotFound, got %v", err)
		}
	})
}

func TestAdminInfluencerService_Create(t *testing.T) {
	tests := map[string]string{
		"WithAt":        "@Some.One_1",
		"WithoutAt":     "Some.One_1",
		"WithSpaces":    "  @Some.One_1  ",
		"WithSpaceOnly": "  Some.One_1 ",
	}
	for name, input := range tests {
		t.Run("NormalizesUsername_"+name, func(t *testing.T) {
			var saved *model.Influencer
			repo := &mockrepo.InfluencerRepoerMock{
				CreateFunc: func(_ context.Context, i *model.Influencer) error {
					saved = i
					i.ID = uuid.New()
					return nil
				},
			}

			got, err := NewAdminInfluencerService(repo).Create(t.Context(),
				&dto.CreateInfluencerRequest{Username: input})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if saved == nil || saved.Username != "Some.One_1" ||
				saved.Platform != "tiktok" ||
				saved.ProfileURL != "https://www.tiktok.com/@Some.One_1" ||
				saved.Source != model.InfluencerSourceSeed {
				t.Errorf("unexpected influencer sent to repo: %+v", saved)
			}
			if got.ID == uuid.Nil {
				t.Errorf("expected the stored ID in the result, got %+v", got)
			}
		})
	}

	for name, repoErr := range map[string]error{
		"Conflict": data.ErrConflict,
		"Internal": data.ErrInternal,
	} {
		t.Run("PassesThrough_"+name, func(t *testing.T) {
			repo := &mockrepo.InfluencerRepoerMock{
				CreateFunc: func(context.Context, *model.Influencer) error { return repoErr },
			}

			got, err := NewAdminInfluencerService(repo).Create(t.Context(),
				&dto.CreateInfluencerRequest{Username: "someone"})
			if !errors.Is(err, repoErr) || got != nil {
				t.Errorf("expected nil and %v, got %+v, %v", repoErr, got, err)
			}
		})
	}
}

func TestAdminInfluencerService_SetActive(t *testing.T) {
	id := uuid.New()

	for _, active := range []bool{true, false} {
		t.Run("PassesIDAndState", func(t *testing.T) {
			var gotID uuid.UUID
			var gotActive bool
			repo := &mockrepo.InfluencerRepoerMock{
				SetActiveFunc: func(
					_ context.Context, id uuid.UUID, active bool,
				) (*model.Influencer, error) {
					gotID, gotActive = id, active
					return &model.Influencer{ID: id, IsActive: active}, nil
				},
			}

			got, err := NewAdminInfluencerService(repo).SetActive(t.Context(), id, active)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotID != id || gotActive != active || got.IsActive != active {
				t.Errorf("unexpected call: id=%s active=%v result=%+v", gotID, gotActive, got)
			}
		})
	}

	t.Run("PassesThroughNotFound", func(t *testing.T) {
		repo := &mockrepo.InfluencerRepoerMock{
			SetActiveFunc: func(context.Context, uuid.UUID, bool) (*model.Influencer, error) {
				return nil, data.ErrNotFound
			},
		}

		_, err := NewAdminInfluencerService(repo).SetActive(t.Context(), id, true)
		if !errors.Is(err, data.ErrNotFound) {
			t.Errorf("expected data.ErrNotFound, got %v", err)
		}
	})
}
