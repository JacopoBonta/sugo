package lint

import (
	"context"
	"os"
	"testing"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/gh"
)

func TestAgentAnalyze(t *testing.T) {
	fixture, err := os.ReadFile("../../../testdata/agents/lint/golangci_output.json")
	if err != nil {
		t.Fatal(err)
	}

	fakeRun := func(ctx context.Context, command string, args []string) ([]byte, error) {
		return fixture, nil
	}

	agent := NewWithRun(fakeRun, nil, nil)
	input := &agents.AnalysisInput{
		PR: &gh.PullRequest{},
		Config: &config.Config{
			Lint: config.LintConfig{
				Command: "golangci-lint",
				Args:    []string{"run", "--out-format", "json"},
			},
		},
	}

	findings, err := agent.Analyze(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 2 {
		t.Errorf("got %d findings, want 2", len(findings))
	}
}

func TestAgentAvailable(t *testing.T) {
	agent := New(nil, nil, "")

	cfg := &config.Config{Lint: config.LintConfig{Command: ""}}
	if agent.Available(cfg) {
		t.Error("empty command should be unavailable")
	}

	cfg.Lint.Command = "nonexistent-binary-xyz"
	if agent.Available(cfg) {
		t.Error("nonexistent binary should be unavailable")
	}

	cfg.Lint.Command = "go"
	if !agent.Available(cfg) {
		t.Error("go binary should be available")
	}
}
