package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jacopobonta/sugo/internal/config"
)

func TestMimirComplete(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/llm/mimir_response.json")
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("missing Authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fixture)
	}))
	defer srv.Close()

	client := NewMimirClient(srv.URL+"/v1", "test-key", "test-model")
	resp, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "hello"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content == "" {
		t.Error("empty content")
	}
	if resp.Usage.PromptTokens != 100 {
		t.Errorf("prompt_tokens = %d, want 100", resp.Usage.PromptTokens)
	}
}

func TestNewClient(t *testing.T) {
	cfg := config.LLMConfig{Provider: "mimir", Model: "test-model", BaseURL: "https://example.com"}
	c, err := NewClient(cfg, "key")
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("nil client")
	}
}

func TestNewClientUnknownProvider(t *testing.T) {
	cfg := config.LLMConfig{Provider: "unknown"}
	_, err := NewClient(cfg, "key")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
