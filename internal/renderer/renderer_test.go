package renderer

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/gh"
	"github.com/jacopobonta/sugo/internal/orchestrator"
)

func sampleReport() *orchestrator.Report {
	fix := "rename it"
	return &orchestrator.Report{
		PR: gh.PullRequest{
			Number:  42,
			Title:   "Add auth",
			Owner:   "myorg",
			Repo:    "myrepo",
			HeadRef: "feature/PROJ-123-auth",
			BaseRef: "main",
		},
		Findings: []finding.Finding{
			{
				Agent:    "rules",
				Severity: finding.SeverityHigh,
				Type:     finding.TypeFix,
				Location: finding.Location{File: "auth.go", LineStart: 12, LineEnd: 24},
				Message:  "branch name invalid",
				Fix:      &fix,
			},
			{
				Agent:    "logic",
				Severity: finding.SeverityMedium,
				Type:     finding.TypeAttentionPoint,
				Location: finding.Location{File: "auth.go", LineStart: 45, LineEnd: 45},
				Message:  "nil dereference possible",
			},
		},
		Warnings: []string{"agent analysisgap skipped: SUGO_JIRA_TOKEN not set"},
		Duration: 2 * time.Second,
	}
}

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleReport(), "json"); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := got["findings"]; !ok {
		t.Error("json missing 'findings' key")
	}
}

func TestRenderTerminal(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleReport(), "terminal"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "PR #42") {
		t.Error("terminal output missing PR number")
	}
	if !strings.Contains(out, "HIGH") {
		t.Error("terminal output missing HIGH severity")
	}
	if !strings.Contains(out, "rename it") {
		t.Error("terminal output missing fix text")
	}
	if !strings.Contains(out, "Warnings") {
		t.Error("terminal output missing warnings section")
	}
}

func TestRenderMarkdown(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleReport(), "markdown"); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "# PR #42") {
		t.Error("markdown missing heading")
	}
	if !strings.Contains(out, "**Fix**") {
		t.Error("markdown missing fix section")
	}
}

func TestRenderUnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(&buf, sampleReport(), "xml"); err == nil {
		t.Error("expected error for unknown format")
	}
}
