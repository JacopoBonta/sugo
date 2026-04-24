package focus

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

func TestFocusAgentAnalyze(t *testing.T) {
	resp := `{"findings": [{"agent": "focus", "severity": "high", "location": {"file": "main.go", "line_start": 0, "line_end": 0}, "message": "Core authentication change — review carefully", "fix": null}]}`

	agent := New(&fakeLLM{resp: resp}, nil, "")
	input := &agents.AnalysisInput{
		PR:     &gh.PullRequest{Title: "Add auth", Diff: "diff --git a/main.go b/main.go\n+func foo() {}"},
		Config: &config.Config{},
	}

	findings, err := agent.Analyze(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 {
		t.Errorf("got %d findings, want 1", len(findings))
	}
}

func TestFocusAgentEmptyDiff(t *testing.T) {
	agent := New(&fakeLLM{}, nil, "")
	findings, err := agent.Analyze(context.Background(), &agents.AnalysisInput{
		PR: &gh.PullRequest{}, Config: &config.Config{},
	})
	if err != nil || findings != nil {
		t.Errorf("empty diff should return nil, nil; got %v, %v", findings, err)
	}
}
