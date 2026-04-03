package tmpl

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance check.
var _ ports.TemplateRenderer = (*GoTemplateRenderer)(nil)

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
