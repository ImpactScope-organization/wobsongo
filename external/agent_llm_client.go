package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// agentLLMHTTPTimeout bounds a single agent-turn completion call.
const agentLLMHTTPTimeout = 2 * time.Minute

// AgentLLMClient implements the agentic bot's LLM calls against a generic
// OpenAI-compatible /v1/chat/completions API that supports tool
type AgentLLMClient struct {
	baseURL    string
	model      string
	apiKey     string
	httpClient *http.Client
}

// NewAgentLLMClient creates a new AgentLLMClient targeting the given base
// URL/model. apiKey may be empty — self-hosted servers often need no auth.
func NewAgentLLMClient(baseURL, model, apiKey string) *AgentLLMClient {
	return &AgentLLMClient{
		baseURL: baseURL,
		model:   model,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: agentLLMHTTPTimeout,
		},
	}
}

// AgentChatMessage is one message in the conversation sent to/received
// from the LLM, following the OpenAI chat-completions message shape.
type AgentChatMessage struct {
	Role       string          `json:"role"`
	Content    string          `json:"content"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	ToolCalls  []AgentToolCall `json:"tool_calls,omitempty"`
}

// AgentToolCall is one structured tool call returned by the LLM.
type AgentToolCall struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Function AgentToolCallFunc `json:"function"`
}

// AgentToolCallFunc holds the name and JSON-encoded arguments of a tool call.
type AgentToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// AgentToolDefinition describes one callable tool in the OpenAI
// function-calling schema shape.
type AgentToolDefinition struct {
	Type     string               `json:"type"`
	Function AgentToolFunctionDef `json:"function"`
}

// AgentToolFunctionDef is the JSON-Schema description of one tool's name,
// purpose, and parameters.
type AgentToolFunctionDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// AgentCompletionResult is the parsed outcome of one completion call.
type AgentCompletionResult struct {
	Content      string
	ToolCalls    []AgentToolCall
	FinishReason string
}

type agentCompletionRequest struct {
	Model      string                `json:"model"`
	Messages   []AgentChatMessage    `json:"messages"`
	Tools      []AgentToolDefinition `json:"tools,omitempty"`
	ToolChoice string                `json:"tool_choice,omitempty"`
}

type agentChatCompletionResponse struct {
	Choices []struct {
		Message      AgentChatMessage `json:"message"`
		FinishReason string           `json:"finish_reason"`
	} `json:"choices"`
}

// Complete sends messages (and, optionally, tool definitions) to the LLM
// and returns either a final text answer or a list of structured tool
// calls for the caller to execute.
func (c *AgentLLMClient) Complete(
	ctx context.Context,
	messages []AgentChatMessage,
	tools []AgentToolDefinition,
) (*AgentCompletionResult, error) {
	payload := agentCompletionRequest{
		Model:    c.model,
		Messages: messages,
	}
	if len(tools) > 0 {
		payload.Tools = tools
		payload.ToolChoice = "auto"
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal agent completion request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent completion request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call agent LLM endpoint: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent completion response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"agent LLM endpoint returned error status: %d. Body: %s",
			resp.StatusCode,
			respBytes,
		)
	}

	var parsed agentChatCompletionResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent completion response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("agent completion response contained no choices: %s", respBytes)
	}

	choice := parsed.Choices[0]
	return &AgentCompletionResult{
		Content:      choice.Message.Content,
		ToolCalls:    choice.Message.ToolCalls,
		FinishReason: choice.FinishReason,
	}, nil
}
