// Package logic implements the logic analysis agent.
// It analyzes each changed file for logic issues using an LLM.
package logic

import (
	_ "embed"
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/agents/llmparse"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/llm"
	"github.com/jacopobonta/sugo/internal/promptutil"
)

//go:embed AGENT.md
var defaultPrompt string

// Agent analyzes changed files for logic issues.
type Agent struct {
	llm    llm.Client
	logger *slog.Logger
	prompt string
}

// New creates a logic Agent.
func New(llmClient llm.Client, logger *slog.Logger, promptOverride string) *Agent {
	prompt, err := promptutil.Load(defaultPrompt, promptOverride)
	if err != nil && logger != nil {
		logger.Warn("logic: failed to load prompt override", "error", err)
		prompt = defaultPrompt
	}
	return &Agent{llm: llmClient, logger: logger, prompt: prompt}
}

// Name returns the agent identifier.
func (a *Agent) Name() string { return "logic" }

// Available checks that an LLM client is configured.
func (a *Agent) Available(_ *config.Config) bool { return a.llm != nil }

// Analyze sends each changed file's patch to the LLM and collects logic findings.
func (a *Agent) Analyze(ctx context.Context, input *agents.AnalysisInput) ([]finding.Finding, error) {
	files := input.PR.Files

	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)

	var (
		mu       sync.Mutex
		all      []finding.Finding
		firstErr error
	)

	var wg sync.WaitGroup
	for _, f := range files {
		if f.Patch == "" {
			continue
		}
		wg.Add(1)
		go func(path, patch string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			findings, err := a.analyzeFile(ctx, path, patch)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			all = append(all, findings...)
		}(f.Path, f.Patch)
	}
	wg.Wait()

	if firstErr != nil && len(all) == 0 {
		return nil, firstErr
	}
	return all, nil
}

func (a *Agent) analyzeFile(ctx context.Context, path, patch string) ([]finding.Finding, error) {
	userMsg := fmt.Sprintf("File: %s\n\n%s", path, patch)
	resp, err := a.llm.Complete(ctx, &llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "system", Content: a.prompt},
			{Role: "user", Content: userMsg},
		},
		MaxTokens:   1024,
		Temperature: 0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("logic LLM call for %s: %w", path, err)
	}
	return llmparse.ParseFindings(resp.Content, a.Name(), finding.TypeAttentionPoint)
}
