package tmpl

import (
	"bytes"
	"fmt"
	"text/template"
)

// GoTemplateRenderer implements ports.TemplateRenderer using Go's text/template.
type GoTemplateRenderer struct{}

// NewGoTemplateRenderer creates a new Go template renderer.
func NewGoTemplateRenderer() *GoTemplateRenderer {
	return &GoTemplateRenderer{}
}

func (r *GoTemplateRenderer) Render(templateStr string, data any) (string, error) {
	tmpl, err := template.New("render").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}
	return buf.String(), nil
}
