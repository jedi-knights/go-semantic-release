package tmpl_test

import (
	"testing"

	tmpl "github.com/jedi-knights/go-semantic-release/internal/adapters/template"
)

func TestGoTemplateRenderer_Render_SimpleSubstitution(t *testing.T) {
	r := tmpl.NewGoTemplateRenderer()

	got, err := r.Render("Hello, {{.Name}}!", map[string]string{"Name": "World"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if got != "Hello, World!" {
		t.Errorf("Render() = %q, want %q", got, "Hello, World!")
	}
}

func TestGoTemplateRenderer_Render_EmptyTemplate(t *testing.T) {
	r := tmpl.NewGoTemplateRenderer()

	got, err := r.Render("", nil)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if got != "" {
		t.Errorf("Render() = %q, want empty string", got)
	}
}

func TestGoTemplateRenderer_Render_InvalidTemplate(t *testing.T) {
	r := tmpl.NewGoTemplateRenderer()

	_, err := r.Render("{{.Unclosed", nil)
	if err == nil {
		t.Error("Render() with invalid template should return error")
	}
}

func TestGoTemplateRenderer_Render_ExecutionError(t *testing.T) {
	r := tmpl.NewGoTemplateRenderer()

	// Accessing a field on a nil pointer causes a template execution error.
	type Inner struct{ Value string }
	type Outer struct{ Inner *Inner }

	_, err := r.Render("{{.Inner.Value}}", Outer{Inner: nil})
	if err == nil {
		t.Error("Render() with nil pointer field access should return error")
	}
}

func TestGoTemplateRenderer_Render_StructData(t *testing.T) {
	r := tmpl.NewGoTemplateRenderer()

	type data struct {
		Version string
		Project string
	}

	got, err := r.Render("{{.Project}}/v{{.Version}}", data{Version: "1.2.3", Project: "my-service"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if got != "my-service/v1.2.3" {
		t.Errorf("Render() = %q, want %q", got, "my-service/v1.2.3")
	}
}
