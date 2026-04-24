package logic

import (
	"context"
	"testing"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/gh"
	"github.com/jacopobonta/sugo/internal/llm"
)

type fakeLLM struct {
	response string
}

func (f *fakeLLM) Complete(_ context.Context, _ *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return &llm.CompletionResponse{Content: f.response}, nil
}

func TestLogicAgentAnalyze(t *testing.T) {
	fakeResp := `{"findings": [{"agent": "logic", "severity": "high", "location": {"file": "auth.go", "line_start": 10, "line_end": 15}, "message": "potential nil dereference", "fix": null}]}`

	agent := New(&fakeLLM{response: fakeResp}, nil, "")
	input := &agents.AnalysisInput{
		PR: &gh.PullRequest{
			Files: []gh.ChangedFile{
				{Path: "auth.go", Patch: "@@ -1 +1 @@\n+func foo() {}"},
			},
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
	if findings[0].Fix != nil {
		t.Error("logic findings must have nil Fix")
	}
}

func TestLogicAgentAvailable(t *testing.T) {
	if New(nil, nil, "").Available(nil) {
		t.Error("should be unavailable without LLM")
	}
	if !New(&fakeLLM{}, nil, "").Available(nil) {
		t.Error("should be available with LLM")
	}
}

func TestLogicAgentSkipsEmptyPatch(t *testing.T) {
	counter := &callCountLLM{}
	agent := &Agent{llm: counter, prompt: defaultPrompt}
	input := &agents.AnalysisInput{
		PR: &gh.PullRequest{Files: []gh.ChangedFile{{Path: "a.go", Patch: ""}}},
		Config: &config.Config{},
	}
	_, err := agent.Analyze(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if counter.calls != 0 {
		t.Error("LLM should not be called for empty patch")
	}
}

type callCountLLM struct{ calls int }

func (c *callCountLLM) Complete(_ context.Context, _ *llm.CompletionRequest) (*llm.CompletionResponse, error) {
	c.calls++
	return &llm.CompletionResponse{Content: `{"findings":[]}`}, nil
}
