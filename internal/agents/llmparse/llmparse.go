// Package llmparse provides helpers for parsing LLM JSON responses into findings.
package llmparse

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jacopobonta/sugo/internal/finding"
)

// findingJSON mirrors the JSON schema agents ask the LLM to return.
type findingJSON struct {
	Agent    string `json:"agent"`
	Severity string `json:"severity"`
	Location struct {
		File      string `json:"file"`
		LineStart int    `json:"line_start"`
		LineEnd   int    `json:"line_end"`
	} `json:"location"`
	Message string  `json:"message"`
	Fix     *string `json:"fix"`
}

type responseJSON struct {
	Findings []findingJSON `json:"findings"`
}

// ParseFindings extracts findings from a raw LLM response string.
// The response may be wrapped in a markdown code block.
func ParseFindings(content, agentName string, findingType finding.FindingType) ([]finding.Finding, error) {
	content = extractJSON(content)

	var resp responseJSON
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		return nil, fmt.Errorf("parse LLM response: %w", err)
	}

	results := make([]finding.Finding, 0, len(resp.Findings))
	for _, f := range resp.Findings {
		sev := parseSeverity(f.Severity)
		name := f.Agent
		if name == "" {
			name = agentName
		}
		results = append(results, finding.Finding{
			Agent:    name,
			Severity: sev,
			Type:     findingType,
			Location: finding.Location{
				File:      f.Location.File,
				LineStart: f.Location.LineStart,
				LineEnd:   f.Location.LineEnd,
			},
			Message: f.Message,
			Fix:     f.Fix,
		})
	}
	return results, nil
}

// extractJSON strips markdown code fences if present.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.SplitN(s, "\n", 2)
		if len(lines) == 2 {
			s = lines[1]
		}
		s = strings.TrimSuffix(s, "```")
		s = strings.TrimSpace(s)
	}
	return s
}

func parseSeverity(s string) finding.Severity {
	switch strings.ToLower(s) {
	case "high":
		return finding.SeverityHigh
	case "low":
		return finding.SeverityLow
	default:
		return finding.SeverityMedium
	}
}
