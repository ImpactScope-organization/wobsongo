package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

// narrativeRepo implements data.NarrativeRepoer.
type narrativeRepo struct {
	q             *db.Queries
	pool          *pgxpool.Pool
	tx            pgx.Tx
	riverClientFn func() *river.Client[pgx.Tx]
}

// NewNarrativeRepo creates a new repository for the claim-to-narrative flow.
func NewNarrativeRepo(
	q *db.Queries,
	pool *pgxpool.Pool,
	riverClientFn func() *river.Client[pgx.Tx],
) data.NarrativeRepoer {
	return &narrativeRepo{q: q, pool: pool, riverClientFn: riverClientFn}
}

// WithTx runs fn inside a transaction.
func (r *narrativeRepo) WithTx(ctx context.Context, fn func(data.NarrativeRepoer) error) error {
	if r.tx != nil {
		return fn(r)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	repoWithTx := &narrativeRepo{
		q:             r.q.WithTx(tx),
		pool:          r.pool,
		tx:            tx,
		riverClientFn: r.riverClientFn,
	}
	if err := fn(repoWithTx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ListActiveTopics returns every active topic, ordered by name.
func (r *narrativeRepo) ListActiveTopics(ctx context.Context) ([]model.Topic, error) {
	rows, err := r.q.ListActiveTopics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active topics: %w", mapPostgresError(err))
	}
	topics := make([]model.Topic, len(rows))
	for i := range rows {
		topics[i] = model.Topic{
			ID:          rows[i].ID,
			Name:        rows[i].Name,
			Description: rows[i].Description,
			IsActive:    rows[i].IsActive,
			CreatedAt:   rows[i].CreatedAt,
			UpdatedAt:   rows[i].UpdatedAt,
		}
	}
	return topics, nil
}

// CreateClaim stores a claim, or returns data.ErrConflict when the same
// (message, text) claim already exists.
func (r *narrativeRepo) CreateClaim(ctx context.Context, c *model.Claim) (*model.Claim, error) {
	var messageID pgtype.UUID
	if c.MessageID != nil {
		messageID = pgtype.UUID{Bytes: *c.MessageID, Valid: true}
	}

	citations := c.Citations
	if citations == nil {
		citations = []model.ClaimCitation{}
	}
	rawCitations, err := json.Marshal(citations)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal claim citations: %w", err)
	}

	row, err := r.q.InsertClaim(ctx, db.InsertClaimParams{
		MessageID:               messageID,
		Text:                    c.Text,
		TopicID:                 c.TopicID,
		Language:                int32(c.Language),
		Platform:                c.Platform,
		Format:                  string(c.Format),
		OccurredAt:              c.OccurredAt,
		Verdict:                 int32(c.Verdict),
		Severity:                int32(c.Severity),
		HighRisk:                c.HighRisk,
		HighRiskReason:          c.HighRiskReason,
		RecommendMedicalConsult: c.RecommendMedicalConsult,
		Reasoning:               c.Reasoning,
		BriefReasoning:          c.BriefReasoning,
		Citations:               rawCitations,
		JudgedAt:                c.JudgedAt,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, data.ErrConflict
	}
	if err != nil {
		return nil, fmt.Errorf("failed to insert claim: %w", mapPostgresError(err))
	}
	return toModelClaim(&row)
}

// GetClaim returns the claim with the given id.
func (r *narrativeRepo) GetClaim(ctx context.Context, id uuid.UUID) (*model.Claim, error) {
	row, err := r.q.GetClaim(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get claim: %w", mapPostgresError(err))
	}
	return toModelClaim(&row)
}

// EnqueueClusterClaim adds a cluster-claim job to the River queue, inside the
// current transaction when there is one.
func (r *narrativeRepo) EnqueueClusterClaim(ctx context.Context, claimID uuid.UUID) error {
	payload := queue.ClusterClaimJob{ClaimID: claimID.String()}

	if r.tx != nil {
		if _, err := r.riverClientFn().InsertTx(ctx, r.tx, payload, nil); err != nil {
			return fmt.Errorf("failed to insert cluster claim job into river queue: %w", err)
		}
		return nil
	}
	if _, err := r.riverClientFn().Insert(ctx, payload, nil); err != nil {
		return fmt.Errorf("failed to insert cluster claim job into river queue: %w", err)
	}
	return nil
}

// LockTopic takes a transaction-scoped advisory lock on the topic.
func (r *narrativeRepo) LockTopic(ctx context.Context, topicID uuid.UUID) error {
	if err := r.q.LockTopicForClustering(ctx, topicID.String()); err != nil {
		return fmt.Errorf("failed to lock topic: %w", mapPostgresError(err))
	}
	return nil
}

// ListNarrativesByTopic returns every narrative of a topic.
func (r *narrativeRepo) ListNarrativesByTopic(
	ctx context.Context,
	topicID uuid.UUID,
) ([]model.Narrative, error) {
	rows, err := r.q.ListNarrativesByTopic(ctx, topicID)
	if err != nil {
		return nil, fmt.Errorf("failed to list narratives: %w", mapPostgresError(err))
	}
	narratives := make([]model.Narrative, len(rows))
	for i := range rows {
		narratives[i] = *toModelNarrative(&rows[i])
	}
	return narratives, nil
}

// GetNarrative returns the narrative with the given id.
func (r *narrativeRepo) GetNarrative(ctx context.Context, id uuid.UUID) (*model.Narrative, error) {
	row, err := r.q.GetNarrative(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get narrative: %w", mapPostgresError(err))
	}
	return toModelNarrative(&row), nil
}

// PaginateNarratives returns one page of narratives, newest first.
func (r *narrativeRepo) PaginateNarratives(
	ctx context.Context,
	q data.SupportsPagination,
) (*dto.PaginationResults[model.Narrative], error) {
	limit, offset := q.Limit(), q.Offset()

	rows, err := r.q.PaginateNarratives(
		ctx,
		db.PaginateNarrativesParams{Limit: limit, Offset: offset},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to paginate narratives: %w", mapPostgresError(err))
	}

	total, err := r.q.CountNarratives(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count narratives: %w", mapPostgresError(err))
	}

	items := make([]model.Narrative, len(rows))
	for i := range rows {
		items[i] = *toModelNarrative(&rows[i])
	}

	page := int32(1)
	if limit > 0 {
		page = offset/limit + 1
	}

	return &dto.PaginationResults[model.Narrative]{
		Page:       int(page),
		PerPage:    int(limit),
		TotalItems: int(total),
		Items:      items,
	}, nil
}

// CreateNarrative stores a new active narrative.
func (r *narrativeRepo) CreateNarrative(
	ctx context.Context,
	n *model.Narrative,
) (*model.Narrative, error) {
	row, err := r.q.InsertNarrative(ctx, db.InsertNarrativeParams{
		TopicID:       n.TopicID,
		Statement:     n.Statement,
		Harm:          pgtype.Int2{},
		HarmValidated: false,
		Status:        string(model.NarrativeStatusNeedsCheck),
		IsActive:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert narrative: %w", mapPostgresError(err))
	}
	return toModelNarrative(&row), nil
}

// LinkClaim links a claim to a narrative.
func (r *narrativeRepo) LinkClaim(ctx context.Context, claimID, narrativeID uuid.UUID) error {
	err := r.q.InsertNarrativeClaim(ctx, db.InsertNarrativeClaimParams{
		ClaimID:     claimID,
		NarrativeID: narrativeID,
	})
	if err != nil {
		return fmt.Errorf("failed to link claim to narrative: %w", mapPostgresError(err))
	}
	return nil
}

// GetNewestClaim returns the claim of the narrative with the greatest OccurredAt.
func (r *narrativeRepo) GetNewestClaim(
	ctx context.Context,
	narrativeID uuid.UUID,
) (*model.Claim, error) {
	row, err := r.q.GetNewestClaimForNarrative(ctx, narrativeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get newest claim: %w", mapPostgresError(err))
	}
	return toModelClaim(&row)
}

// UpdateNarrativeEstimate sets the harm and status computed from the newest claim.
func (r *narrativeRepo) UpdateNarrativeEstimate(
	ctx context.Context,
	narrativeID uuid.UUID,
	harm *int,
	status model.NarrativeStatus,
) error {
	var pgHarm pgtype.Int2
	if harm != nil {
		pgHarm = pgtype.Int2{Int16: int16(*harm), Valid: true}
	}
	err := r.q.UpdateNarrativeFromNewestClaim(ctx, db.UpdateNarrativeFromNewestClaimParams{
		ID:     narrativeID,
		Harm:   pgHarm,
		Status: string(status),
	})
	if err != nil {
		return fmt.Errorf("failed to update narrative: %w", mapPostgresError(err))
	}
	return nil
}

// DeactivateStaleNarratives deactivates narratives without a claim since the given time.
func (r *narrativeRepo) DeactivateStaleNarratives(
	ctx context.Context,
	since time.Time,
) (int64, error) {
	n, err := r.q.DeactivateStaleNarratives(ctx, since)
	if err != nil {
		return 0, fmt.Errorf("failed to deactivate narratives: %w", mapPostgresError(err))
	}
	return n, nil
}

func toModelClaim(row *db.Claim) (*model.Claim, error) {
	var citations []model.ClaimCitation
	if len(row.Citations) > 0 {
		if err := json.Unmarshal(row.Citations, &citations); err != nil {
			return nil, fmt.Errorf("failed to unmarshal claim citations: %w", err)
		}
	}

	var messageID *uuid.UUID
	if row.MessageID.Valid {
		id := uuid.UUID(row.MessageID.Bytes)
		messageID = &id
	}

	return &model.Claim{
		ID:                      row.ID,
		MessageID:               messageID,
		Text:                    row.Text,
		TopicID:                 row.TopicID,
		Language:                model.Language(row.Language),
		Platform:                row.Platform,
		Format:                  model.ClaimFormat(row.Format),
		OccurredAt:              row.OccurredAt,
		Verdict:                 model.Verdict(row.Verdict),
		Severity:                model.Severity(row.Severity),
		HighRisk:                row.HighRisk,
		HighRiskReason:          row.HighRiskReason,
		RecommendMedicalConsult: row.RecommendMedicalConsult,
		Reasoning:               row.Reasoning,
		BriefReasoning:          row.BriefReasoning,
		Citations:               citations,
		JudgedAt:                row.JudgedAt,
		CreatedAt:               row.CreatedAt,
	}, nil
}

func toModelNarrative(row *db.Narrative) *model.Narrative {
	n := &model.Narrative{
		ID:            row.ID,
		TopicID:       row.TopicID,
		Statement:     row.Statement,
		HarmValidated: row.HarmValidated,
		Status:        model.NarrativeStatus(row.Status),
		IsActive:      row.IsActive,
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
	}
	if row.Harm.Valid {
		harm := int(row.Harm.Int16)
		n.Harm = &harm
	}
	if row.StatementEditedBy.Valid {
		id := uuid.UUID(row.StatementEditedBy.Bytes)
		n.StatementEditedBy = &id
	}
	if row.StatementEditedAt.Valid {
		at := row.StatementEditedAt.Time
		n.StatementEditedAt = &at
	}
	return n
}
