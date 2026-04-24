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
	agent := New(nil, nil, "", config.LintConfig{})

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

func TestSelectSpecs(t *testing.T) {
	goContent := "# Go style rules"
	tsContent := "# TS style rules"

	agent := &Agent{
		specFiles: []config.LintSpecFile{
			{Path: "codestyle.go.md", Extensions: []string{".go"}},
			{Path: "codestyle.ts.md", Extensions: []string{".ts", ".tsx"}},
		},
		specContents: map[string]string{
			"codestyle.go.md": goContent,
			"codestyle.ts.md": tsContent,
		},
	}

	tests := []struct {
		name     string
		exts     map[string]struct{}
		wantLen  int
		wantSpec string
	}{
		{
			name:     "go only",
			exts:     map[string]struct{}{".go": {}},
			wantLen:  1,
			wantSpec: goContent,
		},
		{
			name:     "ts only",
			exts:     map[string]struct{}{".ts": {}},
			wantLen:  1,
			wantSpec: tsContent,
		},
		{
			name:    "mixed go and tsx",
			exts:    map[string]struct{}{".go": {}, ".tsx": {}},
			wantLen: 2,
		},
		{
			name:    "no match",
			exts:    map[string]struct{}{".py": {}},
			wantLen: 0,
		},
		{
			name:    "empty diff",
			exts:    map[string]struct{}{},
			wantLen: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := agent.selectSpecs(tc.exts)
			if len(got) != tc.wantLen {
				t.Errorf("selectSpecs len = %d, want %d", len(got), tc.wantLen)
			}
			if tc.wantSpec != "" && len(got) > 0 && got[0] != tc.wantSpec {
				t.Errorf("selectSpecs[0] = %q, want %q", got[0], tc.wantSpec)
			}
		})
	}
}
