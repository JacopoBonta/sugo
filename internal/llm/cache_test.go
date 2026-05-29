package llm

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type mockClient struct {
	mu          sync.Mutex
	callsCount  int
	lastRequest *CompletionRequest
	response    *CompletionResponse
	err         error
}

func (m *mockClient) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callsCount++
	m.lastRequest = req
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestCachingClient(t *testing.T) {
	// Temporarily switch working directory so updateGitignore does not run on real root
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	// Create fake gitignore
	if err := os.WriteFile(".gitignore", []byte("sugo\n"), 0644); err != nil {
		t.Fatal(err)
	}

	mock := &mockClient{
		response: &CompletionResponse{
			Content: "hello cache",
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
			},
		},
	}

	cc, err := NewCachingClient(mock, filepath.Join(tempDir, "cache"))
	if err != nil {
		t.Fatalf("failed to create CachingClient: %v", err)
	}

	req := &CompletionRequest{
		Model:       "test-model",
		Messages:    []Message{{Role: "user", Content: "hi"}},
		Temperature: Float64(0.5),
		Seed:        Int(123),
		JSONMode:    true,
	}

	// Call 1: cache miss
	resp1, err := cc.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp1.Content != "hello cache" {
		t.Errorf("expected content 'hello cache', got %q", resp1.Content)
	}
	if mock.callsCount != 1 {
		t.Errorf("expected mock calls to be 1, got %d", mock.callsCount)
	}

	// Call 2: cache hit
	resp2, err := cc.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}
	if resp2.Content != "hello cache" {
		t.Errorf("expected content 'hello cache', got %q", resp2.Content)
	}
	if mock.callsCount != 1 {
		t.Errorf("expected mock calls to remain 1, got %d", mock.callsCount)
	}

	// Read cache file directly to verify it was written
	cacheFilePath := filepath.Join(tempDir, "cache", "llm.json")
	if _, err := os.Stat(cacheFilePath); err != nil {
		t.Errorf("expected cache file to be created: %v", err)
	}

	// Instantiate new client with same cacheDir, verify it loads cache
	mock2 := &mockClient{
		response: &CompletionResponse{
			Content: "should not be called",
		},
	}
	cc2, err := NewCachingClient(mock2, filepath.Join(tempDir, "cache"))
	if err != nil {
		t.Fatalf("failed to create second CachingClient: %v", err)
	}

	resp3, err := cc2.Complete(context.Background(), req)
	if err != nil {
		t.Fatalf("Complete failed on loaded cache: %v", err)
	}
	if resp3.Content != "hello cache" {
		t.Errorf("expected loaded cache content 'hello cache', got %q", resp3.Content)
	}
	if mock2.callsCount != 0 {
		t.Errorf("expected mock2 calls to be 0, got %d", mock2.callsCount)
	}
}

func TestUpdateGitignore(t *testing.T) {
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWd)
	})

	// Case 1: .gitignore doesn't exist
	if err := updateGitignore(); err != nil {
		t.Errorf("unexpected error on missing .gitignore: %v", err)
	}

	// Case 2: .gitignore exists but missing .sugo/
	ignorePath := filepath.Join(tempDir, ".gitignore")
	if err := os.WriteFile(ignorePath, []byte("node_modules\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := updateGitignore(); err != nil {
		t.Errorf("unexpected error on updateGitignore: %v", err)
	}

	data, err := os.ReadFile(ignorePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), ".sugo/") {
		t.Errorf("expected .gitignore to contain .sugo/, got %q", string(data))
	}

	// Case 3: .gitignore already has .sugo/
	originalLen := len(data)
	if err := updateGitignore(); err != nil {
		t.Errorf("unexpected error on second updateGitignore: %v", err)
	}
	data2, err := os.ReadFile(ignorePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data2) != originalLen {
		t.Errorf("expected no modification to .gitignore if already has ignore rule, got %q", string(data2))
	}
}
