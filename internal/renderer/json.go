package renderer

import (
	"encoding/json"
	"io"

	"github.com/jacopobonta/sugo/internal/orchestrator"
)

func renderJSON(w io.Writer, report *orchestrator.Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
