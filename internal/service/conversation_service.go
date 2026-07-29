package service

import (
	"context"

	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
)

// ConversationService wraps the agentic bot's per-jid conversation history repo
type ConversationService struct {
	conversationRepo data.ConversationRepoer
}

// NewConversationService creates a new ConversationService.
func NewConversationService(conversationRepo data.ConversationRepoer) *ConversationService {
	return &ConversationService{conversationRepo: conversationRepo}
}

// AppendMessage stores one message (user, assistant, or system) for the given jid.
func (s *ConversationService) AppendMessage(
	ctx context.Context,
	jid string,
	role model.ConversationRole,
	content, phoneNumber, countryCode string,
) error {
	return s.conversationRepo.AppendMessage(ctx, jid, role, content, phoneNumber, countryCode)
}

// RecentMessages returns up to limit most-recent messages for the given
// jid, in chronological order (oldest first).
func (s *ConversationService) RecentMessages(
	ctx context.Context,
	jid string,
	limit int,
) ([]model.ConversationMessage, error) {
	return s.conversationRepo.RecentMessages(ctx, jid, limit)
}

// EnqueueAgentTurn adds a new agent-turn job to the River queue.
func (s *ConversationService) EnqueueAgentTurn(
	ctx context.Context,
	payload queue.AgentTurnJob,
) error {
	return s.conversationRepo.EnqueueAgentTurn(ctx, payload)
}
