// Package orchestrator wires together all agents and produces a final Report.
package orchestrator

import (
	"time"

	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/gh"
)

// Report is the output of a full PR analysis run.
type Report struct {
	PR       gh.PullRequest    `json:"pr"`
	Findings []finding.Finding `json:"findings"`
	Warnings []string          `json:"warnings"` // skipped or failed agents
	Duration time.Duration     `json:"duration"`
}
