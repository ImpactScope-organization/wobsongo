package service

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestConversationService_MessageVolume(t *testing.T) {
	const day = 24 * time.Hour

	t.Run("ReturnsRepoVolume", func(t *testing.T) {
		want := &model.MessageVolume{Current: 5, Previous: 3, Trend: []int64{2, 3}}
		repo := &mockrepo.ConversationRepoerMock{
			MessageVolumeFunc: func(
				context.Context,
				data.MessageVolumeFilter,
			) (*model.MessageVolume, error) {
				return want, nil
			},
		}
		svc := NewConversationService(repo)

		got, err := svc.MessageVolume(t.Context(), &dto.TimeRangeDTO{Range: dto.TimeRange7d})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != want {
			t.Errorf("expected the repo's volume %+v, got %+v", want, got)
		}
	})

	t.Run("PassesWindowAndBucketToRepo", func(t *testing.T) {
		repo := &mockrepo.ConversationRepoerMock{
			MessageVolumeFunc: func(
				context.Context,
				data.MessageVolumeFilter,
			) (*model.MessageVolume, error) {
				return &model.MessageVolume{}, nil
			},
		}
		svc := NewConversationService(repo)

		ranges := []dto.TimeRange{dto.TimeRange24h, dto.TimeRange7d, dto.TimeRange30d, ""}
		for _, r := range ranges {
			if _, err := svc.MessageVolume(t.Context(), &dto.TimeRangeDTO{Range: r}); err != nil {
				t.Fatalf("range %q: unexpected error: %v", r, err)
			}
		}

		want := []data.MessageVolumeFilter{
			{Window: day, Bucket: time.Hour},
			{Window: 7 * day, Bucket: day},
			{Window: 30 * day, Bucket: day},
			{Window: day, Bucket: time.Hour}, // left out: 24h
		}
		calls := repo.MessageVolumeCalls()
		if len(calls) != len(want) {
			t.Fatalf("expected %d repo calls, got %d", len(want), len(calls))
		}
		for i := range want {
			if calls[i].Filter != want[i] {
				t.Errorf(
					"range %q: expected filter %+v, got %+v",
					ranges[i],
					want[i],
					calls[i].Filter,
				)
			}
		}
	})

	t.Run("PropagatesRepoError", func(t *testing.T) {
		wantErr := errors.New("boom")
		repo := &mockrepo.ConversationRepoerMock{
			MessageVolumeFunc: func(
				context.Context,
				data.MessageVolumeFilter,
			) (*model.MessageVolume, error) {
				return nil, wantErr
			},
		}
		svc := NewConversationService(repo)

		got, err := svc.MessageVolume(t.Context(), &dto.TimeRangeDTO{})
		if !errors.Is(err, wantErr) {
			t.Fatalf("expected error %v, got %v", wantErr, err)
		}
		if got != nil {
			t.Errorf("expected no volume on error, got %+v", got)
		}
	})
}
