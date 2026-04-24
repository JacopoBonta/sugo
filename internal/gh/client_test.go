package gh

import (
	"context"
	"os"
	"testing"
)

func TestFetchPR(t *testing.T) {
	viewData, err := os.ReadFile("../../testdata/gh/pr_view.json")
	if err != nil {
		t.Fatal(err)
	}
	diffData, err := os.ReadFile("../../testdata/gh/pr_diff.txt")
	if err != nil {
		t.Fatal(err)
	}

	callCount := 0
	fakeRun := func(ctx context.Context, args ...string) ([]byte, error) {
		callCount++
		if callCount == 1 {
			return viewData, nil
		}
		return diffData, nil
	}

	client := NewCLIClientWithRun(fakeRun)
	pr, err := client.FetchPR(context.Background(), "myorg", "myrepo", 42)
	if err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Errorf("callCount = %d, want 2", callCount)
	}
	if pr.Number != 42 {
		t.Errorf("Number = %d, want 42", pr.Number)
	}
	if pr.Diff == "" {
		t.Error("Diff should be populated")
	}
}
