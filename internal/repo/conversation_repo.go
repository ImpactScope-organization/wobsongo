package repo

import (
	"context"
	"fmt"
	"math"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/db"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

// conversationRepo implements data.ConversationRepoer.
type conversationRepo struct {
	q             *db.Queries
	pool          *pgxpool.Pool
	riverClientFn func() *river.Client[pgx.Tx]
}

// NewConversationRepo creates a new repository for the agentic bot's
// per-jid conversation history.
func NewConversationRepo(
	q *db.Queries,
	pool *pgxpool.Pool,
	riverClientFn func() *river.Client[pgx.Tx],
) data.ConversationRepoer {
	return &conversationRepo{
		q:             q,
		pool:          pool,
		riverClientFn: riverClientFn,
	}
}

// AppendMessage stores one message for the given jid.
func (r *conversationRepo) AppendMessage(
	ctx context.Context,
	jid string,
	role model.ConversationRole,
	content, phoneNumber, countryCode string,
) error {
	_, err := r.q.InsertConversationMessage(ctx, db.InsertConversationMessageParams{
		Jid:         jid,
		Role:        string(role),
		Content:     content,
		PhoneNumber: pgtype.Text{String: phoneNumber, Valid: phoneNumber != ""},
		CountryCode: pgtype.Text{String: countryCode, Valid: countryCode != ""},
	})
	if err != nil {
		return fmt.Errorf("failed to insert conversation message: %w", mapPostgresError(err))
	}
	return nil
}

// RecentMessages returns up to limit most-recent messages for the given
// jid, in chronological order (oldest first).
func (r *conversationRepo) RecentMessages(
	ctx context.Context,
	jid string,
	limit int,
) ([]model.ConversationMessage, error) {
	if limit > math.MaxInt32 || limit < 0 {
		return nil, fmt.Errorf("limit %d is out of bounds for int32", limit)
	}

	rows, err := r.q.GetRecentConversationMessages(ctx, db.GetRecentConversationMessagesParams{
		Jid:   jid,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get recent conversation messages: %w",
			mapPostgresError(err),
		)
	}

	messages := make([]model.ConversationMessage, len(rows))
	for i := range rows {
		row := &rows[i]
		messages[len(rows)-1-i] = model.ConversationMessage{
			ID:          row.ID,
			Jid:         row.Jid,
			Role:        model.ConversationRole(row.Role),
			Content:     row.Content,
			PhoneNumber: row.PhoneNumber.String,
			CountryCode: row.CountryCode.String,
			CreatedAt:   row.CreatedAt,
		}
	}
	return messages, nil
}

// EnqueueAgentTurn adds a new agent-turn job to the River queue.
func (r *conversationRepo) EnqueueAgentTurn(ctx context.Context, payload queue.AgentTurnJob) error {
	_, err := r.riverClientFn().Insert(ctx, payload, nil)
	if err != nil {
		return fmt.Errorf("failed to insert agent turn job into river queue: %w", err)
	}
	return nil
}
