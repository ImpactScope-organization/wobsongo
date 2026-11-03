package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// AdminKeywordService lets an admin manage the keywords used to discover videos.
type AdminKeywordService struct {
	repo data.KeywordRepoer
}

// NewAdminKeywordService constructs an AdminKeywordService.
func NewAdminKeywordService(repo data.KeywordRepoer) *AdminKeywordService {
	return &AdminKeywordService{repo: repo}
}

// List returns a page of keywords.
func (s *AdminKeywordService) List(
	ctx context.Context,
	query *dto.KeywordListDTO,
) (*dto.PaginationResults[model.DiscoveryKeyword], error) {
	return s.repo.Paginate(ctx, query, data.AdminListFilter{
		IsActive: query.IsActive,
		Search:   query.Search,
	})
}

// Get returns one keyword.
func (s *AdminKeywordService) Get(
	ctx context.Context,
	id uuid.UUID,
) (*model.DiscoveryKeyword, error) {
	return s.repo.GetByID(ctx, id)
}

// Create adds a keyword. It returns data.ErrConflict when the same keyword
// and language already exist.
func (s *AdminKeywordService) Create(
	ctx context.Context,
	req *dto.CreateKeywordRequest,
) (*model.DiscoveryKeyword, error) {
	language, err := model.ParseLanguage(req.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to parse keyword language: %w", err)
	}

	keyword := &model.DiscoveryKeyword{
		Keyword:  normalizeKeyword(req.Keyword),
		Language: language,
		IsActive: req.IsActive == nil || *req.IsActive,
	}
	if err := s.repo.Create(ctx, keyword); err != nil {
		return nil, err
	}
	return keyword, nil
}

// Update replaces a keyword.
func (s *AdminKeywordService) Update(
	ctx context.Context,
	id uuid.UUID,
	req *dto.UpdateKeywordRequest,
) (*model.DiscoveryKeyword, error) {
	language, err := model.ParseLanguage(req.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to parse keyword language: %w", err)
	}

	keyword := &model.DiscoveryKeyword{
		ID:       id,
		Keyword:  normalizeKeyword(req.Keyword),
		Language: language,
		IsActive: *req.IsActive,
	}
	if err := s.repo.Update(ctx, keyword); err != nil {
		return nil, err
	}
	return keyword, nil
}

// Delete removes a keyword.
func (s *AdminKeywordService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// normalizeKeyword trims the keyword, collapses inner spaces and lower-cases
// it, so "Vaccin  COVID" and "vaccin covid" are the same keyword.
func normalizeKeyword(keyword string) string {
	return strings.ToLower(strings.Join(strings.Fields(keyword), " "))
}
