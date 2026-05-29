package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/jacopobonta/sugo/internal/agents"
	"github.com/jacopobonta/sugo/internal/config"
	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/gh"
)

type agentState string

const (
	statePending agentState = "pending"
	stateRunning agentState = "running"
	stateDone    agentState = "done"
	stateFailed  agentState = "failed"
	stateSkipped agentState = "skipped"
)

type agentProgress struct {
	name  string
	state agentState
}

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

	// Determine if we should show the CLI loader
	useSpinner := isStderrTTY() && (o.logger == nil || !o.logger.Enabled(ctx, slog.LevelInfo))

	var progressList []agentProgress
	agentIdxMap := make(map[string]int)
	for _, a := range o.agents {
		name := a.Name()
		state := statePending
		if !a.Available(o.config) {
			state = stateSkipped
		}
		agentIdxMap[name] = len(progressList)
		progressList = append(progressList, agentProgress{
			name:  name,
			state: state,
		})
	}

	var progressMu sync.Mutex
	updateProgress := func(name string, state agentState) {
		progressMu.Lock()
		if idx, ok := agentIdxMap[name]; ok {
			progressList[idx].state = state
		}
		progressMu.Unlock()
	}

	var doneChan chan struct{}
	if useSpinner {
		doneChan = make(chan struct{})
		drawProgress(progressList, 0, true)
		go func() {
			ticker := time.NewTicker(80 * time.Millisecond)
			defer ticker.Stop()
			var spinnerIdx int
			for {
				select {
				case <-ticker.C:
					spinnerIdx++
					progressMu.Lock()
					drawProgress(progressList, spinnerIdx, false)
					progressMu.Unlock()
				case <-doneChan:
					progressMu.Lock()
					drawProgress(progressList, spinnerIdx, false)
					progressMu.Unlock()
					return
				}
			}
		}()
	}

	for _, a := range o.agents {
		if !a.Available(o.config) {
			msg := fmt.Sprintf("agent %s skipped: not available", a.Name())
			warnings = append(warnings, msg)
			o.logger.Warn("agent skipped", "agent", a.Name())
			continue
		}

		wg.Add(1)
		updateProgress(a.Name(), stateRunning)
		go func(ag agents.Agent) {
			defer wg.Done()
			f, err := ag.Analyze(ctx, input)
			mu.Lock()
			results = append(results, result{findings: f, err: err, name: ag.Name()})
			mu.Unlock()

			if err != nil {
				updateProgress(ag.Name(), stateFailed)
			} else {
				updateProgress(ag.Name(), stateDone)
			}
		}(a)
	}
	wg.Wait()

	if useSpinner {
		close(doneChan)
		time.Sleep(10 * time.Millisecond)
		fmt.Fprintln(os.Stderr)
	}

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

func isStderrTTY() bool {
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func drawProgress(progress []agentProgress, spinnerIdx int, firstTime bool) {
	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frame := spinnerFrames[spinnerIdx%len(spinnerFrames)]

	if !firstTime && len(progress) > 1 {
		fmt.Fprintf(os.Stderr, "\033[%dA", len(progress)-1)
	}

	for i, ap := range progress {
		var icon string
		var stateText string
		switch ap.state {
		case statePending:
			icon = "○"
			stateText = "\033[90mpending\033[0m"
		case stateRunning:
			icon = fmt.Sprintf("\033[34m%s\033[0m", frame)
			stateText = "\033[36mrunning\033[0m"
		case stateDone:
			icon = "\033[32m✓\033[0m"
			stateText = "\033[32mdone\033[0m"
		case stateFailed:
			icon = "\033[31m✗\033[0m"
			stateText = "\033[31mfailed\033[0m"
		case stateSkipped:
			icon = "\033[90m-\033[0m"
			stateText = "\033[90mskipped\033[0m"
		}
		if i > 0 {
			fmt.Fprint(os.Stderr, "\n")
		}
		fmt.Fprintf(os.Stderr, "\r\033[K %s  %-12s (%s)", icon, ap.name, stateText)
	}
}
