// Package agents defines the Agent interface and shared input/output types.
package agents

import (
	"context"

	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/gh"
)

// AnalysisInput bundles all data an agent might need. Agents read what they require.
type AnalysisInput struct {
	PR     *gh.PullRequest
	Config *config.Config
}

// Agent is the interface every analysis agent must implement.
type Agent interface {
	// Name returns a stable lowercase identifier for this agent.
	Name() string

	// Available reports whether this agent can run in the current environment
	// (e.g., required env vars are set, binaries are on PATH).
	Available(cfg *config.Config) bool

	// Analyze runs the agent's analysis and returns findings.
	Analyze(ctx context.Context, input *AnalysisInput) ([]finding.Finding, error)
}
