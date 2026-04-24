package orchestrator

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/gh"
)

// fakeGHClient returns a canned PullRequest.
type fakeGHClient struct{ pr *gh.PullRequest }

func (f *fakeGHClient) FetchPR(_ context.Context, owner, repo string, number int) (*gh.PullRequest, error) {
	return f.pr, nil
}

// fakeAgent returns canned findings.
type fakeAgent struct {
	name      string
	available bool
	findings  []finding.Finding
	err       error
}

func (f *fakeAgent) Name() string                   { return f.name }
func (f *fakeAgent) Available(_ *config.Config) bool { return f.available }
func (f *fakeAgent) Analyze(_ context.Context, _ *agents.AnalysisInput) ([]finding.Finding, error) {
	return f.findings, f.err
}

func testPR() *gh.PullRequest {
	return &gh.PullRequest{Number: 42, Title: "Test PR", Owner: "org", Repo: "repo"}
}

func TestOrchestratorRun(t *testing.T) {
	fix := "do this"
	agentList := []agents.Agent{
		&fakeAgent{name: "a", available: true, findings: []finding.Finding{
			{Agent: "a", Severity: finding.SeverityHigh, Type: finding.TypeFix, Message: "issue", Fix: &fix},
		}},
		&fakeAgent{name: "b", available: true, findings: []finding.Finding{
			{Agent: "b", Severity: finding.SeverityLow, Type: finding.TypeAttentionPoint},
		}},
	}

	orch := New(&fakeGHClient{pr: testPR()}, agentList, &config.Config{}, slog.Default())
	report, err := orch.Run(context.Background(), "org", "repo", 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 2 {
		t.Errorf("got %d findings, want 2", len(report.Findings))
	}
	if len(report.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", report.Warnings)
	}
}

func TestOrchestratorSkipsUnavailableAgents(t *testing.T) {
	agentList := []agents.Agent{
		&fakeAgent{name: "unavail", available: false},
		&fakeAgent{name: "avail", available: true},
	}
	orch := New(&fakeGHClient{pr: testPR()}, agentList, &config.Config{}, slog.Default())
	report, err := orch.Run(context.Background(), "org", "repo", 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Warnings) != 1 {
		t.Errorf("want 1 warning for skipped agent, got %d", len(report.Warnings))
	}
}

func TestOrchestratorAgentFailure(t *testing.T) {
	agentList := []agents.Agent{
		&fakeAgent{name: "failing", available: true, err: errors.New("boom")},
		&fakeAgent{name: "ok", available: true, findings: []finding.Finding{
			{Agent: "ok", Severity: finding.SeverityLow},
		}},
	}
	orch := New(&fakeGHClient{pr: testPR()}, agentList, &config.Config{}, slog.Default())
	report, err := orch.Run(context.Background(), "org", "repo", 42)
	if err != nil {
		t.Fatal(err)
	}
	// Failing agent should produce a warning, not abort the whole run
	if len(report.Warnings) == 0 {
		t.Error("expected warning for failed agent")
	}
	// Other agent's findings should still be present
	if len(report.Findings) == 0 {
		t.Error("expected findings from non-failing agent")
	}
}

func TestOrchestratorDeduplicates(t *testing.T) {
	loc := finding.Location{File: "a.go", LineStart: 1, LineEnd: 5}
	agentList := []agents.Agent{
		&fakeAgent{name: "dup", available: true, findings: []finding.Finding{
			{Agent: "dup", Severity: finding.SeverityLow, Location: loc},
			{Agent: "dup", Severity: finding.SeverityHigh, Location: loc},
		}},
	}
	orch := New(&fakeGHClient{pr: testPR()}, agentList, &config.Config{}, slog.Default())
	report, err := orch.Run(context.Background(), "org", "repo", 42)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 1 {
		t.Errorf("expected 1 deduplicated finding, got %d", len(report.Findings))
	}
	if report.Findings[0].Severity != finding.SeverityHigh {
		t.Errorf("kept wrong severity: %s", report.Findings[0].Severity)
	}
}
