// Package renderer formats a report for human or machine consumption.
package renderer

import (
	"fmt"
	"io"

	"github.com/jacopobonta/sugo/internal/orchestrator"
)

// Render writes the report to w in the requested format.
// format must be "terminal", "json", or "markdown".
func Render(w io.Writer, report *orchestrator.Report, format string) error {
	switch format {
	case "json":
		return renderJSON(w, report)
	case "markdown", "md":
		return renderMarkdown(w, report)
	case "terminal", "":
		return renderTerminal(w, report)
	default:
		return fmt.Errorf("unknown output format %q; use terminal, json, or markdown", format)
	}
}
