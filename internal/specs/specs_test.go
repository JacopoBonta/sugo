package specs

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "spec-*.md")
	if err != nil {
		t.Fatal(err)
	}
	content := "  hello world  "
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()

	got, err := Load(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello world" {
		t.Errorf("Load = %q, want %q", got, "hello world")
	}
}

func TestLoadMissing(t *testing.T) {
	_, err := Load("/nonexistent/path/spec.md")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadAll(t *testing.T) {
	dir := t.TempDir()
	paths := make([]string, 3)
	for i := range paths {
		f, _ := os.CreateTemp(dir, "spec-*.md")
		f.WriteString("content")
		f.Close()
		paths[i] = f.Name()
	}

	contents, err := LoadAll(paths)
	if err != nil {
		t.Fatal(err)
	}
	if len(contents) != 3 {
		t.Errorf("LoadAll len = %d, want 3", len(contents))
	}
}

func TestExtensionsFromDiff(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want map[string]struct{}
	}{
		{
			name: "empty",
			diff: "",
			want: map[string]struct{}{},
		},
		{
			name: "single go file",
			diff: "diff --git a/foo/bar.go b/foo/bar.go\n--- a/foo/bar.go\n+++ b/foo/bar.go\n",
			want: map[string]struct{}{".go": {}},
		},
		{
			name: "mixed go and ts",
			diff: "diff --git a/main.go b/main.go\ndiff --git a/ui/app.ts b/ui/app.ts\n",
			want: map[string]struct{}{".go": {}, ".ts": {}},
		},
		{
			name: "tsx and ts",
			diff: "diff --git a/ui/App.tsx b/ui/App.tsx\ndiff --git a/ui/types.ts b/ui/types.ts\n",
			want: map[string]struct{}{".tsx": {}, ".ts": {}},
		},
		{
			name: "no extension",
			diff: "diff --git a/Makefile b/Makefile\n",
			want: map[string]struct{}{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtensionsFromDiff(tc.diff)
			if len(got) != len(tc.want) {
				t.Errorf("ExtensionsFromDiff len = %d, want %d (got %v, want %v)", len(got), len(tc.want), got, tc.want)
				return
			}
			for ext := range tc.want {
				if _, ok := got[ext]; !ok {
					t.Errorf("missing extension %q in result %v", ext, got)
				}
			}
		})
	}
}
