package external

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/impactscope-organization/wobsongo/internal/data"
	"github.com/impactscope-organization/wobsongo/internal/model"
)

// analyzerMaxTokens bounds the response length for scope/decomposition — a
// short structured decision plus a handful of short sub-claim strings, not a
// long-form generation, so a small budget is enough.
const analyzerMaxTokens = 1000

// analyzerHTTPTimeout bounds a single analyzer call — much shorter than
// ExtractionClient's, since the input (one message) and expected output
// (a short JSON decision) are both small.
const analyzerHTTPTimeout = 2 * time.Minute

// analyzerMaxSubClaims caps how many sub-claims Analyze returns, regardless
// of how many the model proposes — bounds worst-case downstream retrieval/
// judge cost per request.
const analyzerMaxSubClaims = 5

// ClaimAnalyzerClient implements data.ClaimAnalyzer against a generic
// OpenAI-compatible text chat-completions API — same shape as ExtractionClient.
type ClaimAnalyzerClient struct {
	baseURL    string
	model      string
	apiKey     string
	httpClient *http.Client
}

// Ensure ClaimAnalyzerClient implements data.ClaimAnalyzer.
var _ data.ClaimAnalyzer = (*ClaimAnalyzerClient)(nil)

// NewClaimAnalyzerClient creates a new ClaimAnalyzerClient targeting the
// given base URL/model. apiKey may be empty — self-hosted servers often need
// no auth.
func NewClaimAnalyzerClient(baseURL, model, apiKey string) *ClaimAnalyzerClient {
	return &ClaimAnalyzerClient{
		baseURL: baseURL,
		model:   model,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: analyzerHTTPTimeout,
		},
	}
}

// claimAnalysisJSON is the wire shape the LLM is instructed to respond with.
type claimAnalysisJSON struct {
	InScope       *bool          `json:"in_scope"`
	RefusalReason string         `json:"refusal_reason"`
	SubClaims     []subClaimJSON `json:"sub_claims"`
	Language      string         `json:"language"`
}

// subClaimJSON is one sub_claims entry's wire shape.
type subClaimJSON struct {
	Text           string `json:"text"`
	HighRisk       bool   `json:"high_risk"`
	HighRiskReason string `json:"high_risk_reason"`
	Topic          string `json:"topic"`
}

// analyzeOnce makes a single analyzer call. Analyze wraps it with a retry
// for sub-claims whose topic is outside the closed list.
func (c *ClaimAnalyzerClient) analyzeOnce(
	ctx context.Context,
	message string,
	topics []data.AnalyzerTopic,
) (*data.ClaimAnalysis, error) {
	payload := extractionCompletionRequest{
		Model: c.model,
		Messages: []extractionChatMessage{
			{Role: "user", Content: buildAnalyzerPrompt(message, topics)},
		},
		MaxTokens: analyzerMaxTokens,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal analyzer request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/v1/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create analyzer request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call analyzer endpoint: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read analyzer response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"analyzer endpoint returned error status: %d. Body: %s",
			resp.StatusCode,
			string(respBytes),
		)
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal analyzer response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("analyzer response contained no choices: %s", respBytes)
	}

	content := stripJSONCodeFence(parsed.Choices[0].Message.Content)
	log.Printf("[ClaimAnalyzerClient] raw analyzer output: %s", content)
	var raw claimAnalysisJSON
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claim analysis JSON: %w: %s", err, content)
	}

	if raw.InScope == nil {
		return nil, fmt.Errorf("analyzer JSON missing in_scope: %s", content)
	}
	if !*raw.InScope && strings.TrimSpace(raw.RefusalReason) == "" {
		raw.RefusalReason = "Désolé, je ne peux pas traiter cette demande."
	}

	rawSubClaims := raw.SubClaims
	if len(rawSubClaims) > analyzerMaxSubClaims {
		rawSubClaims = rawSubClaims[:analyzerMaxSubClaims]
	}
	subClaims := make([]data.SubClaim, len(rawSubClaims))
	for i, sc := range rawSubClaims {
		subClaims[i] = data.SubClaim{
			Text:           sc.Text,
			HighRisk:       sc.HighRisk,
			HighRiskReason: sc.HighRiskReason,
			TopicID:        resolveTopicID(sc.Topic, topics),
		}
	}

	language, err := model.ParseLanguage(raw.Language)
	if err != nil {
		// Defaults to English (ParseLanguage's own zero-value fallback)
		// rather than failing the whole analysis over a language-detection
		// miss — the worst case is an English-language summary/reasoning
		// for a non-English query, not a broken response.
		log.Printf(
			"[ClaimAnalyzerClient] unrecognized language %q, defaulting to English",
			raw.Language,
		)
	}

	return &data.ClaimAnalysis{
		InScope:       *raw.InScope,
		RefusalReason: raw.RefusalReason,
		SubClaims:     subClaims,
		Language:      language,
	}, nil
}

// buildAnalyzerPrompt builds the scope+decomposition instruction.
func buildAnalyzerPrompt(message string, topics []data.AnalyzerTopic) string {
	var b strings.Builder
	b.WriteString("You are the scope gate for a Sexual and Reproductive Health (SRH) ")
	b.WriteString("fact-checking system. Given a raw input message (which may be a casual ")
	b.WriteString("sentence, a claim from a video transcript, or a direct question), decide:\n\n")
	b.WriteString("1. Is this a health-related inquiry? This system only checks health claims ")
	b.WriteString("— reject anything unrelated to health (e.g. general trivia, requests to do ")
	b.WriteString("unrelated tasks, small talk).\n")
	b.WriteString("2. If it is health-related, decompose it into one or more short, independently ")
	b.WriteString(
		"checkable factual claims, rephrased as crisp, self-contained propositions — not ",
	)
	b.WriteString(
		"the original casual/compound phrasing. A simple claim decomposes to one item; a ",
	)
	b.WriteString("compound claim (\"X treats Y and also prevents Z\") decomposes to multiple.\n")
	b.WriteString(
		"3. What language is the input message written in — exactly one of \"en\" or \"fr\".\n",
	)
	b.WriteString(
		"4. For each sub-claim, set high_risk to true ONLY if it promotes or asks about: ",
	)
	b.WriteString(
		"tobacco, alcohol, or a recreational or illicit drug; an unregulated homemade ",
	)
	b.WriteString(
		"mixture or potion to ingest or apply; or self-medication with a prescription-only ",
	)
	b.WriteString(
		"medication (e.g. antibiotics, hormones, strong corticosteroids) including when a ",
	)
	b.WriteString(
		"specific medication is named and is known to require a prescription. ",
	)
	b.WriteString(
		"Do NOT set high_risk for ordinary over-the-counter products used for common, ",
	)
	b.WriteString(
		"non-urgent complaints (e.g. a cream or ointment for itching, paracetamol for mild ",
	)
	b.WriteString(
		"pain), nor when no specific prescription-only medication is named. If the matter ",
	)
	b.WriteString(
		"is routine and not life-threatening, set high_risk to false. This never depends on ",
	)
	b.WriteString(
		"whether the claim turns out to be true or false, only on the subject matter itself. ",
	)
	b.WriteString(
		"When high_risk is true, set high_risk_reason to one short sentence in French saying ",
	)
	b.WriteString("which of the above criteria applies; otherwise use \"\".\n\n")
	if len(topics) > 0 {
		b.WriteString("5. For each sub-claim, choose exactly ONE topic from this closed list. ")
		b.WriteString("Copy the topic name exactly as written below, without translating or ")
		b.WriteString("changing it. Choose based on the claim's meaning. If no topic fits ")
		b.WriteString(
			"perfectly, choose the closest one. Never invent a topic or leave it empty:\n",
		)
		for _, t := range topics {
			b.WriteString("- ")
			b.WriteString(t.Name)
			b.WriteString(": ")
			b.WriteString(t.Description)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString(
		"Respond with ONLY a JSON object (no markdown, no commentary), with this shape:\n",
	)
	b.WriteString(
		`{"in_scope": true/false, "refusal_reason": "...", ` +
			`"sub_claims": [{"text": "...", "high_risk": true/false, ` +
			`"high_risk_reason": "...", "topic": "..."}], "language": "en"}` + "\n\n",
	)
	b.WriteString("refusal_reason is only used when in_scope is false (briefly explain why); ")
	b.WriteString(
		"otherwise use \"\". sub_claims is only used when in_scope is true; ")
	b.WriteString("otherwise use []. topic is only used when a topic list is given above; ")
	b.WriteString("otherwise use \"\".\n")
	b.WriteString(
		"Always write each sub-claim's text and refusal_reason in French, regardless of what ",
	)
	b.WriteString(
		"language you detected the input message to be written in (the \"language\" field is for ",
	)
	b.WriteString(
		"detection only) — the system currently only replies in French, even if you expect the ",
	)
	b.WriteString("knowledge base evidence to be in a different language.\n\n")
	b.WriteString("Message:\n\"\"\"\n")
	b.WriteString(message)
	b.WriteString("\n\"\"\"\n")
	return b.String()
}

// Analyze implements data.ClaimAnalyzer. When topics are given and the model
// answers with a topic outside the list, the analysis is retried once; a
// sub-claim that still has no valid topic keeps TopicID uuid.Nil.
func (c *ClaimAnalyzerClient) Analyze(
	ctx context.Context,
	message string,
	topics []data.AnalyzerTopic,
) (*data.ClaimAnalysis, error) {
	analysis, err := c.analyzeOnce(ctx, message, topics)
	if err != nil {
		return nil, err
	}
	if len(topics) == 0 || !hasUnresolvedTopic(analysis) {
		return analysis, nil
	}

	log.Printf("[ClaimAnalyzerClient] sub-claim topic outside the closed list, retrying once")
	retry, err := c.analyzeOnce(ctx, message, topics)
	if err != nil {
		// Keep the first analysis: its unresolved sub-claims are skipped downstream.
		log.Printf("[ClaimAnalyzerClient] topic retry failed, keeping first analysis: %v", err)
		return analysis, nil
	}
	return retry, nil
}

// hasUnresolvedTopic reports whether an in-scope analysis has a sub-claim
// without a valid topic.
func hasUnresolvedTopic(a *data.ClaimAnalysis) bool {
	if !a.InScope {
		return false
	}
	for _, sc := range a.SubClaims {
		if sc.TopicID == uuid.Nil {
			return true
		}
	}
	return false
}

// resolveTopicID maps the topic name the model returned to a topic of the
// closed list (case-insensitive), or uuid.Nil.
func resolveTopicID(name string, topics []data.AnalyzerTopic) uuid.UUID {
	name = strings.TrimSpace(name)
	for _, t := range topics {
		if strings.EqualFold(t.Name, name) {
			return t.ID
		}
	}
	return uuid.Nil
}
