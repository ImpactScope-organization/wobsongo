package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"slices"
	"strings"
	"time"

	agenticgokit "github.com/agenticgokit/agenticgokit/v1beta"
	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/external"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
	"github.com/impactscope-organization/wobsongo/internal/queue"
)

var (
	bareURLRegex     = regexp.MustCompile(`^\s*(https?://(www\.)?(vt\.)?tiktok\.com/\S+)\s*$`)
	embeddedURLRegex = regexp.MustCompile(`https?://(www\.)?(vt\.)?tiktok\.com/\S+`)
	anyURLRegex      = regexp.MustCompile(`https?://\S+`)
)

const (
	toolCheckHealthClaim = "check_health_claim"

	// conversationHistoryLimit bounds how many recent messages are fed to
	// the LLM as context for each turn.
	conversationHistoryLimit = 20

	agentBuilderTimeout = 2 * time.Minute

	agentSystemPrompt = `You are Wobsongo's assistant, helping WhatsApp users verify claims about reproductive and sexual health (e.g. contraception, pregnancy, menstruation, STIs/STDs, fertility, sexual wellness, and related myths or misinformation circulating on social media).
If the user presents a claim in this scope that needs verification, use the check_health_claim tool.
If a claim clearly falls outside reproductive/sexual health (e.g. unrelated general health topics), say briefly that this is outside what you can verify, without using the tool.
If the question can be answered from conversation history (e.g. re-explaining a previous result), answer directly without tools.
For greetings or casual conversation, respond naturally and concisely.
Because this topic can be sensitive or stigmatized, always respond factually, respectfully, and without judgment — never shame or lecture the user for asking.`
)

type agentTurnContextKey struct{}

// agentTurnContext stores the message history for the current turn,
// passed via context because HandlerFunc only accepts a single input string.
type agentTurnContext struct {
	jid      string
	messages []external.AgentChatMessage
}

func withAgentTurnContext(ctx context.Context, turnCtx *agentTurnContext) context.Context {
	return context.WithValue(ctx, agentTurnContextKey{}, turnCtx)
}

func agentTurnFromContext(ctx context.Context) (*agentTurnContext, bool) {
	turnCtx, ok := ctx.Value(agentTurnContextKey{}).(*agentTurnContext)
	return turnCtx, ok
}

// AgentService handles bot conversation turns, routing requests to either
// the direct extraction pipeline or the LLM agent, and persists conversation history.
type AgentService struct {
	conversationRepo data.ConversationRepoer
	apifyService     *ApifyService
	claimService     *ClaimService
	llmClient        *external.AgentLLMClient
	agent            agenticgokit.Agent
	agentEnabled     bool
}

// NewAgentService creates an AgentService and initializes the underlying
// AgenticGoKit agent.
func NewAgentService(
	conversationRepo data.ConversationRepoer,
	apifyService *ApifyService,
	claimService *ClaimService,
	llmClient *external.AgentLLMClient,
	enabled bool,
	llmProvider, llmModel, llmBaseURL, llmAPIKey string,
) (*AgentService, error) {
	s := &AgentService{
		conversationRepo: conversationRepo,
		apifyService:     apifyService,
		claimService:     claimService,
		llmClient:        llmClient,
		agentEnabled:     enabled,
	}
	if !enabled {
		return s, nil
	}

	agent, err := agenticgokit.NewBuilder("WobsongoAgent").
		WithConfig(&agenticgokit.Config{
			Name:         "WobsongoAgent",
			SystemPrompt: agentSystemPrompt,
			Timeout:      agentBuilderTimeout,
			LLM: agenticgokit.LLMConfig{
				Provider: llmProvider,
				Model:    llmModel,
				BaseURL:  llmBaseURL,
				APIKey:   llmAPIKey,
			},
		}).
		WithHandler(s.runHandler).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build agentic workflow agent: %w", err)
	}

	// s.agent is built only to validate the AgenticGoKit configuration at
	// startup. Runtime execution calls runHandler directly because Agent.Run()
	// does not reliably return the handler's result.
	s.agent = agent

	return s, nil
}

// HandleInboundMessage processes inbound WhatsApp messages, routing
// TikTok URLs to the extraction pipeline and all other messages to the agent.
func (s *AgentService) HandleInboundMessage(
	ctx context.Context,
	jid, text string,
) (*dto.AgentInboundResponse, error) {
	if err := s.conversationRepo.AppendMessage(
		ctx,
		jid,
		model.ConversationRoleUser,
		text,
	); err != nil {
		return nil, fmt.Errorf("failed to store inbound message: %w", err)
	}

	if match := bareURLRegex.FindStringSubmatch(text); match != nil {
		resp, err := s.apifyService.TriggerExtraction(ctx, match[1], "", jid, false)
		if err != nil {
			return nil, fmt.Errorf("failed to trigger fast-path extraction: %w", err)
		}
		return &dto.AgentInboundResponse{Status: resp.Status, JobID: resp.JobID}, nil
	}

	if url := embeddedURLRegex.FindString(text); url != "" {
		resp, err := s.apifyService.TriggerExtraction(ctx, url, "", jid, true)
		if err != nil {
			return nil, fmt.Errorf("failed to trigger video extraction: %w", err)
		}
		return &dto.AgentInboundResponse{Status: resp.Status, JobID: resp.JobID}, nil
	}

	if anyURLRegex.MatchString(text) {
		rejectMsg := "Sorry, I can currently only process TikTok video links. Please send a TikTok link if you'd like the video checked."
		if err := s.conversationRepo.AppendMessage(
			ctx, jid, model.ConversationRoleAssistant, rejectMsg,
		); err != nil {
			log.Printf("[AgentService] failed to store rejection reply for jid=%s: %v", jid, err)
		}
		return &dto.AgentInboundResponse{
			Status:  dto.StatusRejected,
			Message: rejectMsg,
		}, nil
	}

	extractionID := uuid.New().String()
	if err := s.conversationRepo.EnqueueAgentTurn(ctx, queue.AgentTurnJob{
		Jid: jid, ExtractionID: extractionID, UserText: text,
	}); err != nil {
		return nil, fmt.Errorf("failed to enqueue agent turn: %w", err)
	}

	return &dto.AgentInboundResponse{Status: dto.StatusProcessing, JobID: extractionID}, nil
}

// RunTurn processes a single AgentTurnJob by loading conversation history,
// running the agent, and saving the assistant's response.
func (s *AgentService) RunTurn(ctx context.Context, job queue.AgentTurnJob) (string, error) {
	if job.SystemNote != "" {
		return s.runVideoContinuation(ctx, job)
	}
	if !s.agentEnabled {
		return s.runFallbackReply(ctx, job)
	}
	return s.runConversationalTurn(ctx, job)
}

func (s *AgentService) runFallbackReply(
	ctx context.Context,
	job queue.AgentTurnJob,
) (string, error) {
	result, err := s.claimService.CheckClaim(ctx, &dto.CheckClaimDTO{Text: job.UserText})
	if err != nil {
		return "", fmt.Errorf("claim check failed: %w", err)
	}

	content := result.FormattedMessage
	if !result.InScope {
		content = result.RefusalReason
	}

	if err := s.conversationRepo.AppendMessage(
		ctx, job.Jid, model.ConversationRoleAssistant, content,
	); err != nil {
		log.Printf("[AgentService] failed to store fallback reply for jid=%s: %v", job.Jid, err)
	}
	return content, nil
}

func (s *AgentService) runVideoContinuation(
	ctx context.Context,
	job queue.AgentTurnJob,
) (string, error) {
	if err := s.conversationRepo.AppendMessage(
		ctx, job.Jid, model.ConversationRoleSystem, job.SystemNote,
	); err != nil {
		return "", fmt.Errorf("failed to store system note: %w", err)
	}

	history, err := s.conversationRepo.RecentMessages(ctx, job.Jid, conversationHistoryLimit)
	if err != nil {
		return "", fmt.Errorf("failed to load conversation history: %w", err)
	}

	originalClaim := ""
	for _, msg := range slices.Backward(history) {
		if msg.Role == model.ConversationRoleUser {
			originalClaim = msg.Content
			break
		}
	}

	combinedText := job.SystemNote
	if originalClaim != "" {
		combinedText = originalClaim + "\n\n" + job.SystemNote
	}

	result, err := s.claimService.CheckClaim(ctx, &dto.CheckClaimDTO{Text: combinedText})
	if err != nil {
		return "", fmt.Errorf("claim check failed: %w", err)
	}

	content := result.FormattedMessage
	if !result.InScope {
		content = result.RefusalReason
	}

	if err := s.conversationRepo.AppendMessage(
		ctx, job.Jid, model.ConversationRoleAssistant, content,
	); err != nil {
		log.Printf("[AgentService] failed to store assistant reply for jid=%s: %v", job.Jid, err)
	}

	return content, nil
}

func (s *AgentService) runConversationalTurn(
	ctx context.Context,
	job queue.AgentTurnJob,
) (string, error) {
	history, err := s.conversationRepo.RecentMessages(ctx, job.Jid, conversationHistoryLimit)
	if err != nil {
		return "", fmt.Errorf("failed to load conversation history: %w", err)
	}

	messages := make([]external.AgentChatMessage, 0, len(history)+1)
	messages = append(
		messages,
		external.AgentChatMessage{Role: "system", Content: agentSystemPrompt},
	)
	for _, m := range history {
		messages = append(
			messages,
			external.AgentChatMessage{Role: string(m.Role), Content: m.Content},
		)
	}

	turnCtx := &agentTurnContext{jid: job.Jid, messages: messages}
	content, err := s.runHandler(withAgentTurnContext(ctx, turnCtx), job.CurrentTurnInput(), nil)
	if err != nil {
		return "", fmt.Errorf("agent handler failed: %w", err)
	}

	if err := s.conversationRepo.AppendMessage(
		ctx, job.Jid, model.ConversationRoleAssistant, content,
	); err != nil {
		log.Printf("[AgentService] failed to store assistant reply for jid=%s: %v", job.Jid, err)
	}

	return content, nil
}

// runHandler executes the LLM tool-calling loop for a single agent turn.
// It uses the LLM client directly because HandlerFunc does not support
// passing tools through caps.LLM.
func (s *AgentService) runHandler(
	ctx context.Context,
	_ string,
	_ *agenticgokit.Capabilities,
) (string, error) {
	turnCtx, ok := agentTurnFromContext(ctx)
	if !ok {
		return "", errors.New("agent handler invoked without turn context")
	}

	messages := turnCtx.messages
	tools := []external.AgentToolDefinition{checkHealthClaimToolDef()}

	first, err := s.llmClient.Complete(ctx, messages, tools)
	if err != nil {
		return "", fmt.Errorf("first LLM completion failed: %w", err)
	}

	if len(first.ToolCalls) == 0 {
		return first.Content, nil
	}

	results := make([]string, 0, len(first.ToolCalls))
	for _, tc := range first.ToolCalls {
		results = append(results, s.executeTool(ctx, turnCtx.jid, tc))
	}
	return strings.Join(results, "\n\n"), nil
}

// executeTool runs a tool call and returns its result. Tool errors are
// returned as JSON so the LLM can handle them in its response.
func (s *AgentService) executeTool(
	ctx context.Context,
	_ string,
	tc external.AgentToolCall,
) string {
	switch tc.Function.Name {
	case toolCheckHealthClaim:
		return s.executeCheckHealthClaim(ctx, tc)
	default:
		return fmt.Sprintf(`{"error": "unknown tool: %s"}`, tc.Function.Name)
	}
}

func (s *AgentService) executeCheckHealthClaim(
	ctx context.Context,
	tc external.AgentToolCall,
) string {
	var args struct {
		ClaimText string `json:"claim_text"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return fmt.Sprintf(`{"error": "invalid arguments: %s"}`, err.Error())
	}

	result, err := s.claimService.CheckClaim(ctx, &dto.CheckClaimDTO{Text: args.ClaimText})
	if err != nil {
		return fmt.Sprintf(`{"error": "claim check failed: %s"}`, err.Error())
	}

	message := result.FormattedMessage
	if !result.InScope {
		message = result.RefusalReason
	}
	return message
}

func checkHealthClaimToolDef() external.AgentToolDefinition {
	return external.AgentToolDefinition{
		Type: "function",
		Function: external.AgentToolFunctionDef{
			Name: toolCheckHealthClaim,
			Description: "Check a health-related claim against Wobsongo's knowledge base " +
				"and return a cited verdict.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"claim_text": map[string]any{
						"type":        "string",
						"description": "The health claim to check, in the language the user wrote it in.",
					},
				},
				"required": []string{"claim_text"},
			},
		},
	}
}
