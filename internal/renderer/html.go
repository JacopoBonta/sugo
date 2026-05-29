package renderer

import (
	_ "embed"
	"html/template"
	"io"

	"github.com/jacopobonta/sugo/internal/orchestrator"
)

//go:embed template.html
var htmlTemplateSource string

func renderHTML(w io.Writer, report *orchestrator.Report) error {
	tmpl, err := template.New("report").Parse(htmlTemplateSource)
	if err != nil {
		return err
	}
	return tmpl.Execute(w, report)
}
