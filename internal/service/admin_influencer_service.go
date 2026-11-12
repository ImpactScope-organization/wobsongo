package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// AdminInfluencerService lets an admin manage the influencers that are monitored.
type AdminInfluencerService struct {
	repo data.InfluencerRepoer
}

// NewAdminInfluencerService constructs an AdminInfluencerService.
func NewAdminInfluencerService(repo data.InfluencerRepoer) *AdminInfluencerService {
	return &AdminInfluencerService{repo: repo}
}

// List returns a page of influencers.
func (s *AdminInfluencerService) List(
	ctx context.Context,
	query *dto.InfluencerListDTO,
) (*dto.PaginationResults[model.Influencer], error) {
	return s.repo.Paginate(ctx, query, data.InfluencerListFilter{
		AdminListFilter: data.AdminListFilter{
			IsActive: query.IsActive,
			Search:   query.Search,
		},
		Source: query.Source,
		Sort:   query.Sort,
	})
}

// Get returns one influencer.
func (s *AdminInfluencerService) Get(ctx context.Context, id uuid.UUID) (*model.Influencer, error) {
	return s.repo.GetByID(ctx, id)
}

// Create adds a TikTok influencer to monitor. The profile fields are filled
// by the next monitor run, which starts right away because no check is
// scheduled yet. It returns data.ErrConflict when the username already exists.
func (s *AdminInfluencerService) Create(
	ctx context.Context,
	req *dto.CreateInfluencerRequest,
) (*model.Influencer, error) {
	username := strings.TrimPrefix(strings.TrimSpace(req.Username), "@")

	influencer := &model.Influencer{
		Platform:   tiktokPlatform,
		Username:   username,
		ProfileURL: "https://www.tiktok.com/@" + username,
		Source:     model.InfluencerSourceSeed,
	}
	if err := s.repo.Create(ctx, influencer); err != nil {
		return nil, err
	}
	return influencer, nil
}

// SetActive turns monitoring of an influencer on or off.
func (s *AdminInfluencerService) SetActive(
	ctx context.Context,
	id uuid.UUID,
	active bool,
) (*model.Influencer, error) {
	return s.repo.SetActive(ctx, id, active)
}

// TopCreators returns the influencers that posted in the query's window, the
// most posts first.
func (s *AdminInfluencerService) TopCreators(
	ctx context.Context,
	query *dto.CreatorListDTO,
) (*dto.PaginationResults[model.TopCreator], error) {
	return s.repo.PaginateTopCreators(ctx, query, data.TopCreatorFilter{
		AdminListFilter: data.AdminListFilter{
			IsActive: query.IsActive,
			Search:   query.Search,
		},
		Since: time.Now().Add(-query.Range.Window()),
	})
}
