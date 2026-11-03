package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// CrawlSettingService lets an admin read and change the crawler configuration.
type CrawlSettingService struct {
	crawlerRepo data.CrawlerRepoer
	rescheduler data.DiscoveryRescheduler
}

// NewCrawlSettingService constructs a CrawlSettingService.
func NewCrawlSettingService(
	crawlerRepo data.CrawlerRepoer,
	rescheduler data.DiscoveryRescheduler,
) *CrawlSettingService {
	return &CrawlSettingService{crawlerRepo: crawlerRepo, rescheduler: rescheduler}
}

// Get returns the current crawler configuration.
func (s *CrawlSettingService) Get(ctx context.Context) (*model.CrawlerSetting, error) {
	setting, err := s.crawlerRepo.GetSetting(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get crawler setting: %w", err)
	}
	return setting, nil
}

// Update replaces the crawler configuration and records who changed it. When
// the discovery interval changed, the periodic discovery job is replaced so
// the new interval applies right away.
func (s *CrawlSettingService) Update(
	ctx context.Context,
	updatedBy uuid.UUID,
	req *dto.UpdateCrawlerSettingRequest,
) (*model.CrawlerSetting, error) {
	current, err := s.crawlerRepo.GetSetting(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get crawler setting: %w", err)
	}

	setting := &model.CrawlerSetting{
		DiscoveryIntervalHours: req.DiscoveryIntervalHours,
		MaxResultsKeyword:      req.MaxResultsKeyword,
		MaxResultsInfluencer:   req.MaxResultsInfluencer,
		MaxCostPerRun:          req.MaxCostPerRun,
		UpdatedBy:              &updatedBy,
	}
	if err := s.crawlerRepo.UpdateSetting(ctx, setting); err != nil {
		return nil, fmt.Errorf("failed to update crawler setting: %w", err)
	}

	if s.rescheduler != nil && setting.DiscoveryIntervalHours != current.DiscoveryIntervalHours {
		s.rescheduler.Reschedule(setting.DiscoveryIntervalHours, false)
	}
	return setting, nil
}
