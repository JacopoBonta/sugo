// Package finding defines the shared Finding type produced by all analysis agents.
package finding

// Severity represents the urgency of a finding.
type Severity string

const (
	SeverityHigh   Severity = "high"
	SeverityMedium Severity = "medium"
	SeverityLow    Severity = "low"
)

// severityRank maps severity to a numeric rank for comparison (higher = more severe).
var severityRank = map[Severity]int{
	SeverityHigh:   3,
	SeverityMedium: 2,
	SeverityLow:    1,
}

// Rank returns the numeric severity rank (higher = more severe).
func (s Severity) Rank() int {
	return severityRank[s]
}

// FindingType distinguishes actionable fixes from attention points.
type FindingType string

const (
	TypeFix            FindingType = "fix"
	TypeAttentionPoint FindingType = "attention_point"
)

// Location identifies the file and line range a finding refers to.
type Location struct {
	File      string `json:"file"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
}

// Finding is the canonical output type every agent produces.
// Fix is nil for AttentionPoint findings.
type Finding struct {
	Agent    string      `json:"agent"`
	Severity Severity    `json:"severity"`
	Type     FindingType `json:"type"`
	Location Location    `json:"location"`
	Message  string      `json:"message"`
	Fix      *string     `json:"fix"`
}
