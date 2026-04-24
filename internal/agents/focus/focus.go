// Package focus implements the focus analysis agent.
// It summarizes the full PR diff and ranks files by review priority.
package focus

import (
	_ "embed"
	"context"
	"fmt"
	"log/slog"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/agents/llmparse"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/llm"
	"github.com/jacopobonta/sugo/internal/promptutil"
)

//go:embed AGENT.md
var defaultPrompt string

// Agent summarizes the PR diff and highlights review priorities.
type Agent struct {
	llm    llm.Client
	logger *slog.Logger
	prompt string
}

// New creates a focus Agent.
func New(llmClient llm.Client, logger *slog.Logger, promptOverride string) *Agent {
	prompt, err := promptutil.Load(defaultPrompt, promptOverride)
	if err != nil && logger != nil {
		logger.Warn("focus: failed to load prompt override", "error", err)
		prompt = defaultPrompt
	}
	return &Agent{llm: llmClient, logger: logger, prompt: prompt}
}

// Name returns the agent identifier.
func (a *Agent) Name() string { return "focus" }

// Available checks that an LLM client is configured.
func (a *Agent) Available(_ *config.Config) bool { return a.llm != nil }

// Analyze sends the full PR diff to the LLM for a focus summary.
func (a *Agent) Analyze(ctx context.Context, input *agents.AnalysisInput) ([]finding.Finding, error) {
	if input.PR.Diff == "" {
		return nil, nil
	}

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: a.prompt},
			{Role: "user", Content: fmt.Sprintf("PR title: %s\n\n%s", input.PR.Title, input.PR.Diff)},
		},
		MaxTokens:   2048,
		Temperature: 0.3,
	})
	if err != nil {
		return nil, fmt.Errorf("focus LLM call: %w", err)
	}

	return llmparse.ParseFindings(resp.Content, a.Name(), finding.TypeAttentionPoint)
}
