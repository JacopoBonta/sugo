package rules

import (
	"context"
	"testing"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/gh"
)

func TestCheckBranch(t *testing.T) {
	tests := []struct {
		name     string
		headRef  string
		patterns []string
		wantFind bool
	}{
		{"no patterns", "anything", nil, false},
		{"matches", "feature/PROJ-1-foo", []string{`^feature/[A-Z]+-\d+-.*`}, false},
		{"no match", "bad-branch", []string{`^feature/.*`, `^fix/.*`}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := checkBranch(tc.headRef, tc.patterns)
			if (len(f) > 0) != tc.wantFind {
				t.Errorf("wantFind=%v, got %d findings", tc.wantFind, len(f))
			}
		})
	}
}

func TestCheckCommits(t *testing.T) {
	tests := []struct {
		msg      string
		wantFind bool
	}{
		{"feat: add thing", false},
		{"feat(scope): add thing", false},
		{"fix!: breaking fix", false},
		{"bad commit message", true},
		{"WIP: stuff", true},
	}
	for _, tc := range tests {
		t.Run(tc.msg, func(t *testing.T) {
			f := checkCommits([]gh.Commit{{Message: tc.msg}})
			if (len(f) > 0) != tc.wantFind {
				t.Errorf("msg=%q wantFind=%v got %d findings", tc.msg, tc.wantFind, len(f))
			}
		})
	}
}

func TestCheckLabels(t *testing.T) {
	tests := []struct {
		name     string
		have     []string
		required []string
		wantFind bool
	}{
		{"no required", []string{"x"}, nil, false},
		{"all present", []string{"reviewed", "feature"}, []string{"reviewed"}, false},
		{"missing one", []string{"feature"}, []string{"reviewed"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := checkLabels(tc.have, tc.required)
			if (len(f) > 0) != tc.wantFind {
				t.Errorf("wantFind=%v got %d findings", tc.wantFind, len(f))
			}
		})
	}
}

func TestAgentAnalyze(t *testing.T) {
	agent := New(nil, nil, "")
	input := &agents.AnalysisInput{
		PR: &gh.PullRequest{
			HeadRef: "bad-branch",
			Commits: []gh.Commit{{Message: "bad message"}},
			Labels:  nil,
		},
		Config: &config.Config{
			Rules: config.RulesConfig{
				BranchPatterns:     []string{`^feature/.*`},
				ConventionalCommit: true,
				RequiredLabels:     []string{"reviewed"},
			},
		},
	}
	findings, err := agent.Analyze(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 3 {
		t.Errorf("got %d findings, want 3 (branch + commit + label)", len(findings))
	}
	for _, f := range findings {
		if f.Type != finding.TypeFix {
			t.Errorf("finding type = %s, want fix", f.Type)
		}
		if f.Fix == nil {
			t.Error("Fix must not be nil for rules agent")
		}
	}
}
