package renderer

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/jacopobonta/sugo/internal/finding"
	"github.com/jacopobonta/sugo/internal/gh"
	"github.com/jacopobonta/sugo/internal/orchestrator"
)

// ANSI color codes.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGreen  = "\033[32m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
	colorBorder = "\033[38;5;242m" // Sleek dark gray
)

func renderTerminal(w io.Writer, report *orchestrator.Report) error {
	pr := report.PR

	boxWidth := 74
	printBoxHeader(w, pr, boxWidth)

	findings := sortedFindings(report.Findings)

	high := countBySeverity(findings, finding.SeverityHigh)
	med := countBySeverity(findings, finding.SeverityMedium)
	low := countBySeverity(findings, finding.SeverityLow)

	// Summary section
	fmt.Fprintf(w, "%s📊 SUMMARY OF FINDINGS%s\n", colorBold, colorReset)
	fmt.Fprintf(w, "%s──────────────────────────────────────────────────────────────────────────%s\n", colorBorder, colorReset)
	fmt.Fprintf(w, "  Total Findings: %d\n", len(findings))
	fmt.Fprintf(w, "  🔴 %sHIGH%s   : %d\n", colorRed, colorReset, high)
	fmt.Fprintf(w, "  🟡 %sMEDIUM%s : %d\n", colorYellow, colorReset, med)
	fmt.Fprintf(w, "  🔵 %sLOW%s    : %d\n", colorCyan, colorReset, low)
	fmt.Fprintf(w, "\n")

	// Findings section
	fmt.Fprintf(w, "%s📋 DETAIL OF FINDINGS%s\n", colorBold, colorReset)
	fmt.Fprintf(w, "%s──────────────────────────────────────────────────────────────────────────%s\n", colorBorder, colorReset)

	if len(findings) == 0 {
		fmt.Fprintf(w, "  %sNo findings. Great job!%s\n", colorGreen, colorReset)
	} else {
		for _, f := range findings {
			printFinding(w, f)
		}
	}

	if len(report.Warnings) > 0 {
		fmt.Fprintf(w, "\n")
		fmt.Fprintf(w, "%s⚠️ WARNINGS%s\n", colorBold, colorReset)
		fmt.Fprintf(w, "%s──────────────────────────────────────────────────────────────────────────%s\n", colorBorder, colorReset)
		for _, w2 := range report.Warnings {
			fmt.Fprintf(w, "  %s%s%s\n", colorYellow, w2, colorReset)
		}
	}

	fmt.Fprintf(w, "\n%sAnalysis completed in %s%s\n", colorGray, report.Duration.Round(1e6), colorReset)
	return nil
}

func printBoxHeader(w io.Writer, pr gh.PullRequest, boxWidth int) {
	topBorder := "┌" + strings.Repeat("─", boxWidth-2) + "┐"
	botBorder := "└" + strings.Repeat("─", boxWidth-2) + "┘"

	fmt.Fprintf(w, "%s%s%s\n", colorBorder, topBorder, colorReset)

	line1 := fmt.Sprintf("PR #%d: %s (%s/%s)", pr.Number, pr.Title, pr.Owner, pr.Repo)
	printBoxLine(w, line1, boxWidth)

	line2 := fmt.Sprintf("Branch: %s → %s", pr.HeadRef, pr.BaseRef)
	printBoxLine(w, line2, boxWidth)

	fmt.Fprintf(w, "%s%s%s\n\n", colorBorder, botBorder, colorReset)
}

// Map pull request to matching interface (our gh.PullRequest satisfies this or we assert directly)
func printBoxLine(w io.Writer, content string, boxWidth int) {
	innerLimit := boxWidth - 6
	var line string
	if len(content) > innerLimit {
		line = content[:innerLimit-3] + "..."
	} else {
		line = content + strings.Repeat(" ", innerLimit-len(content))
	}
	fmt.Fprintf(w, "%s│  %s%s%s  │%s\n", colorBorder, colorReset, line, colorBorder, colorReset)
}

func printFinding(w io.Writer, f finding.Finding) {
	col := severityColor(f.Severity)
	emoji := agentEmoji(f.Agent)
	bullet := severityBullet(f.Severity)

	fmt.Fprintf(w, "\n%s %s[%s]%s %s%s%s%s: %s\n",
		bullet, col, strings.ToUpper(string(f.Severity)), colorReset,
		emoji, colorBold, f.Agent, colorReset, f.Message)

	if f.Location.File != "" {
		if f.Location.LineStart == f.Location.LineEnd {
			fmt.Fprintf(w, "     %s→ %s:%d%s\n", colorGray, f.Location.File, f.Location.LineStart, colorReset)
		} else {
			fmt.Fprintf(w, "     %s→ %s:%d-%d%s\n", colorGray, f.Location.File, f.Location.LineStart, f.Location.LineEnd, colorReset)
		}
	}

	if f.Fix != nil {
		fmt.Fprintf(w, "     %sSuggested Fix:%s\n", colorGreen, colorReset)
		lines := strings.Split(strings.TrimSpace(*f.Fix), "\n")
		for _, line := range lines {
			fmt.Fprintf(w, "       %s%s%s\n", colorGray, line, colorReset)
		}
	} else {
		fmt.Fprintf(w, "     %s(attention point)%s\n", colorGray, colorReset)
	}
}

func severityColor(s finding.Severity) string {
	switch s {
	case finding.SeverityHigh:
		return colorRed
	case finding.SeverityMedium:
		return colorYellow
	default:
		return colorCyan
	}
}

func severityBullet(s finding.Severity) string {
	switch s {
	case finding.SeverityHigh:
		return "🔴"
	case finding.SeverityMedium:
		return "🟡"
	default:
		return "🔵"
	}
}

func agentEmoji(agent string) string {
	switch strings.ToLower(agent) {
	case "rules":
		return "📏 "
	case "lint":
		return "⚙️  "
	case "logic":
		return "🧠 "
	case "focus":
		return "👁️  "
	case "analysisgap":
		return "🔍 "
	case "security":
		return "🛡️  "
	case "coverage":
		return "🧪 "
	default:
		return "🤖 "
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
