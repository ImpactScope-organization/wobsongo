package service

import (
	"context"
	"encoding/json"

	agenticgokit "github.com/agenticgokit/agenticgokit/v1beta"
	"github.com/impactscope-organization/wobsongo/internal/dto"
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
			result, err := claimService.CheckClaim(ctx, &dto.CheckClaimDTO{Text: args.Claim})
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
