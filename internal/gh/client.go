package gh

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
)

// Client fetches GitHub PR data.
type Client interface {
	FetchPR(ctx context.Context, owner, repo string, number int) (*PullRequest, error)
}

// runFn is the signature for running external commands, injectable for testing.
type runFn func(ctx context.Context, args ...string) ([]byte, error)

// CLIClient fetches PR data by shelling out to the gh CLI.
type CLIClient struct {
	run runFn
}

// NewCLIClient returns a CLIClient that calls the real gh binary.
func NewCLIClient() *CLIClient {
	return &CLIClient{run: ghRun}
}

// NewCLIClientWithRun returns a CLIClient with an injectable run function (for testing).
func NewCLIClientWithRun(fn runFn) *CLIClient {
	return &CLIClient{run: fn}
}

// ghRun calls the gh binary with the given arguments.
// It uses exec.CommandContext which passes args as a slice (no shell interpolation).
func ghRun(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh: %w", err)
	}
	return out, nil
}

// FetchPR fetches PR metadata and diff for the given PR reference.
func (c *CLIClient) FetchPR(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	ref := fmt.Sprintf("%s/%s", owner, repo)
	numStr := strconv.Itoa(number)

	viewJSON, err := c.run(ctx,
		"gh", "pr", "view", numStr,
		"-R", ref,
		"--json", "number,title,body,baseRefName,headRefName,additions,deletions,labels,commits,files",
	)
	if err != nil {
		return nil, fmt.Errorf("gh pr view: %w", err)
	}

	pr, err := parsePRView(viewJSON, owner, repo)
	if err != nil {
		return nil, err
	}

	diff, err := c.run(ctx, "gh", "pr", "diff", numStr, "-R", ref)
	if err != nil {
		return nil, fmt.Errorf("gh pr diff: %w", err)
	}

	attachPatches(pr, string(diff))
	return pr, nil
}
