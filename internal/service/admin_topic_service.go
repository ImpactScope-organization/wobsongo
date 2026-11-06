package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// ErrLastActiveTopic is returned when an update would deactivate the last
// active topic.
var ErrLastActiveTopic = fmt.Errorf("%w: at least one topic must stay active", data.ErrConflict)

// AdminTopicService lets an admin manage the closed list of claim topics.
type AdminTopicService struct {
	repo data.TopicRepoer
}

// NewAdminTopicService constructs an AdminTopicService.
func NewAdminTopicService(repo data.TopicRepoer) *AdminTopicService {
	return &AdminTopicService{repo: repo}
}

// List returns a page of topics.
func (s *AdminTopicService) List(
	ctx context.Context,
	query *dto.TopicListDTO,
) (*dto.PaginationResults[model.Topic], error) {
	return s.repo.Paginate(ctx, query, data.AdminListFilter{
		IsActive: query.IsActive,
		Search:   query.Search,
	})
}

// Get returns one topic.
func (s *AdminTopicService) Get(ctx context.Context, id uuid.UUID) (*model.Topic, error) {
	return s.repo.GetByID(ctx, id)
}

// Create adds a topic. It returns data.ErrConflict when the name is taken.
func (s *AdminTopicService) Create(
	ctx context.Context,
	req *dto.CreateTopicRequest,
) (*model.Topic, error) {
	topic := &model.Topic{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		IsActive:    req.IsActive == nil || *req.IsActive,
	}
	if err := s.repo.Create(ctx, topic); err != nil {
		return nil, err
	}
	return topic, nil
}

// Update replaces a topic. The last active topic cannot be deactivated,
// because the claim analyzer must always have a topic to assign.
func (s *AdminTopicService) Update(
	ctx context.Context,
	id uuid.UUID,
	req *dto.UpdateTopicRequest,
) (*model.Topic, error) {
	topic := &model.Topic{
		ID:          id,
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		IsActive:    *req.IsActive,
	}

	err := s.repo.Update(ctx, topic)
	if err != nil {
		if errors.Is(err, data.ErrNotFound) {
			if _, getErr := s.repo.GetByID(ctx, id); getErr != nil {
				return nil, getErr
			}
			return nil, ErrLastActiveTopic
		}
		return nil, err
	}

	return topic, nil
}
