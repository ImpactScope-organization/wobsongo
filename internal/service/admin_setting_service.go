package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// AdminSettingService lets an admin read and change the settings of video
// discovery and creator monitoring.
type AdminSettingService struct {
	crawlerRepo data.CrawlerRepoer
	rescheduler data.DiscoveryRescheduler
}

// NewAdminSettingService constructs an AdminSettingService.
func NewAdminSettingService(
	crawlerRepo data.CrawlerRepoer,
	rescheduler data.DiscoveryRescheduler,
) *AdminSettingService {
	return &AdminSettingService{crawlerRepo: crawlerRepo, rescheduler: rescheduler}
}

// Get returns the current settings.
func (s *AdminSettingService) Get(ctx context.Context) (*model.AdminSetting, error) {
	return s.crawlerRepo.GetSetting(ctx)
}

// Update replaces the settings and records who changed them. When the
// discovery interval changed, the periodic discovery job is replaced so the
// new interval applies right away.
func (s *AdminSettingService) Update(
	ctx context.Context,
	updatedBy uuid.UUID,
	req *dto.UpdateAdminSettingRequest,
) (*model.AdminSetting, error) {
	setting := &model.AdminSetting{
		DiscoveryIntervalHours: req.DiscoveryIntervalHours,
		MaxResultsKeyword:      req.MaxResultsKeyword,
		MaxResultsInfluencer:   req.MaxResultsInfluencer,
		MaxCostPerRun:          req.MaxCostPerRun,
		UpdatedBy:              &updatedBy,
	}

	previousInterval, err := s.crawlerRepo.UpdateSetting(ctx, setting)
	if err != nil {
		return nil, err
	}

	if s.rescheduler != nil && setting.DiscoveryIntervalHours != previousInterval {
		s.rescheduler.Reschedule(setting.DiscoveryIntervalHours, false)
	}
	return setting, nil
}
