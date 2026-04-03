package ports

// TemplateRenderer renders templates with the given data.
type TemplateRenderer interface {
	Render(templateStr string, data any) (string, error)
}
