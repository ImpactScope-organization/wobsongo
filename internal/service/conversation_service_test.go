package service

import (
	"context"
	"errors"
	"testing"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/utils"
)

func TestConversationService_UserActivity(t *testing.T) {
	t.Run("ReturnsRepoActivity", func(t *testing.T) {
		want := &model.UserActivity{Users: 3, Messages: 41}
		repo := &mockrepo.ConversationRepoerMock{
			UserActivityFunc: func(
				context.Context,
				data.UserActivityFilter,
			) (*model.UserActivity, error) {
				return want, nil
			},
		}
		svc := NewConversationService(repo)

		got, err := svc.UserActivity(t.Context(), &dto.UserActivityDTO{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("expected the repo's activity %+v, got %+v", want, got)
		}
	})

	t.Run("PassesDaysToRepo", func(t *testing.T) {
		repo := &mockrepo.ConversationRepoerMock{
			UserActivityFunc: func(
				context.Context,
				data.UserActivityFilter,
			) (*model.UserActivity, error) {
				return &model.UserActivity{}, nil
			},
		}
		svc := NewConversationService(repo)

		if _, err := svc.UserActivity(
			t.Context(),
			&dto.UserActivityDTO{Days: utils.PtrOf(int32(7))},
		); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := svc.UserActivity(t.Context(), &dto.UserActivityDTO{}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		calls := repo.UserActivityCalls()
		if len(calls) != 2 {
			t.Fatalf("expected 2 repo calls, got %d", len(calls))
		}
		if calls[0].Filter.Days == nil || *calls[0].Filter.Days != 7 {
			t.Errorf("expected days=7 to reach the repo, got %v", calls[0].Filter.Days)
		}
		if calls[1].Filter.Days != nil {
			t.Errorf("expected no days filter, got %d", *calls[1].Filter.Days)
		}
	})

	t.Run("PropagatesRepoError", func(t *testing.T) {
		wantErr := errors.New("boom")
		repo := &mockrepo.ConversationRepoerMock{
			UserActivityFunc: func(
				context.Context,
				data.UserActivityFilter,
			) (*model.UserActivity, error) {
				return nil, wantErr
			},
		}
		svc := NewConversationService(repo)

		got, err := svc.UserActivity(t.Context(), &dto.UserActivityDTO{})
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected error %v, got %v", wantErr, err)
		}
		if got != nil {
			t.Errorf("expected no activity on error, got %+v", got)
		}
	})
}
