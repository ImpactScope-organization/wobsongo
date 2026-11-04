package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/mockrepo"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

var settingUpdatedAt = time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)

func newUpdateSettingRequest(interval int) *dto.UpdateAdminSettingRequest {
	return &dto.UpdateAdminSettingRequest{
		DiscoveryIntervalHours: interval,
		MaxResultsKeyword:      30,
		MaxResultsInfluencer:   15,
		MaxCostPerRun:          12.5,
	}
}

// newSettingRepoMock returns a mock whose UpdateSetting reports previous as
// the interval stored before the update and stamps UpdatedAt like the real
// repo does.
func newSettingRepoMock(previous int) *mockrepo.CrawlerRepoerMock {
	return &mockrepo.CrawlerRepoerMock{
		UpdateSettingFunc: func(_ context.Context, setting *model.AdminSetting) (int, error) {
			setting.UpdatedAt = settingUpdatedAt
			return previous, nil
		},
	}
}

func newRescheduleRecorder() *mockrepo.DiscoveryReschedulerMock {
	return &mockrepo.DiscoveryReschedulerMock{
		RescheduleFunc: func(int, bool) {},
	}
}

func TestAdminSettingService_Get(t *testing.T) {
	t.Run("ReturnsStoredSetting", func(t *testing.T) {
		stored := &model.AdminSetting{DiscoveryIntervalHours: 24, MaxResultsKeyword: 20}
		repo := &mockrepo.CrawlerRepoerMock{
			GetSettingFunc: func(context.Context) (*model.AdminSetting, error) {
				return stored, nil
			},
		}

		got, err := NewAdminSettingService(repo, nil).Get(t.Context())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != stored {
			t.Errorf("expected the stored setting, got %+v", got)
		}
	})

	t.Run("PropagatesRepoError", func(t *testing.T) {
		repo := &mockrepo.CrawlerRepoerMock{
			GetSettingFunc: func(context.Context) (*model.AdminSetting, error) {
				return nil, data.ErrInternal
			},
		}

		got, err := NewAdminSettingService(repo, nil).Get(t.Context())
		if err == nil || !errorsIs(err, data.ErrInternal) {
			t.Fatalf("expected ErrInternal, got %v", err)
		}
		if got != nil {
			t.Errorf("expected no setting on error, got %+v", got)
		}
	})
}

func TestAdminSettingService_Update(t *testing.T) {
	updater := uuid.New()

	t.Run("MapsRequestAndRecordsUpdater", func(t *testing.T) {
		repo := newSettingRepoMock(24)
		svc := NewAdminSettingService(repo, newRescheduleRecorder())

		got, err := svc.Update(t.Context(), updater, newUpdateSettingRequest(6))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		calls := repo.UpdateSettingCalls()
		if len(calls) != 1 {
			t.Fatalf("expected 1 repo call, got %d", len(calls))
		}
		saved := calls[0].Setting
		if saved.DiscoveryIntervalHours != 6 || saved.MaxResultsKeyword != 30 ||
			saved.MaxResultsInfluencer != 15 || saved.MaxCostPerRun != 12.5 {
			t.Errorf("request fields were not mapped to the setting: %+v", saved)
		}
		if saved.UpdatedBy == nil || *saved.UpdatedBy != updater {
			t.Errorf("expected UpdatedBy %s, got %v", updater, saved.UpdatedBy)
		}
		if !got.UpdatedAt.Equal(settingUpdatedAt) {
			t.Errorf("expected UpdatedAt filled by the repo, got %v", got.UpdatedAt)
		}
	})

	t.Run("ReschedulesWhenIntervalChanged", func(t *testing.T) {
		rescheduler := newRescheduleRecorder()
		svc := NewAdminSettingService(newSettingRepoMock(24), rescheduler)

		if _, err := svc.Update(t.Context(), updater, newUpdateSettingRequest(6)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		calls := rescheduler.RescheduleCalls()
		if len(calls) != 1 {
			t.Fatalf("expected 1 reschedule, got %d", len(calls))
		}
		if calls[0].IntervalHours != 6 || calls[0].RunOnStart {
			t.Errorf("expected Reschedule(6, false), got %+v", calls[0])
		}
	})

	t.Run("SkipsRescheduleWhenIntervalUnchanged", func(t *testing.T) {
		rescheduler := newRescheduleRecorder()
		svc := NewAdminSettingService(newSettingRepoMock(6), rescheduler)

		if _, err := svc.Update(t.Context(), updater, newUpdateSettingRequest(6)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n := len(rescheduler.RescheduleCalls()); n != 0 {
			t.Errorf("expected no reschedule, got %d", n)
		}
	})

	t.Run("ReschedulesOnFirstSave", func(t *testing.T) {
		// The repo reports 0 when no row existed before.
		rescheduler := newRescheduleRecorder()
		svc := NewAdminSettingService(newSettingRepoMock(0), rescheduler)

		if _, err := svc.Update(t.Context(), updater, newUpdateSettingRequest(24)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n := len(rescheduler.RescheduleCalls()); n != 1 {
			t.Errorf("expected 1 reschedule on the first save, got %d", n)
		}
	})

	t.Run("WorksWithoutRescheduler", func(t *testing.T) {
		svc := NewAdminSettingService(newSettingRepoMock(24), nil)

		if _, err := svc.Update(t.Context(), updater, newUpdateSettingRequest(6)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("RepoError_DoesNotReschedule", func(t *testing.T) {
		repo := &mockrepo.CrawlerRepoerMock{
			UpdateSettingFunc: func(context.Context, *model.AdminSetting) (int, error) {
				return 0, data.ErrInternal
			},
		}
		rescheduler := newRescheduleRecorder()
		svc := NewAdminSettingService(repo, rescheduler)

		got, err := svc.Update(t.Context(), updater, newUpdateSettingRequest(6))
		if !errorsIs(err, data.ErrInternal) {
			t.Fatalf("expected ErrInternal, got %v", err)
		}
		if got != nil {
			t.Errorf("expected no setting on error, got %+v", got)
		}
		if n := len(rescheduler.RescheduleCalls()); n != 0 {
			t.Errorf("expected no reschedule after a failed update, got %d", n)
		}
	})
}

func errorsIs(err, target error) bool { return errors.Is(err, target) }
