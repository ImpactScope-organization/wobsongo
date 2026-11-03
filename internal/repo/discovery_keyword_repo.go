package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

type keywordRepo struct {
	q *db.Queries
}

// NewKeywordRepo creates the admin repository of discovery keywords.
func NewKeywordRepo(q *db.Queries) data.KeywordRepoer {
	return &keywordRepo{q: q}
}

// Paginate returns a filtered page of discovery keywords.
func (r *keywordRepo) Paginate(
	ctx context.Context,
	q data.SupportsPagination,
	filter data.AdminListFilter,
) (*dto.PaginationResults[model.DiscoveryKeyword], error) {
	limit, offset := q.Limit(), q.Offset()
	isActive, search := filterBool(filter.IsActive), filterText(filter.Search)

	rows, err := r.q.PaginateDiscoveryKeywords(ctx, db.PaginateDiscoveryKeywordsParams{
		IsActive:   isActive,
		Search:     search,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list discovery keywords: %w", mapPostgresError(err))
	}

	total, err := r.q.CountDiscoveryKeywords(ctx, db.CountDiscoveryKeywordsParams{
		IsActive: isActive,
		Search:   search,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count discovery keywords: %w", mapPostgresError(err))
	}

	items := make([]model.DiscoveryKeyword, 0, len(rows))
	for i := range rows {
		items = append(items, *toModelKeyword(&rows[i]))
	}

	return &dto.PaginationResults[model.DiscoveryKeyword]{
		Page:       pageOf(limit, offset),
		PerPage:    int(limit),
		TotalItems: int(total),
		Items:      items,
	}, nil
}

// GetByID returns the keyword with the given ID.
func (r *keywordRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.DiscoveryKeyword, error) {
	row, err := r.q.GetDiscoveryKeyword(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get discovery keyword: %w", mapPostgresError(err))
	}
	return toModelKeyword(&row), nil
}

// Create inserts a keyword.
func (r *keywordRepo) Create(ctx context.Context, keyword *model.DiscoveryKeyword) error {
	row, err := r.q.CreateDiscoveryKeyword(ctx, db.CreateDiscoveryKeywordParams{
		Keyword:  keyword.Keyword,
		Language: int32(keyword.Language),
		IsActive: keyword.IsActive,
	})
	if err != nil {
		return fmt.Errorf("failed to create discovery keyword: %w", mapPostgresError(err))
	}

	*keyword = *toModelKeyword(&row)
	return nil
}

// Update replaces the editable fields of a keyword.
func (r *keywordRepo) Update(ctx context.Context, keyword *model.DiscoveryKeyword) error {
	row, err := r.q.UpdateDiscoveryKeyword(ctx, db.UpdateDiscoveryKeywordParams{
		ID:       keyword.ID,
		Keyword:  keyword.Keyword,
		Language: int32(keyword.Language),
		IsActive: keyword.IsActive,
	})
	if err != nil {
		return fmt.Errorf("failed to update discovery keyword: %w", mapPostgresError(err))
	}

	*keyword = *toModelKeyword(&row)
	return nil
}

// Delete removes a keyword.
func (r *keywordRepo) Delete(ctx context.Context, id uuid.UUID) error {
	deleted, err := r.q.DeleteDiscoveryKeyword(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete discovery keyword: %w", mapPostgresError(err))
	}
	if deleted == 0 {
		return fmt.Errorf("%w: discovery keyword %s", data.ErrNotFound, id)
	}
	return nil
}

func toModelKeyword(row *db.DiscoveryKeyword) *model.DiscoveryKeyword {
	keyword := &model.DiscoveryKeyword{
		ID:           row.ID,
		Keyword:      row.Keyword,
		Language:     model.Language(row.Language),
		LanguageCode: model.Language(row.Language).String(),
		IsActive:     row.IsActive,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
	if row.LastRunAt.Valid {
		t := row.LastRunAt.Time
		keyword.LastRunAt = &t
	}
	return keyword
}
