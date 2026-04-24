package rules

import (
	"fmt"

	"github.com/jacopobonta/sugo/internal/finding"
)

func checkLabels(prLabels []string, required []string) []finding.Finding {
	if len(required) == 0 {
		return nil
	}
	have := make(map[string]bool, len(prLabels))
	for _, l := range prLabels {
		have[l] = true
	}
	var findings []finding.Finding
	for _, req := range required {
		if !have[req] {
			fix := fmt.Sprintf("Add the %q label to this PR", req)
			findings = append(findings, finding.Finding{
				Agent:    "rules",
				Severity: finding.SeverityMedium,
				Type:     finding.TypeFix,
				Message:  fmt.Sprintf("Required label %q is missing", req),
				Fix:      &fix,
			})
		}
	}
	return findings
}
