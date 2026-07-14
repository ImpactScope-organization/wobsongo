package external_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/impactscope-organization/wobsongo/external"
)

func TestModalBGEClient_Embed_Success(t *testing.T) {
	var gotPath string
	var gotBody map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"model":"BAAI/bge-m3","embeddings":[[0.1,0.2],[0.3,0.4]],"modal_execution_time":0.5}`))
	}))
	defer server.Close()

	client := external.NewModalBGEClient(server.URL, "")

	vectors, err := client.Embed(t.Context(), []string{"first", "second"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vectors) != 2 {
		t.Fatalf("expected 2 vectors, got %d", len(vectors))
	}
	if vectors[0][0] != 0.1 || vectors[1][0] != 0.3 {
		t.Errorf("unexpected vectors: %v", vectors)
	}

	if gotPath != "/" {
		t.Errorf("expected POST to root path, got %q", gotPath)
	}
	texts, ok := gotBody["texts"].([]any)
	if !ok || len(texts) != 2 || texts[0] != "first" || texts[1] != "second" {
		t.Errorf("expected texts [\"first\" \"second\"], got %v", gotBody["texts"])
	}
}

func TestModalBGEClient_Embed_ErrorField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"error":"Input 'texts' tidak boleh kosong."}`))
	}))
	defer server.Close()

	client := external.NewModalBGEClient(server.URL, "")
	if _, err := client.Embed(t.Context(), []string{}); err == nil {
		t.Fatal("expected an error when the response carries an error field")
	}
}

func TestModalBGEClient_Embed_MismatchedCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"model":"BAAI/bge-m3","embeddings":[[0.1]]}`))
	}))
	defer server.Close()

	client := external.NewModalBGEClient(server.URL, "")
	if _, err := client.Embed(t.Context(), []string{"first", "second"}); err == nil {
		t.Fatal("expected an error when the response vector count doesn't match the input count")
	}
}

func TestModalBGEClient_Embed_ErrorStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := external.NewModalBGEClient(server.URL, "")
	if _, err := client.Embed(t.Context(), []string{"text"}); err == nil {
		t.Fatal("expected an error for a non-200 response")
	}
}
