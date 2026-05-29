package llm

import (
	"context"
	"encoding/json"
	"io"
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

	client := NewMimirClient(srv.URL+"/v1", "test-key", "test-model", nil, nil, nil)
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

func TestMimirOverridesAndFallback(t *testing.T) {
	var requestBodies [][]byte
	var requestCount int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		var bodyBytes []byte
		if r.Body != nil {
			var err error
			bodyBytes, err = io.ReadAll(r.Body)
			if err != nil {
				t.Fatal(err)
			}
		}
		requestBodies = append(requestBodies, bodyBytes)

		if requestCount == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Bad Request"))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"choices": [{"message": {"content": "fallback success"}}],
			"usage": {"prompt_tokens": 10, "completion_tokens": 5}
		}`))
	}))
	defer srv.Close()

	client := NewMimirClient(
		srv.URL+"/v1",
		"test-key",
		"test-model",
		Float64(0.7),
		Int(777),
		boolPtr(true),
	)

	resp, err := client.Complete(context.Background(), &CompletionRequest{
		Messages: []Message{
			{Role: "user", Content: "test"},
		},
		Temperature: Float64(0.9),
		Seed:        Int(888),
		JSONMode:    true,
	})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.Content != "fallback success" {
		t.Errorf("expected content 'fallback success', got %q", resp.Content)
	}

	if requestCount != 2 {
		t.Errorf("expected 2 requests, got %d", requestCount)
	}

	if len(requestBodies) != 2 {
		t.Fatalf("expected 2 request bodies, got %d", len(requestBodies))
	}

	var req1 openAIRequest
	if err := json.Unmarshal(requestBodies[0], &req1); err != nil {
		t.Fatalf("failed to unmarshal first request: %v", err)
	}
	if req1.Temperature != 0.9 {
		t.Errorf("expected temperature 0.9, got %v", req1.Temperature)
	}
	if req1.Seed == nil || *req1.Seed != 888 {
		t.Errorf("expected seed 888, got %v", req1.Seed)
	}
	if req1.ResponseFormat == nil || req1.ResponseFormat.Type != "json_object" {
		t.Errorf("expected response format json_object, got %v", req1.ResponseFormat)
	}

	var req2 openAIRequest
	if err := json.Unmarshal(requestBodies[1], &req2); err != nil {
		t.Fatalf("failed to unmarshal second request: %v", err)
	}
	if req2.Temperature != 0.9 {
		t.Errorf("expected temperature 0.9, got %v", req2.Temperature)
	}
	if req2.Seed != nil {
		t.Errorf("expected seed to be nil in fallback request, got %v", req2.Seed)
	}
	if req2.ResponseFormat != nil {
		t.Errorf("expected response format to be nil in fallback request, got %v", req2.ResponseFormat)
	}
}

func boolPtr(b bool) *bool {
	return &b
}
