package renderer

import (
	"fmt"
	"io"
	"strings"

	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/orchestrator"
)

func renderMarkdown(w io.Writer, report *orchestrator.Report) error {
	pr := report.PR
	fmt.Fprintf(w, "# PR #%d: %s\n\n", pr.Number, pr.Title)
	fmt.Fprintf(w, "**Repo**: %s/%s  \n", pr.Owner, pr.Repo)
	fmt.Fprintf(w, "**Branch**: `%s` → `%s`\n\n", pr.HeadRef, pr.BaseRef)

	findings := sortedFindings(report.Findings)
	high := countBySeverity(findings, finding.SeverityHigh)
	med := countBySeverity(findings, finding.SeverityMedium)
	low := countBySeverity(findings, finding.SeverityLow)

	fmt.Fprintf(w, "## Findings (%d total: %d high, %d medium, %d low)\n\n", len(findings), high, med, low)

	if len(findings) == 0 {
		fmt.Fprintln(w, "No findings.")
	}

	for _, f := range findings {
		fmt.Fprintf(w, "### [%s] %s — %s\n\n", strings.ToUpper(string(f.Severity)), f.Agent, f.Message)

		if f.Location.File != "" {
			if f.Location.LineStart == f.Location.LineEnd {
				fmt.Fprintf(w, "**Location**: `%s:%d`\n\n", f.Location.File, f.Location.LineStart)
			} else {
				fmt.Fprintf(w, "**Location**: `%s:%d-%d`\n\n", f.Location.File, f.Location.LineStart, f.Location.LineEnd)
			}
		}

		if f.Fix != nil {
			fmt.Fprintf(w, "**Fix**: %s\n\n", *f.Fix)
		} else {
			fmt.Fprint(w, "_Attention point — no fix prescribed._\n\n")
		}
	}

	if len(report.Warnings) > 0 {
		fmt.Fprint(w, "## Warnings\n\n")
		for _, warn := range report.Warnings {
			fmt.Fprintf(w, "- %s\n", warn)
		}
	}

	return nil
}
