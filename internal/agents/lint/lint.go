// Package lint implements the lint analysis agent.
// It runs a configurable linter on changed files and optionally enriches results with LLM.
package lint

import (
	_ "embed"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/agents/llmparse"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/llm"
	"github.com/jacopobonta/sugo/internal/promptutil"
)

//go:embed AGENT.md
var defaultPrompt string

// runFn runs an external command and returns its combined output.
type runFn func(ctx context.Context, command string, args []string) ([]byte, error)

// Agent runs a configured linter and converts its output to findings.
type Agent struct {
	run    runFn
	llm    llm.Client // may be nil
	logger *slog.Logger
	prompt string
}

// New creates a lint Agent.
func New(llmClient llm.Client, logger *slog.Logger, promptOverride string) *Agent {
	prompt, err := promptutil.Load(defaultPrompt, promptOverride)
	if err != nil && logger != nil {
		logger.Warn("lint: failed to load prompt override", "error", err)
		prompt = defaultPrompt
	}
	return &Agent{run: lintRun, llm: llmClient, logger: logger, prompt: prompt}
}

// NewWithRun creates a lint Agent with an injectable run function (for testing).
func NewWithRun(fn runFn, llmClient llm.Client, logger *slog.Logger) *Agent {
	return &Agent{run: fn, llm: llmClient, logger: logger, prompt: defaultPrompt}
}

func lintRun(ctx context.Context, command string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	_ = cmd.Run() // linters return non-zero when issues found; capture output regardless
	return out.Bytes(), nil
}

// Name returns the agent identifier.
func (a *Agent) Name() string { return "lint" }

// Available checks that the configured linter binary exists on PATH.
func (a *Agent) Available(cfg *config.Config) bool {
	if cfg.Lint.Command == "" {
		return false
	}
	_, err := exec.LookPath(cfg.Lint.Command)
	return err == nil
}

// Analyze runs the linter and converts output to findings.
func (a *Agent) Analyze(ctx context.Context, input *agents.AnalysisInput) ([]finding.Finding, error) {
	lintCfg := input.Config.Lint
	output, err := a.run(ctx, lintCfg.Command, lintCfg.Args)
	if err != nil {
		return nil, fmt.Errorf("lint run: %w", err)
	}

	parser := selectParser(lintCfg.Command)
	issues, err := parser.Parse(output)
	if err != nil {
		return nil, fmt.Errorf("lint parse: %w", err)
	}

	findings := toFindings(issues, a.Name(), lintCfg.SeverityOverrides)

	if a.llm != nil && len(findings) > 0 {
		enriched, err := a.enrichWithLLM(ctx, findings)
		if err != nil {
			if a.logger != nil {
				a.logger.Warn("lint LLM enrichment failed", "error", err)
			}
			return findings, nil
		}
		return enriched, nil
	}

	return findings, nil
}

func (a *Agent) enrichWithLLM(ctx context.Context, findings []finding.Finding) ([]finding.Finding, error) {
	issuesJSON, _ := json.Marshal(findings)
	userMsg := fmt.Sprintf("Linter findings:\n%s", issuesJSON)

	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: a.prompt},
			{Role: "user", Content: userMsg},
		},
		MaxTokens:   2048,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, err
	}

	return llmparse.ParseFindings(resp.Content, a.Name(), finding.TypeFix)
}
