package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type topicRepo struct {
	q    *db.Queries
	pool *pgxpool.Pool
	tx   pgx.Tx
}

func NewTopicRepo(q *db.Queries, pool *pgxpool.Pool) data.TopicRepoer {
	return &topicRepo{q: q, pool: pool}
}

func (r *topicRepo) WithTx(ctx context.Context, fn func(data.TopicRepoer) error) error {
	if r.tx != nil {
		return fn(r)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	repoWithTx := &topicRepo{q: r.q.WithTx(tx), pool: r.pool, tx: tx}
	if err := fn(repoWithTx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *topicRepo) LockActiveGuard(ctx context.Context) error {
	if err := r.q.LockActiveTopicGuard(ctx); err != nil {
		return fmt.Errorf("failed to lock topics: %w", mapPostgresError(err))
	}
	return nil
}

// Paginate returns a filtered page of topics.
func (r *topicRepo) Paginate(
	ctx context.Context,
	q data.SupportsPagination,
	filter data.AdminListFilter,
) (*dto.PaginationResults[model.Topic], error) {
	return paginateAdmin(q, filter, "topics",
		func(a adminPageArgs) ([]db.Topic, error) {
			return r.q.PaginateTopics(ctx, db.PaginateTopicsParams(a))
		},
		func(a adminCountArgs) (int64, error) {
			return r.q.CountTopics(ctx, db.CountTopicsParams(a))
		},
		toModelTopic,
	)
}

// GetByID returns the topic with the given ID.
func (r *topicRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Topic, error) {
	row, err := r.q.GetTopicByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get topic: %w", mapPostgresError(err))
	}
	return toModelTopic(&row), nil
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
