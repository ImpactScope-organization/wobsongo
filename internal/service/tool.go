package service

import (
	"context"
	"encoding/json"
	"errors"

	agenticgokit "github.com/agenticgokit/agenticgokit/v1beta"
	"github.com/impactscope-organization/wobsongo/internal/dto"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// Tool is a generic AgenticGoKit tool wrapper. It converts the raw
// arguments provided by AgenticGoKit into a typed struct before
// invoking the handler.
type Tool[T any] struct {
	name        string
	description string
	schema      map[string]any
	handler     func(ctx context.Context, args T) (*agenticgokit.ToolResult, error)
}

// NewTool constructs a generic Tool. T is the argument struct the handler
// receives; schema is the JSON schema advertised to the LLM for tool-calling.
func NewTool[T any](
	name, description string,
	schema map[string]any,
	handler func(ctx context.Context, args T) (*agenticgokit.ToolResult, error),
) *Tool[T] {
	return &Tool[T]{
		name:        name,
		description: description,
		schema:      schema,
		handler:     handler,
	}
}

func (t *Tool[T]) Name() string               { return t.name }
func (t *Tool[T]) Description() string        { return t.description }
func (t *Tool[T]) JSONSchema() map[string]any { return t.schema }

func (t *Tool[T]) Execute(
	ctx context.Context,
	raw map[string]any,
) (*agenticgokit.ToolResult, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return &agenticgokit.ToolResult{Success: false, Error: "invalid arguments"}, err
	}

	var args T
	if err := json.Unmarshal(b, &args); err != nil {
		return &agenticgokit.ToolResult{Success: false, Error: "invalid arguments"}, err
	}

	return t.handler(ctx, args)
}

// checkHealthClaimArgs is the argument shape for the check_health_claim tool.
type checkHealthClaimArgs struct {
	Claim string `json:"claim"`
}

// newCheckHealthClaimTool wraps ClaimService.CheckClaim as an AgenticGoKit tool.
func newCheckHealthClaimTool(claimService *ClaimService) agenticgokit.Tool {
	return NewTool[checkHealthClaimArgs](
		toolCheckHealthClaim,
		"Check a health-related claim against Wobsongo's knowledge base "+
			"and return a cited verdict.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"claim": map[string]any{
					"type":        "string",
					"description": "The health claim to check, in the language the user wrote it in.",
				},
			},
			"required": []string{"claim"},
		},
		func(ctx context.Context, args checkHealthClaimArgs) (*agenticgokit.ToolResult, error) {
			turn := agentTurnFromContext(ctx)
			result, err := claimService.CheckClaim(ctx, &dto.CheckClaimDTO{
				Text:   args.Claim,
				Length: string(turn.Settings.Length),
				Source: claimSource(turn.MessageID, turn.Format),
			})
			if err != nil {
				return &agenticgokit.ToolResult{Success: false, Error: err.Error()}, err
			}

			content := result.FormattedMessage
			if !result.InScope {
				content = result.RefusalReason
			}

			return &agenticgokit.ToolResult{Success: true, Content: content}, nil
		},
	)
}

// updateBotSettingsArgs is the argument shape for the update_bot_settings tool.
// The chat's jid is deliberately not an argument: it comes from the turn
// context, so the LLM can never write another chat's settings.
type updateBotSettingsArgs struct {
	Tone   string `json:"tone,omitempty"`
	Length string `json:"length,omitempty"`
}

// newUpdateBotSettingsTool wraps BotSettingsService.Update as an AgenticGoKit
// tool. On success its content is the confirmation message for the user.
func newUpdateBotSettingsTool(botSettingsService *BotSettingsService) agenticgokit.Tool {
	return NewTool[updateBotSettingsArgs](
		toolUpdateBotSettings,
		"Save how the user wants the bot to answer (tone and/or length). "+
			"Only set the fields the user actually asked for.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"tone": map[string]any{
					"type": "string",
					"enum": []string{
						string(model.BotToneFormal),
						string(model.BotToneAcademic),
						string(model.BotToneFriendly),
					},
				},
				"length": map[string]any{
					"type": "string",
					"enum": []string{
						string(model.BotLengthShort),
						string(model.BotLengthMedium),
						string(model.BotLengthLong),
						string(model.BotLengthExpert),
					},
				},
			},
		},
		func(ctx context.Context, args updateBotSettingsArgs) (*agenticgokit.ToolResult, error) {
			jid := agentTurnFromContext(ctx).JID
			if jid == "" {
				err := errors.New("missing jid in turn context")
				return &agenticgokit.ToolResult{Success: false, Error: err.Error()}, err
			}

			tone, length := model.BotTone(args.Tone), model.BotLength(args.Length)
			if _, err := botSettingsService.Update(ctx, jid, tone, length); err != nil {
				return &agenticgokit.ToolResult{Success: false, Error: err.Error()}, err
			}

			return &agenticgokit.ToolResult{
				Success: true,
				Content: botSettingsConfirmation(tone, length),
			}, nil
		},
	)
}
