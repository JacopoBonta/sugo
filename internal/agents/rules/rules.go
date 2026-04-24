// Package rules implements the rules analysis agent.
// It checks branch naming, conventional commits, and required PR labels.
package rules

import (
	_ "embed"
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/agents/llmparse"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/llm"
	"github.com/jacopobonta/sugo/internal/promptutil"
)

//go:embed AGENT.md
var defaultPrompt string

// Agent performs deterministic rules checks with optional LLM enrichment.
type Agent struct {
	llm    llm.Client // may be nil
	logger *slog.Logger
	prompt string
}

// New creates a rules Agent. llmClient may be nil to run deterministic-only.
func New(llmClient llm.Client, logger *slog.Logger, promptOverride string) *Agent {
	prompt, err := promptutil.Load(defaultPrompt, promptOverride)
	if err != nil && logger != nil {
		logger.Warn("rules: failed to load prompt override", "error", err)
		prompt = defaultPrompt
	}
	return &Agent{llm: llmClient, logger: logger, prompt: prompt}
}

// Name returns the agent identifier.
func (a *Agent) Name() string { return "rules" }

// Available always returns true; the rules agent has no external dependencies.
func (a *Agent) Available(_ *config.Config) bool { return true }

// Analyze runs branch, commit, and label checks then optionally enriches with LLM.
func (a *Agent) Analyze(ctx context.Context, input *agents.AnalysisInput) ([]finding.Finding, error) {
	pr := input.PR
	cfg := input.Config.Rules

	var findings []finding.Finding
	findings = append(findings, checkBranch(pr.HeadRef, cfg.BranchPatterns)...)
	if cfg.ConventionalCommit {
		findings = append(findings, checkCommits(pr.Commits)...)
	}
	findings = append(findings, checkLabels(pr.Labels, cfg.RequiredLabels)...)

	if a.llm != nil && len(findings) > 0 {
		enriched, err := a.enrichWithLLM(ctx, findings)
		if err != nil {
			if a.logger != nil {
				a.logger.Warn("rules LLM enrichment failed", "error", err)
			}
			return findings, nil
		}
		return enriched, nil
	}

	return findings, nil
}

func (a *Agent) enrichWithLLM(ctx context.Context, findings []finding.Finding) ([]finding.Finding, error) {
	var sb strings.Builder
	sb.WriteString("Violations found:\n")
	for _, f := range findings {
		fmt.Fprintf(&sb, "- %s: %s\n", f.Agent, f.Message)
	}
	userMsg := sb.String()

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: a.prompt},
			{Role: "user", Content: userMsg},
		},
		MaxTokens:   1024,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, err
	}

	return llmparse.ParseFindings(resp.Content, a.Name(), finding.TypeFix)
}
