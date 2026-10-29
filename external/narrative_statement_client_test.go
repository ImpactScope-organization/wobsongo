package external_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/impactscope-organization/wobsongo/external"
)

func TestNarrativeStatementClient_Generate_Success(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody struct {
		Model     string `json:"model"`
		MaxTokens int    `json:"max_tokens"`
		Messages  []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Surrounding whitespace and quotes must be trimmed from the statement.
		_, _ = w.Write(chatResponseBody(t, "  \"Les citrons soignent les infections.\"  "))
	}))
	defer server.Close()

	client := external.NewNarrativeStatementClient(server.URL, "test-model", "test-api-key")
	statement, err := client.Generate(t.Context(), "lemon water cures infections")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if statement != "Les citrons soignent les infections." {
		t.Errorf("expected a trimmed statement, got %q", statement)
	}
	if gotPath != "/v1/chat/completions" {
		t.Errorf("expected path /v1/chat/completions, got %q", gotPath)
	}
	if gotAuth != "Bearer test-api-key" {
		t.Errorf("expected bearer auth header, got %q", gotAuth)
	}
	if gotBody.Model != "test-model" {
		t.Errorf("expected model %q, got %q", "test-model", gotBody.Model)
	}
	if gotBody.MaxTokens <= 0 {
		t.Errorf("expected a positive max_tokens, got %d", gotBody.MaxTokens)
	}
	if len(gotBody.Messages) != 1 || gotBody.Messages[0].Role != "user" ||
		!strings.Contains(gotBody.Messages[0].Content, "lemon water cures infections") {
		t.Errorf("expected one user message carrying the claim, got %+v", gotBody.Messages)
	}
}

func TestNarrativeStatementClient_Generate_NoAPIKey_OmitsAuthHeader(t *testing.T) {
	var hasAuth bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, hasAuth = r.Header["Authorization"]
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(chatResponseBody(t, "Une phrase."))
	}))
	defer server.Close()

	client := external.NewNarrativeStatementClient(server.URL, "test-model", "")
	if _, err := client.Generate(t.Context(), "some claim"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if hasAuth {
		t.Error("expected no Authorization header when the API key is empty")
	}
}

func TestNarrativeStatementClient_Generate_Errors(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"NonOKStatus", http.StatusInternalServerError, `{"error":"boom"}`},
		{"InvalidJSON", http.StatusOK, `not json`},
		{"NoChoices", http.StatusOK, `{"choices":[]}`},
		{"EmptyContent", http.StatusOK, string(chatResponseBody(t, "   "))},
		{"OnlyQuotes", http.StatusOK, string(chatResponseBody(t, ` "" `))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.body))
				}),
			)
			defer server.Close()

			client := external.NewNarrativeStatementClient(server.URL, "test-model", "key")
			statement, err := client.Generate(t.Context(), "some claim")
			if err == nil {
				t.Fatalf("expected an error, got statement %q", statement)
			}
			if statement != "" {
				t.Errorf("expected an empty statement on error, got %q", statement)
			}
		})
	}
}

func TestNarrativeStatementClient_Generate_CancelledContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(chatResponseBody(t, "Une phrase."))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	client := external.NewNarrativeStatementClient(server.URL, "test-model", "key")
	_, err := client.Generate(ctx, "some claim")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected %v, got %v", context.Canceled, err)
	}
}
