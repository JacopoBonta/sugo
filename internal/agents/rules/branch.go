package rules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jacopobonta/sugo/internal/finding"
)

func checkBranch(headRef string, patterns []string, ticketRE *regexp.Regexp) []finding.Finding {
	var findings []finding.Finding

	if len(patterns) > 0 {
		matched := false
		for _, pat := range patterns {
			re := regexp.MustCompile(pat) // validated at config load time
			if re.MatchString(headRef) {
				matched = true
				break
			}
		}
		if !matched {
			fix := fmt.Sprintf("Rename branch to match one of: %v", patterns)
			findings = append(findings, finding.Finding{
				Agent:    "rules",
				Severity: finding.SeverityMedium,
				Type:     finding.TypeFix,
				Location: finding.Location{},
				Message:  fmt.Sprintf("Branch %q does not match any required pattern", headRef),
				Fix:      &fix,
			})
		}
	}

	// release/ branches use a version suffix instead of a ticket ID.
	if ticketRE != nil && !strings.HasPrefix(headRef, "release/") {
		if !ticketRE.MatchString(headRef) {
			fix := fmt.Sprintf("Append the ticket ID to the branch name (pattern %q), e.g. my-branch-com-123", ticketRE.String())
			findings = append(findings, finding.Finding{
				Agent:    "rules",
				Severity: finding.SeverityMedium,
				Type:     finding.TypeFix,
				Message:  fmt.Sprintf("Branch %q is missing a trailing ticket ID", headRef),
				Fix:      &fix,
			})
		}
	}

	return findings
}
