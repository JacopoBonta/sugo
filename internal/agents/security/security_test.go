package security

import (
	"context"
	"regexp"
	"testing"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/gh"
)

func TestScanPatchForSecrets(t *testing.T) {
	patch := `@@ -1,4 +1,6 @@
 func main() {
+	// Slack API
+	token := "slack-123456789012-abcdefghijklmnopqrstuvwx"
+	awsKey := "MOCKIOSFODNN7EXAMPLE"
-	oldCode()
+	newCode()
 }`

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`slack-[0-9]{12}-[a-zA-Z0-9]{24}`),
		regexp.MustCompile(`MOCK[A-Z0-9]{16}`),
	}

	findings := scanPatchForSecrets(patch, patterns)
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}

	if findings[0].Line != 3 {
		t.Errorf("expected finding 0 at line 3, got %d", findings[0].Line)
	}
	if findings[1].Line != 4 {
		t.Errorf("expected finding 1 at line 4, got %d", findings[1].Line)
	}
}

func TestParseGosec(t *testing.T) {
	output := []byte(`{
		"Issues": [
			{
				"severity": "HIGH",
				"confidence": "HIGH",
				"rule_id": "G104",
				"details": "Errors unhandled",
				"file": "main.go",
				"line": 12
			}
		]
	}`)

	findings, err := parseGosec(output)
	if err != nil {
		t.Fatal(err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.Severity != finding.SeverityHigh {
		t.Errorf("expected High severity, got %s", f.Severity)
	}
	if f.Location.LineStart != 12 {
		t.Errorf("expected line 12, got %d", f.Location.LineStart)
	}
}

func TestParseSemgrep(t *testing.T) {
	output := []byte(`{
		"results": [
			{
				"path": "main.js",
				"start": { "line": 42 },
				"extra": {
					"message": "insecure eval",
					"severity": "ERROR",
					"metadata": { "rule_id": "rules.eval" }
				}
			}
		]
	}`)

	findings, err := parseSemgrep(output)
	if err != nil {
		t.Fatal(err)
	}

	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}

	f := findings[0]
	if f.Severity != finding.SeverityHigh {
		t.Errorf("expected High severity, got %s", f.Severity)
	}
	if f.Location.LineStart != 42 {
		t.Errorf("expected line 42, got %d", f.Location.LineStart)
	}
}

func TestParseGenericSecurity(t *testing.T) {
	output := []byte(`main.go:15: unhandled error
[another_file.go:20] security breach
`)

	findings, err := parseGenericSecurity(output)
	if err != nil {
		t.Fatal(err)
	}

	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}

	if findings[0].Location.File != "main.go" || findings[0].Location.LineStart != 15 {
		t.Errorf("unexpected finding 0: %+v", findings[0])
	}
	if findings[1].Location.File != "another_file.go" || findings[1].Location.LineStart != 20 {
		t.Errorf("unexpected finding 1: %+v", findings[1])
	}
}

func TestAgentAnalyze(t *testing.T) {
	mockRun := func(ctx context.Context, cmd string, args []string) ([]byte, error) {
		return []byte(`main.go:15: gosec warning`), nil
	}

	agent := NewWithRun(mockRun, nil, nil)
	input := &agents.AnalysisInput{
		PR: &gh.PullRequest{
			Files: []gh.ChangedFile{
				{
					Path:  "main.go",
					Patch: "@@ -1 +1 @@\n+passwd = \"secretpassword123\"\n",
				},
			},
		},
		Config: &config.Config{
			Security: config.SecurityConfig{
				Command: "generic-scanner",
				Args:    []string{"run"},
			},
		},
	}

	findings, err := agent.Analyze(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}

	// Should contain both the CLI result and the Regex result!
	if len(findings) != 2 {
		t.Errorf("expected 2 findings, got %d: %+v", len(findings), findings)
	}
}
