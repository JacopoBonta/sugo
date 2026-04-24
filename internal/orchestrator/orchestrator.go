package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/gh"
)

// Orchestrator fetches a PR, dispatches agents in parallel, and aggregates findings.
type Orchestrator struct {
	ghClient gh.Client
	agents   []agents.Agent
	config   *config.Config
	logger   *slog.Logger
}

// New creates an Orchestrator with the given agent list.
func New(ghClient gh.Client, agentList []agents.Agent, cfg *config.Config, logger *slog.Logger) *Orchestrator {
	return &Orchestrator{
		ghClient: ghClient,
		agents:   agentList,
		config:   cfg,
		logger:   logger,
	}
}

// Run fetches the PR and runs all available agents, returning a Report.
func (o *Orchestrator) Run(ctx context.Context, owner, repo string, number int) (*Report, error) {
	start := time.Now()

	pr, err := o.ghClient.FetchPR(ctx, owner, repo, number)
	if err != nil {
		return nil, fmt.Errorf("fetch PR: %w", err)
	}

	input := &agents.AnalysisInput{PR: pr, Config: o.config}

	type result struct {
		findings []finding.Finding
		err      error
		name     string
	}

	var (
		mu       sync.Mutex
		results  []result
		warnings []string
		wg       sync.WaitGroup
	)

	for _, a := range o.agents {
		if !a.Available(o.config) {
			msg := fmt.Sprintf("agent %s skipped: not available", a.Name())
			warnings = append(warnings, msg)
			o.logger.Warn("agent skipped", "agent", a.Name())
			continue
		}

		wg.Add(1)
		go func(ag agents.Agent) {
			defer wg.Done()
			f, err := ag.Analyze(ctx, input)
			mu.Lock()
			results = append(results, result{findings: f, err: err, name: ag.Name()})
			mu.Unlock()
		}(a)
	}
	wg.Wait()

	var allFindings []finding.Finding
	for _, r := range results {
		if r.err != nil {
			msg := fmt.Sprintf("agent %s failed: %v", r.name, r.err)
			warnings = append(warnings, msg)
			o.logger.Warn("agent failed", "agent", r.name, "error", r.err)
			continue
		}
		allFindings = append(allFindings, r.findings...)
	}

	allFindings = finding.Deduplicate(allFindings)

	return &Report{
		PR:       *pr,
		Findings: allFindings,
		Warnings: warnings,
		Duration: time.Since(start),
	}, nil
}
