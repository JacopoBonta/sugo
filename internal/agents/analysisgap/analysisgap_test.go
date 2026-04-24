package analysisgap

import (
	"context"
	"testing"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/gh"
	"github.com/jacopobonta/sugo/internal/llm"
)

type fakeLLM struct{ resp string }

func (f *fakeLLM) Complete(_ context.Context, _ *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{Content: f.resp}, nil
}

type fakeJira struct{}

func (f *fakeJira) GetIssue(_ context.Context, key string) (*JiraIssue, error) {
	return &JiraIssue{Key: key, Summary: "Implement login", Description: "Add OAuth2 flow"}, nil
}

func TestAnalysisGapAnalyze(t *testing.T) {
	llmResp := `{"findings": [{"agent": "analysisgap", "severity": "high", "location": {"file": "", "line_start": 0, "line_end": 0}, "message": "Token refresh not implemented", "fix": "Add token refresh logic"}]}`

	agent := New(&fakeLLM{resp: llmResp}, &fakeJira{}, nil, "")
	input := &agents.AnalysisInput{
		PR: &gh.PullRequest{
			HeadRef: "feature/PROJ-123-auth",
			Diff:    "diff --git a/auth.go b/auth.go\n+func Login() {}",
		},
		Config: &config.Config{},
	}

	findings, err := agent.Analyze(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Errorf("got %d findings, want 1", len(findings))
	}
	if findings[0].Fix == nil {
		t.Error("analysisgap findings must have Fix")
	}
}

func TestAnalysisGapNoJiraKey(t *testing.T) {
	agent := New(&fakeLLM{}, &fakeJira{}, nil, "")
	input := &agents.AnalysisInput{
		PR:     &gh.PullRequest{HeadRef: "main"},
		Config: &config.Config{},
	}
	findings, err := agent.Analyze(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestJiraKeyExtraction(t *testing.T) {
	tests := []struct {
		branch string
		key    string
	}{
		{"feature/PROJ-123-foo", "PROJ-123"},
		{"fix/AB-99-bar", "AB-99"},
		{"main", ""},
		{"feature/no-ticket", ""},
	}
	for _, tc := range tests {
		got := jiraKeyRE.FindString(tc.branch)
		if got != tc.key {
			t.Errorf("branch %q: got key %q, want %q", tc.branch, got, tc.key)
		}
	}
}
