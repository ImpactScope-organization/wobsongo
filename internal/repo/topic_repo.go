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

type topicRepo struct {
	q *db.Queries
}

// NewTopicRepo creates the admin repository of topics.
func NewTopicRepo(q *db.Queries) data.TopicRepoer {
	return &topicRepo{q: q}
}

// Paginate returns a filtered page of topics.
func (r *topicRepo) Paginate(
	ctx context.Context,
	q data.SupportsPagination,
	filter data.AdminListFilter,
) (*dto.PaginationResults[model.Topic], error) {
	limit, offset := q.Limit(), q.Offset()
	isActive, search := filterBool(filter.IsActive), filterText(filter.Search)

	rows, err := r.q.PaginateTopics(ctx, db.PaginateTopicsParams{
		IsActive:   isActive,
		Search:     search,
		PageLimit:  limit,
		PageOffset: offset,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list topics: %w", mapPostgresError(err))
	}

	total, err := r.q.CountTopics(ctx, db.CountTopicsParams{IsActive: isActive, Search: search})
	if err != nil {
		return nil, fmt.Errorf("failed to count topics: %w", mapPostgresError(err))
	}

	items := make([]model.Topic, 0, len(rows))
	for i := range rows {
		items = append(items, *toModelTopic(&rows[i]))
	}

	return &dto.PaginationResults[model.Topic]{
		Page:       pageOf(limit, offset),
		PerPage:    int(limit),
		TotalItems: int(total),
		Items:      items,
	}, nil
}

// GetByID returns the topic with the given ID.
func (r *topicRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Topic, error) {
	row, err := r.q.GetTopicByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic: %w", mapPostgresError(err))
	}
	return toModelTopic(&row), nil
}

// CountActive returns how many topics are active.
func (r *topicRepo) CountActive(ctx context.Context) (int64, error) {
	count, err := r.q.CountActiveTopics(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to count active topics: %w", mapPostgresError(err))
	}
	return count, nil
}

// Create inserts a topic.
func (r *topicRepo) Create(ctx context.Context, topic *model.Topic) error {
	row, err := r.q.CreateTopic(ctx, db.CreateTopicParams{
		Name:        topic.Name,
		Description: topic.Description,
		IsActive:    topic.IsActive,
	})
	if err != nil {
		return fmt.Errorf("failed to create topic: %w", mapPostgresError(err))
	}

	*topic = *toModelTopic(&row)
	return nil
}

// Update replaces the editable fields of a topic.
func (r *topicRepo) Update(ctx context.Context, topic *model.Topic) error {
	row, err := r.q.UpdateTopic(ctx, db.UpdateTopicParams{
		ID:          topic.ID,
		Name:        topic.Name,
		Description: topic.Description,
		IsActive:    topic.IsActive,
	})
	if err != nil {
		return fmt.Errorf("failed to update topic: %w", mapPostgresError(err))
	}

	*topic = *toModelTopic(&row)
	return nil
}

func toModelTopic(row *db.Topic) *model.Topic {
	return &model.Topic{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		IsActive:    row.IsActive,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
