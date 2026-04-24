package rules

import (
	"context"
	"regexp"
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
			f := checkBranch(tc.headRef, tc.patterns, nil)
			if (len(f) > 0) != tc.wantFind {
				t.Errorf("wantFind=%v, got %d findings", tc.wantFind, len(f))
			}
		})
	}
}

func TestCheckBranchTicket(t *testing.T) {
	ticketRE := regexp.MustCompile(`[a-z]+-[0-9]+$`)
	tests := []struct {
		name     string
		headRef  string
		wantFind bool
	}{
		{"has ticket", "feat/add-login-com-123", false},
		{"missing ticket", "feat/add-login", true},
		{"release branch skipped", "release/v1.2.0", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := checkBranch(tc.headRef, nil, ticketRE)
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
			f := checkCommits([]gh.Commit{{Message: tc.msg}}, nil)
			if (len(f) > 0) != tc.wantFind {
				t.Errorf("msg=%q wantFind=%v got %d findings", tc.msg, tc.wantFind, len(f))
			}
		})
	}
}

func TestCheckCommitsTicket(t *testing.T) {
	ticketRE := regexp.MustCompile(`[a-z]+-[0-9]+$`)
	tests := []struct {
		msg      string
		wantFind bool
	}{
		{"feat: add thing com-123", false},
		{"feat: add thing", true},
	}
	for _, tc := range tests {
		t.Run(tc.msg, func(t *testing.T) {
			f := checkCommits([]gh.Commit{{Message: tc.msg}}, ticketRE)
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
	agent := New(nil, nil, "", config.RulesConfig{
		BranchPatterns:     []string{`^feature/.*`},
		ConventionalCommit: true,
		RequiredLabels:     []string{"reviewed"},
	})
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

func TestAgentAnalyzeWithTicketPattern(t *testing.T) {
	agent := New(nil, nil, "", config.RulesConfig{
		ConventionalCommit:    true,
		TrailingTicketPattern: `[a-z]+-[0-9]+$`,
	})
	input := &agents.AnalysisInput{
		PR: &gh.PullRequest{
			HeadRef: "feat/add-login",
			Commits: []gh.Commit{{Message: "feat: add login"}},
		},
		Config: &config.Config{
			Rules: config.RulesConfig{
				ConventionalCommit:    true,
				TrailingTicketPattern: `[a-z]+-[0-9]+$`,
			},
		},
	}
	findings, err := agent.Analyze(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	// expect 2 findings: missing ticket on branch + missing ticket on commit
	if len(findings) != 2 {
		t.Errorf("got %d findings, want 2 (branch ticket + commit ticket)", len(findings))
	}
}
