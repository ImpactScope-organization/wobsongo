package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/impactscope-organization/wobsongo/internal/data"
)

// narrativeStatementMaxTokens bounds the response: one short sentence.
const narrativeStatementMaxTokens = 200

type NarrativeStatementClient struct {
	baseURL    string
	model      string
	apiKey     string
	httpClient *http.Client
}

// Ensure NarrativeStatementClient implements data.NarrativeStatementGenerator.
var _ data.NarrativeStatementGenerator = (*NarrativeStatementClient)(nil)

// NewNarrativeStatementClient creates a new NarrativeStatementClient.
func NewNarrativeStatementClient(baseURL, model, apiKey string) *NarrativeStatementClient {
	return &NarrativeStatementClient{
		baseURL:    baseURL,
		model:      model,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: analyzerHTTPTimeout},
	}
}

// Generate implements data.NarrativeStatementGenerator.
func (c *NarrativeStatementClient) Generate(ctx context.Context, claim string) (string, error) {
	payload := extractionCompletionRequest{
		Model: c.model,
		Messages: []extractionChatMessage{
			{Role: "user", Content: buildNarrativeStatementPrompt(claim)},
		},
		MaxTokens: narrativeStatementMaxTokens,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal narrative statement request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/v1/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create narrative statement request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to call narrative statement endpoint: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read narrative statement response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"narrative statement endpoint returned error status: %d. Body: %s",
			resp.StatusCode,
			string(respBytes),
		)
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("failed to unmarshal narrative statement response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("narrative statement response contained no choices: %s", respBytes)
	}

	statement := strings.Trim(strings.TrimSpace(parsed.Choices[0].Message.Content), "\"")
	if statement == "" {
		return "", fmt.Errorf("narrative statement response was empty: %s", respBytes)
	}
	return statement, nil
}

// buildNarrativeStatementPrompt builds the instruction that turns one claim
// into the broad statement of the narrative it starts.
func buildNarrativeStatementPrompt(claim string) string {
	var b strings.Builder
	b.WriteString("You name narratives in a Sexual and Reproductive Health misinformation ")
	b.WriteString("tracker. A narrative groups many differently worded claims that share the ")
	b.WriteString("same meaning. Given one claim, write the statement of the narrative it ")
	b.WriteString("belongs to:\n")
	b.WriteString("- exactly one sentence in French, at most 25 words;\n")
	b.WriteString("- deliberately broad: state the general belief, drop incidental details ")
	b.WriteString("such as names, numbers, places or exact wording, so other claims with the ")
	b.WriteString("same meaning fit under it;\n")
	b.WriteString("- phrase it as the belief itself, not as a fact-check or a verdict.\n\n")
	b.WriteString("Respond with ONLY the sentence, no quotes, no commentary.\n\n")
	b.WriteString("Claim:\n\"\"\"\n")
	b.WriteString(claim)
	b.WriteString("\n\"\"\"\n")
	return b.String()
}
