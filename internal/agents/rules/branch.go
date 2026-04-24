package rules

import (
	"fmt"
	"regexp"

	"github.com/jacopobonta/sugo/internal/finding"
)

func checkBranch(headRef string, patterns []string) []finding.Finding {
	if len(patterns) == 0 {
		return nil
	}
	for _, pat := range patterns {
		re := regexp.MustCompile(pat) // validated at config load time
		if re.MatchString(headRef) {
			return nil
		}
	}
	fix := fmt.Sprintf("Rename branch to match one of: %v", patterns)
	return []finding.Finding{
		{
			Agent:    "rules",
			Severity: finding.SeverityMedium,
			Type:     finding.TypeFix,
			Location: finding.Location{},
			Message:  fmt.Sprintf("Branch %q does not match any required pattern", headRef),
			Fix:      &fix,
		},
	}
}
