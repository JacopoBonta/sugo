package renderer

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/orchestrator"
)

// ANSI color codes.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorBold   = "\033[1m"
)

func renderTerminal(w io.Writer, report *orchestrator.Report) error {
	pr := report.PR
	fmt.Fprintf(w, "%sPR #%d: %s (%s/%s)%s\n", colorBold, pr.Number, pr.Title, pr.Owner, pr.Repo, colorReset)
	fmt.Fprintf(w, "Branch: %s → %s\n", pr.HeadRef, pr.BaseRef)
	fmt.Fprintln(w)

	findings := sortedFindings(report.Findings)

	high := countBySeverity(findings, finding.SeverityHigh)
	med := countBySeverity(findings, finding.SeverityMedium)
	low := countBySeverity(findings, finding.SeverityLow)

	fmt.Fprintf(w, "%s=== Findings (%d total: %d high, %d medium, %d low) ===%s\n",
		colorBold, len(findings), high, med, low, colorReset)

	if len(findings) == 0 {
		fmt.Fprintln(w, "  No findings.")
	}

	for _, f := range findings {
		printFinding(w, f)
	}

	if len(report.Warnings) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "%s--- Warnings ---%s\n", colorBold, colorReset)
		for _, w2 := range report.Warnings {
			fmt.Fprintf(w, "  %s\n", w2)
		}
	}

	fmt.Fprintf(w, "\nAnalysis completed in %s\n", report.Duration.Round(1e6))
	return nil
}

func printFinding(w io.Writer, f finding.Finding) {
	col := severityColor(f.Severity)
	fmt.Fprintf(w, "\n%s[%s]%s %s: %s\n", col, strings.ToUpper(string(f.Severity)), colorReset, f.Agent, f.Message)

	if f.Location.File != "" {
		if f.Location.LineStart == f.Location.LineEnd {
			fmt.Fprintf(w, "  → %s:%d\n", f.Location.File, f.Location.LineStart)
		} else {
			fmt.Fprintf(w, "  → %s:%d-%d\n", f.Location.File, f.Location.LineStart, f.Location.LineEnd)
		}
	}

	if f.Fix != nil {
		fmt.Fprintf(w, "  Fix: %s\n", *f.Fix)
	} else {
		fmt.Fprintln(w, "  (attention point)")
	}
}

func severityColor(s finding.Severity) string {
	switch s {
	case finding.SeverityHigh:
		return colorRed
	case finding.SeverityMedium:
		return colorYellow
	default:
		return colorBlue
	}
}

func sortedFindings(findings []finding.Finding) []finding.Finding {
	sorted := make([]finding.Finding, len(findings))
	copy(sorted, findings)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Severity.Rank() > sorted[j].Severity.Rank()
	})
	return sorted
}

func countBySeverity(findings []finding.Finding, sev finding.Severity) int {
	n := 0
	for _, f := range findings {
		if f.Severity == sev {
			n++
		}
	}
	return n
}
