package coverage

import (
	"context"
	"testing"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/gh"
)

func TestIsExcluded(t *testing.T) {
	excludes := []string{"vendor/", "testdata/*"}

	if !isExcluded("internal/vendor/foo.go", excludes) {
		t.Error("expected internal/vendor/foo.go to be excluded")
	}
	if !isExcluded("testdata/mock.json", excludes) {
		t.Error("expected testdata/mock.json to be excluded")
	}
	if isExcluded("internal/config/config.go", excludes) {
		t.Error("expected internal/config/config.go not to be excluded")
	}
}

func TestInspectPatch(t *testing.T) {
	// Go patch with export and no comments
	patchGoExport := `@@ -1,3 +1,4 @@
 package main
+func ExportedFunc() {}
 `
	exported, commented := inspectPatch("main.go", patchGoExport)
	if !exported {
		t.Error("expected exported function to be detected")
	}
	if commented {
		t.Error("did not expect comments to be detected")
	}

	// Go patch with export and comments
	patchGoComment := `@@ -1,3 +1,5 @@
 package main
+// Comment explaining func
+func ExportedFunc() {}
 `
	exported, commented = inspectPatch("main.go", patchGoComment)
	if !exported {
		t.Error("expected exported function to be detected")
	}
	if !commented {
		t.Error("expected comments to be detected")
	}

	// TS patch with export
	patchTS := `@@ -1,3 +1,4 @@
+export const myVar = 10;
 `
	exported, _ = inspectPatch("main.ts", patchTS)
	if !exported {
		t.Error("expected TS export to be detected")
	}
}

func TestAnalyzeDeterministic(t *testing.T) {
	agent := New(nil, nil, "")
	input := &agents.AnalysisInput{
		PR: &gh.PullRequest{
			Files: []gh.ChangedFile{
				{
					Path:  "internal/config/config.go",
					Patch: "@@ -1 +1,2 @@\n+func RunNewCode() {}\n",
				},
			},
		},
		Config: &config.Config{
			Coverage: config.CoverageConfig{
				Mappings: map[string]string{
					`\.go$`: `_test.go`,
				},
			},
		},
	}

	findings, err := agent.Analyze(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}

	// Should trigger both findings: missing test file and missing documentation/inline comments
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d: %+v", len(findings), findings)
	}

	f := findings[0]
	if f.Location.File != "internal/config/config.go" {
		t.Errorf("unexpected file in finding: %s", f.Location.File)
	}
}
