package service

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"slices"
	"strings"
	"time"

	agenticgokit "github.com/agenticgokit/agenticgokit/v1beta"
	"github.com/google/uuid"
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

	agentSystemPrompt = `You are Wobsongo's assistant, helping WhatsApp users verify claims about reproductive and sexual health (e.g. contraception, pregnancy, menstruation, STIs/STDs, fertility, sexual wellness, and related myths or misinformation circulating on social media). CRITICAL RULE: You must NEVER answer a health claim, fact, or question from your own knowledge, even if you are confident you know the answer. For ANY message that states, asks about, or implies a health claim in scope — you MUST call the check_health_claim tool first. This applies even to questions phrased conversationally (e.g. "does X cause Y?", "is it true that...", "can X happen if Y?"). The ONLY cases where you should respond WITHOUT calling the tool are: 1. Greetings or casual small talk with no health claim (e.g. "hello", "thank you"). 2. Follow-up questions that only ask you to re-explain or summarize a claim ALREADY checked earlier in this conversation history — answer using that history, don't re-check. 3. Claims clearly outside reproductive/sexual health — say briefly it's outside what you can verify. When in doubt whether something counts as a claim needing verification, ALWAYS call the tool. Because this topic can be sensitive or stigmatized, always respond factually, respectfully, and without judgment.`
)

// AgentService handles bot conversation turns, routing requests to either
// the direct extraction pipeline or the LLM agent, and persists conversation history.
type AgentService struct {
	conversationRepo data.ConversationRepoer
	apifyService     *ApifyService
	claimService     *ClaimService
	agent            agenticgokit.Agent
	agentEnabled     bool
}

// NewAgentService creates an AgentService, registers its tools with
// AgenticGoKit, and builds the underlying agent.
func NewAgentService(
	conversationRepo data.ConversationRepoer,
	apifyService *ApifyService,
	claimService *ClaimService,
	enabled bool,
	llmProvider, llmModel, llmBaseURL, llmAPIKey string,
) (*AgentService, error) {
	s := &AgentService{
		conversationRepo: conversationRepo,
		apifyService:     apifyService,
		claimService:     claimService,
		agentEnabled:     enabled,
	}
	if !enabled {
		return s, nil
	}

	agenticgokit.RegisterInternalTool(toolCheckHealthClaim, func() agenticgokit.Tool {
		return newCheckHealthClaimTool(claimService)
	})

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
			Tools: &agenticgokit.ToolsConfig{
				Enabled: true,
			},
		}).
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
	originalClaim = strings.TrimSpace(embeddedURLRegex.ReplaceAllString(originalClaim, ""))

	result, err := s.claimService.CheckClaim(ctx, &dto.CheckClaimDTO{Text: job.SystemNote})
	if err != nil {
		return "", fmt.Errorf("claim check failed: %w", err)
	}

	content := result.FormattedMessage
	if !result.InScope {
		content = result.RefusalReason
	} else if originalClaim != "" {
		content = fmt.Sprintf(
			"Here's what I found regarding your question — %q:\n\n%s",
			originalClaim,
			content,
		)
	}

	if err := s.conversationRepo.AppendMessage(
		ctx, job.Jid, model.ConversationRoleAssistant, content,
	); err != nil {
		log.Printf("[AgentService] failed to store assistant reply for jid=%s: %v", job.Jid, err)
	}

	return content, nil
}

// runConversationalTurn loads recent history, hands a transcript to the
// AgenticGoKit agent which owns tool discovery and the tool calling
// loop natively and persists the reply.
func (s *AgentService) runConversationalTurn(
	ctx context.Context,
	job queue.AgentTurnJob,
) (string, error) {
	history, err := s.conversationRepo.RecentMessages(ctx, job.Jid, conversationHistoryLimit)
	if err != nil {
		return "", fmt.Errorf("failed to load conversation history: %w", err)
	}

	prompt := formatConversationTranscript(history)
	res, err := s.agent.Run(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("agent run failed: %w", err)
	}

	content := res.Content
	if len(res.ToolsCalled) > 0 {
		content = strings.TrimPrefix(content, toolCheckHealthClaim+" result: ")
	}

	if err := s.conversationRepo.AppendMessage(
		ctx, job.Jid, model.ConversationRoleAssistant, content,
	); err != nil {
		log.Printf("[AgentService] failed to store assistant reply for jid=%s: %v", job.Jid, err)
	}

	return content, nil
}

// formatConversationTranscript renders recent history plus the current
// turn's input as a flat transcript.
func formatConversationTranscript(history []model.ConversationMessage) string {
	var b strings.Builder
	for _, m := range history {
		fmt.Fprintf(&b, "%s: %s\n", m.Role, m.Content)
	}
	return strings.TrimSpace(b.String())
}
