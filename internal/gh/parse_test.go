package gh

import (
	"os"
	"testing"
)

func TestParsePRView(t *testing.T) {
	data, err := os.ReadFile("../../testdata/gh/pr_view.json")
	if err != nil {
		t.Fatal(err)
	}
	pr, err := parsePRView(data, "myorg", "myrepo")
	if err != nil {
		t.Fatal(err)
	}
	if pr.Number != 42 {
		t.Errorf("Number = %d, want 42", pr.Number)
	}
	if pr.Title != "Add user authentication" {
		t.Errorf("Title = %q", pr.Title)
	}
	if pr.HeadRef != "feature/PROJ-123-auth" {
		t.Errorf("HeadRef = %q", pr.HeadRef)
	}
	if len(pr.Labels) != 2 {
		t.Errorf("Labels len = %d, want 2", len(pr.Labels))
	}
	if len(pr.Commits) != 2 {
		t.Errorf("Commits len = %d, want 2", len(pr.Commits))
	}
	if len(pr.Files) != 2 {
		t.Errorf("Files len = %d, want 2", len(pr.Files))
	}
}

func TestAttachPatches(t *testing.T) {
	diff, err := os.ReadFile("../../testdata/gh/pr_diff.txt")
	if err != nil {
		t.Fatal(err)
	}
	pr := &PullRequest{
		Files: []ChangedFile{
			{Path: "internal/auth/auth.go"},
			{Path: "internal/auth/auth_test.go"},
		},
	}
	attachPatches(pr, string(diff))

	if pr.Diff == "" {
		t.Error("Diff should be populated")
	}
	if pr.Files[0].Patch == "" {
		t.Error("auth.go patch should be populated")
	}
	if pr.Files[1].Patch == "" {
		t.Error("auth_test.go patch should be populated")
	}
}

func TestSplitDiffByFile(t *testing.T) {
	diff := `diff --git a/foo.go b/foo.go
index 000..111 100644
--- a/foo.go
+++ b/foo.go
@@ -1 +1,2 @@
 package main
+// added
diff --git a/bar.go b/bar.go
index 000..222 100644
--- a/bar.go
+++ b/bar.go
@@ -1 +1,2 @@
 package main
+// added too
`
	patches := splitDiffByFile(diff)
	if len(patches) != 2 {
		t.Errorf("got %d patches, want 2", len(patches))
	}
	if _, ok := patches["foo.go"]; !ok {
		t.Error("missing foo.go patch")
	}
	if _, ok := patches["bar.go"]; !ok {
		t.Error("missing bar.go patch")
	}
}
