package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
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

var bareURLRegex = regexp.MustCompile(`^\s*(https?://(www\.)?(vt\.)?tiktok\.com/\S+)\s*$`)

const (
	toolCheckHealthClaim = "check_health_claim"
	toolTranscribeVideo  = "transcribe_video"

	// conversationHistoryLimit bounds how many recent messages are fed to
	// the LLM as context for each turn.
	conversationHistoryLimit = 20

	// agentBuilderTimeout is only used to satisfy v1beta's builder
	// validation (it requires Timeout > 0) — the agent turn's real
	// timeout is governed by AgentTurnWorker's River job Timeout.
	agentBuilderTimeout = 2 * time.Minute

	agentSystemPrompt = `You are Wobsongo's assistant, helping WhatsApp users verify health claims and videos that are potentially hoaxes.
If a user sends a TikTok link, use the transcribe_video tool.
If a user presents a claim that needs verification (with or without a video), use the check_health_claim tool.
You may call both tools simultaneously in a single turn if relevant.
If the question can be answered based on the conversation history (e.g., the user asks for a re-explanation of a previous video), answer directly without using tools.
For greetings or casual conversation, respond naturally and concisely, in keeping with the context of a health fact-checking platform.`
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
}

// NewAgentService creates an AgentService and initializes the underlying
// AgenticGoKit agent.
func NewAgentService(
	conversationRepo data.ConversationRepoer,
	apifyService *ApifyService,
	claimService *ClaimService,
	llmClient *external.AgentLLMClient,
	llmProvider, llmModel, llmBaseURL, llmAPIKey string,
) (*AgentService, error) {
	s := &AgentService{
		conversationRepo: conversationRepo,
		apifyService:     apifyService,
		claimService:     claimService,
		llmClient:        llmClient,
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
		ctx, jid, model.ConversationRoleUser, text,
	); err != nil {
		return nil, fmt.Errorf("failed to store inbound message: %w", err)
	}

	if match := bareURLRegex.FindStringSubmatch(text); match != nil {
		resp, err := s.apifyService.TriggerExtraction(ctx, match[1], "", jid)
		if err != nil {
			return nil, fmt.Errorf("failed to trigger fast-path extraction: %w", err)
		}
		return &dto.AgentInboundResponse{Status: resp.Status, JobID: resp.JobID}, nil
	}

	extractionID := uuid.New().String()
	if err := s.conversationRepo.EnqueueAgentTurn(ctx, queue.AgentTurnJob{
		Jid:          jid,
		ExtractionID: extractionID,
		UserText:     text,
	}); err != nil {
		return nil, fmt.Errorf("failed to enqueue agent turn: %w", err)
	}

	return &dto.AgentInboundResponse{Status: dto.StatusProcessing, JobID: extractionID}, nil
}

// RunTurn processes a single AgentTurnJob by loading conversation history,
// running the agent, and saving the assistant's response.
func (s *AgentService) RunTurn(ctx context.Context, job queue.AgentTurnJob) (string, error) {
	if job.SystemNote != "" {
		if err := s.conversationRepo.AppendMessage(
			ctx, job.Jid, model.ConversationRoleSystem, job.SystemNote,
		); err != nil {
			return "", fmt.Errorf("failed to store system note: %w", err)
		}
	}

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

	runCtx := withAgentTurnContext(ctx, &agentTurnContext{jid: job.Jid, messages: messages})

	result, err := s.agent.Run(runCtx, job.CurrentTurnInput())
	if err != nil {
		return "", fmt.Errorf("agent run failed: %w", err)
	}

	if err := s.conversationRepo.AppendMessage(
		ctx, job.Jid, model.ConversationRoleAssistant, result.Content,
	); err != nil {
		log.Printf("[AgentService] failed to store assistant reply for jid=%s: %v", job.Jid, err)
	}

	return result.Content, nil
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
	tools := []external.AgentToolDefinition{checkHealthClaimToolDef(), transcribeVideoToolDef()}

	first, err := s.llmClient.Complete(ctx, messages, tools)
	if err != nil {
		return "", fmt.Errorf("first LLM completion failed: %w", err)
	}

	if len(first.ToolCalls) == 0 {
		return first.Content, nil
	}

	messages = append(messages, external.AgentChatMessage{
		Role:      "assistant",
		Content:   first.Content,
		ToolCalls: first.ToolCalls,
	})

	for _, tc := range first.ToolCalls {
		messages = append(messages, external.AgentChatMessage{
			Role:       "tool",
			ToolCallID: tc.ID,
			Content:    s.executeTool(ctx, turnCtx.jid, tc),
		})
	}

	second, err := s.llmClient.Complete(ctx, messages, nil)
	if err != nil {
		return "", fmt.Errorf("follow-up LLM completion failed: %w", err)
	}

	return second.Content, nil
}

// executeTool runs a tool call and returns its result. Tool errors are
// returned as JSON so the LLM can handle them in its response.
func (s *AgentService) executeTool(
	ctx context.Context,
	jid string,
	tc external.AgentToolCall,
) string {
	switch tc.Function.Name {
	case toolCheckHealthClaim:
		return s.executeCheckHealthClaim(ctx, tc)
	case toolTranscribeVideo:
		return s.executeTranscribeVideo(ctx, jid, tc)
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

func (s *AgentService) executeTranscribeVideo(
	ctx context.Context,
	jid string,
	tc external.AgentToolCall,
) string {
	var args struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
		return fmt.Sprintf(`{"error": "invalid arguments: %s"}`, err.Error())
	}

	resp, err := s.apifyService.TriggerExtraction(ctx, strings.TrimSpace(args.URL), "", jid)
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to start video processing: %s"}`, err.Error())
	}

	return fmt.Sprintf(
		`{"status": "%s", "note": "video processing started in the background; `+
			`a follow-up message with the result will be sent separately once it's ready"}`,
		resp.Status,
	)
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

func transcribeVideoToolDef() external.AgentToolDefinition {
	return external.AgentToolDefinition{
		Type: "function",
		Function: external.AgentToolFunctionDef{
			Name: toolTranscribeVideo,
			Description: "Start transcribing and fact-checking a TikTok video URL. Runs in the " +
				"background; the result arrives as a separate follow-up message.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "The TikTok video URL mentioned by the user.",
					},
				},
				"required": []string{"url"},
			},
		},
	}
}
