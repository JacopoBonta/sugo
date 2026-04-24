package rules

import (
	"fmt"
	"regexp"

	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/gh"
)

// conventionalCommitRE matches "type(scope): description" or "type: description".
var conventionalCommitRE = regexp.MustCompile(`^[a-z]+(\([^)]+\))?!?: .+`)

func checkCommits(commits []gh.Commit) []finding.Finding {
	var findings []finding.Finding
	for _, c := range commits {
		if !conventionalCommitRE.MatchString(c.Message) {
			fix := "Use conventional commit format: type(scope): description (e.g. feat(auth): add login)"
			findings = append(findings, finding.Finding{
				Agent:    "rules",
				Severity: finding.SeverityLow,
				Type:     finding.TypeFix,
				Message:  fmt.Sprintf("Commit %q does not follow conventional commit format", c.Message),
				Fix:      &fix,
			})
		}
	}
	return findings
}
