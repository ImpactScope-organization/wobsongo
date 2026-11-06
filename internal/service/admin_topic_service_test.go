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

func boolPtr(b bool) *bool { return &b }

// withTopicTx makes WithTx run fn on the mock itself, like a transaction that
// commits, and lets the lock succeed unless a test overrides it.
func withTopicTx(repo *mockrepo.TopicRepoerMock) *mockrepo.TopicRepoerMock {
	repo.WithTxFunc = func(_ context.Context, fn func(data.TopicRepoer) error) error {
		return fn(repo)
	}
	if repo.LockActiveGuardFunc == nil {
		repo.LockActiveGuardFunc = func(context.Context) error { return nil }
	}
	return repo
}

func TestAdminTopicService_List(t *testing.T) {
	t.Run("PassesPagingAndFilterToRepo", func(t *testing.T) {
		page := &dto.PaginationResults[model.Topic]{Page: 2, PerPage: 5, TotalItems: 7}
		var gotQuery data.SupportsPagination
		var gotFilter data.AdminListFilter
		repo := &mockrepo.TopicRepoerMock{
			PaginateFunc: func(
				_ context.Context,
				q data.SupportsPagination,
				f data.AdminListFilter,
			) (*dto.PaginationResults[model.Topic], error) {
				gotQuery, gotFilter = q, f
				return page, nil
			},
		}
		query := &dto.TopicListDTO{
			PaginationDTO: dto.PaginationDTO{Page: 2, PerPage: 5},
			IsActive:      boolPtr(true),
			Search:        "heal",
		}

		got, err := NewAdminTopicService(repo).List(t.Context(), query)
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
		if gotFilter.IsActive == nil || !*gotFilter.IsActive || gotFilter.Search != "heal" {
			t.Errorf("expected the filters to reach the repo, got %+v", gotFilter)
		}
	})

	t.Run("PassesThroughRepoError", func(t *testing.T) {
		repo := &mockrepo.TopicRepoerMock{
			PaginateFunc: func(
				context.Context, data.SupportsPagination, data.AdminListFilter,
			) (*dto.PaginationResults[model.Topic], error) {
				return nil, data.ErrInternal
			},
		}

		_, err := NewAdminTopicService(repo).List(t.Context(), &dto.TopicListDTO{})
		if !errors.Is(err, data.ErrInternal) {
			t.Errorf("expected data.ErrInternal, got %v", err)
		}
	})
}

func TestAdminTopicService_Get(t *testing.T) {
	id := uuid.New()

	t.Run("ReturnsTopic", func(t *testing.T) {
		repo := &mockrepo.TopicRepoerMock{
			GetByIDFunc: func(_ context.Context, id uuid.UUID) (*model.Topic, error) {
				return &model.Topic{ID: id, Name: "Health"}, nil
			},
		}

		got, err := NewAdminTopicService(repo).Get(t.Context(), id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.ID != id || got.Name != "Health" {
			t.Errorf("unexpected topic: %+v", got)
		}
	})

	t.Run("PassesThroughNotFound", func(t *testing.T) {
		repo := &mockrepo.TopicRepoerMock{
			GetByIDFunc: func(context.Context, uuid.UUID) (*model.Topic, error) {
				return nil, data.ErrNotFound
			},
		}

		_, err := NewAdminTopicService(repo).Get(t.Context(), id)
		if !errors.Is(err, data.ErrNotFound) {
			t.Errorf("expected data.ErrNotFound, got %v", err)
		}
	})
}

func TestAdminTopicService_Create(t *testing.T) {
	newRepo := func(saved **model.Topic) *mockrepo.TopicRepoerMock {
		return &mockrepo.TopicRepoerMock{
			CreateFunc: func(_ context.Context, topic *model.Topic) error {
				*saved = topic
				topic.ID = uuid.New()
				return nil
			},
		}
	}

	t.Run("TrimsAndDefaultsToActive", func(t *testing.T) {
		var saved *model.Topic

		got, err := NewAdminTopicService(newRepo(&saved)).Create(t.Context(),
			&dto.CreateTopicRequest{Name: "  Health ", Description: "  Vaccines  "})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if saved == nil || saved.Name != "Health" || saved.Description != "Vaccines" ||
			!saved.IsActive {
			t.Errorf("unexpected topic sent to repo: %+v", saved)
		}
		if got.ID == uuid.Nil {
			t.Errorf("expected the stored ID in the result, got %+v", got)
		}
	})

	t.Run("ExplicitInactive", func(t *testing.T) {
		var saved *model.Topic

		_, err := NewAdminTopicService(newRepo(&saved)).Create(t.Context(),
			&dto.CreateTopicRequest{Name: "Health", IsActive: boolPtr(false)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if saved == nil || saved.IsActive {
			t.Errorf("expected an inactive topic to reach the repo, got %+v", saved)
		}
	})

	t.Run("PassesThroughConflict", func(t *testing.T) {
		repo := &mockrepo.TopicRepoerMock{
			CreateFunc: func(context.Context, *model.Topic) error { return data.ErrConflict },
		}

		got, err := NewAdminTopicService(repo).Create(t.Context(),
			&dto.CreateTopicRequest{Name: "Health"})
		if !errors.Is(err, data.ErrConflict) || got != nil {
			t.Errorf("expected nil and data.ErrConflict, got %+v, %v", got, err)
		}
	})
}

func TestAdminTopicService_Update(t *testing.T) {
	id := uuid.New()
	req := &dto.UpdateTopicRequest{
		Name:        " Health ",
		Description: " Desc ",
		IsActive:    boolPtr(false),
	}

	t.Run("Success_TrimsAndKeepsID", func(t *testing.T) {
		var saved *model.Topic
		repo := withTopicTx(&mockrepo.TopicRepoerMock{
			UpdateFunc: func(_ context.Context, topic *model.Topic) error {
				saved = topic
				return nil
			},
		})

		got, err := NewAdminTopicService(repo).Update(t.Context(), id, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if saved == nil || saved.ID != id || saved.Name != "Health" ||
			saved.Description != "Desc" || saved.IsActive {
			t.Errorf("unexpected topic sent to repo: %+v", saved)
		}
		if got != saved {
			t.Errorf("expected the saved topic to be returned")
		}
		if n := len(repo.GetByIDCalls()); n != 0 {
			t.Errorf("expected no lookup on success, got %d", n)
		}
	})

	t.Run("NotFoundAndTopicExists_IsLastActiveTopic", func(t *testing.T) {
		repo := withTopicTx(&mockrepo.TopicRepoerMock{
			UpdateFunc: func(context.Context, *model.Topic) error { return data.ErrNotFound },
			GetByIDFunc: func(context.Context, uuid.UUID) (*model.Topic, error) {
				return &model.Topic{ID: id, IsActive: true}, nil
			},
		})

		_, err := NewAdminTopicService(repo).Update(t.Context(), id, req)
		if !errors.Is(err, ErrLastActiveTopic) {
			t.Errorf("expected ErrLastActiveTopic, got %v", err)
		}
		if !errors.Is(err, data.ErrConflict) {
			t.Errorf("expected ErrLastActiveTopic to wrap data.ErrConflict, got %v", err)
		}
	})

	t.Run("NotFoundAndTopicMissing_IsNotFound", func(t *testing.T) {
		repo := withTopicTx(&mockrepo.TopicRepoerMock{
			UpdateFunc: func(context.Context, *model.Topic) error { return data.ErrNotFound },
			GetByIDFunc: func(context.Context, uuid.UUID) (*model.Topic, error) {
				return nil, data.ErrNotFound
			},
		})

		_, err := NewAdminTopicService(repo).Update(t.Context(), id, req)
		if !errors.Is(err, data.ErrNotFound) || errors.Is(err, ErrLastActiveTopic) {
			t.Errorf("expected plain data.ErrNotFound, got %v", err)
		}
	})

	for name, repoErr := range map[string]error{
		"Conflict": data.ErrConflict,
		"Internal": data.ErrInternal,
	} {
		t.Run("PassesThrough_"+name, func(t *testing.T) {
			repo := withTopicTx(&mockrepo.TopicRepoerMock{
				UpdateFunc: func(context.Context, *model.Topic) error { return repoErr },
			})

			_, err := NewAdminTopicService(repo).Update(t.Context(), id, req)
			if !errors.Is(err, repoErr) {
				t.Errorf("expected %v, got %v", repoErr, err)
			}
			if n := len(repo.GetByIDCalls()); n != 0 {
				t.Errorf("expected no lookup, got %d", n)
			}
		})
	}

	t.Run("LocksBeforeUpdate", func(t *testing.T) {
		var order []string
		repo := withTopicTx(&mockrepo.TopicRepoerMock{
			LockActiveGuardFunc: func(context.Context) error {
				order = append(order, "lock")
				return nil
			},
			UpdateFunc: func(context.Context, *model.Topic) error {
				order = append(order, "update")
				return nil
			},
		})

		if _, err := NewAdminTopicService(repo).Update(t.Context(), id, req); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(order) != 2 || order[0] != "lock" || order[1] != "update" {
			t.Errorf("expected lock before update, got %v", order)
		}
	})

	t.Run("LockError_SkipsUpdate", func(t *testing.T) {
		repo := withTopicTx(&mockrepo.TopicRepoerMock{
			LockActiveGuardFunc: func(context.Context) error { return data.ErrInternal },
		})

		_, err := NewAdminTopicService(repo).Update(t.Context(), id, req)
		if !errors.Is(err, data.ErrInternal) {
			t.Errorf("expected data.ErrInternal, got %v", err)
		}
		if n := len(repo.UpdateCalls()); n != 0 {
			t.Errorf("expected no update after a failed lock, got %d", n)
		}
		if n := len(repo.GetByIDCalls()); n != 0 {
			t.Errorf("expected no lookup, got %d", n)
		}
	})

	t.Run("WithTxError_PassesThrough", func(t *testing.T) {
		repo := &mockrepo.TopicRepoerMock{
			WithTxFunc: func(context.Context, func(data.TopicRepoer) error) error {
				return data.ErrInternal
			},
		}

		_, err := NewAdminTopicService(repo).Update(t.Context(), id, req)
		if !errors.Is(err, data.ErrInternal) {
			t.Errorf("expected data.ErrInternal, got %v", err)
		}
		if n := len(repo.UpdateCalls()); n != 0 {
			t.Errorf("expected no update, got %d", n)
		}
	})
}
