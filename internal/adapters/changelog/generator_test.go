package changelog_test

import (
	"strings"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/changelog"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestTemplateGenerator_Generate_BasicOutput(t *testing.T) {
	g := changelog.NewTemplateGenerator("")
	version := domain.NewVersion(1, 2, 3)
	commits := []domain.Commit{
		{Hash: "abc1234", Type: "feat", Description: "add login"},
	}
	sections := domain.DefaultChangelogSections()

	out, err := g.Generate(version, "", commits, sections)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !strings.Contains(out, "1.2.3") {
		t.Errorf("output %q should contain version 1.2.3", out)
	}
	if !strings.Contains(out, "add login") {
		t.Errorf("output %q should contain commit description", out)
	}
}

func TestTemplateGenerator_Generate_IncludesProjectName(t *testing.T) {
	g := changelog.NewTemplateGenerator("")
	version := domain.NewVersion(2, 0, 0)
	commits := []domain.Commit{
		{Hash: "def5678", Type: "fix", Description: "correct nil pointer"},
	}

	out, err := g.Generate(version, "my-service", commits, domain.DefaultChangelogSections())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !strings.Contains(out, "my-service") {
		t.Errorf("output %q should contain project name", out)
	}
}

func TestTemplateGenerator_Generate_ScopeInOutput(t *testing.T) {
	g := changelog.NewTemplateGenerator("")
	version := domain.NewVersion(1, 0, 0)
	commits := []domain.Commit{
		{Hash: "aaa0001", Type: "feat", Scope: "auth", Description: "add OAuth"},
	}

	out, err := g.Generate(version, "", commits, domain.DefaultChangelogSections())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !strings.Contains(out, "auth") {
		t.Errorf("output %q should contain scope", out)
	}
}

func TestTemplateGenerator_Generate_BreakingChangesSection(t *testing.T) {
	g := changelog.NewTemplateGenerator("")
	version := domain.NewVersion(2, 0, 0)
	commits := []domain.Commit{
		{Hash: "bbb0001", Type: "feat", Description: "remove legacy API", IsBreakingChange: true},
	}

	out, err := g.Generate(version, "", commits, domain.DefaultChangelogSections())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !strings.Contains(out, "Breaking Changes") {
		t.Errorf("output %q should contain Breaking Changes section", out)
	}
}

func TestTemplateGenerator_Generate_HiddenSectionsOmitted(t *testing.T) {
	g := changelog.NewTemplateGenerator("")
	version := domain.NewVersion(1, 0, 1)
	commits := []domain.Commit{
		{Hash: "ccc0001", Type: "chore", Description: "update deps"},
	}

	out, err := g.Generate(version, "", commits, domain.DefaultChangelogSections())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	// "chore" is a hidden section by default, so its title "Chores" should not appear
	if strings.Contains(out, "Chores") {
		t.Errorf("output %q should NOT contain hidden section title 'Chores'", out)
	}
}

func TestTemplateGenerator_Generate_EmptySectionsOmitted(t *testing.T) {
	g := changelog.NewTemplateGenerator("")
	version := domain.NewVersion(1, 0, 0)
	commits := []domain.Commit{
		{Hash: "ddd0001", Type: "feat", Description: "add feature"},
	}

	out, err := g.Generate(version, "", commits, domain.DefaultChangelogSections())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	// Bug Fixes section should not appear since there are no fix commits
	if strings.Contains(out, "Bug Fixes") {
		t.Errorf("output %q should NOT contain empty 'Bug Fixes' section", out)
	}
}

func TestTemplateGenerator_Generate_ShortHash(t *testing.T) {
	g := changelog.NewTemplateGenerator("")
	version := domain.NewVersion(1, 0, 0)
	commits := []domain.Commit{
		{Hash: "abcdef1234567890", Type: "fix", Description: "fix crash"},
	}

	out, err := g.Generate(version, "", commits, domain.DefaultChangelogSections())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	// Short hash (7 chars) should appear in output
	if !strings.Contains(out, "abcdef1") {
		t.Errorf("output %q should contain short hash abcdef1", out)
	}
}

func TestTemplateGenerator_Generate_CustomTemplate(t *testing.T) {
	customTmpl := "VERSION={{.Version}}"
	g := changelog.NewTemplateGenerator(customTmpl)
	version := domain.NewVersion(3, 1, 0)

	out, err := g.Generate(version, "", nil, domain.DefaultChangelogSections())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if out != "VERSION=3.1.0" {
		t.Errorf("Generate() with custom template = %q, want %q", out, "VERSION=3.1.0")
	}
}

func TestTemplateGenerator_Generate_InvalidCustomTemplate(t *testing.T) {
	g := changelog.NewTemplateGenerator("{{.Unclosed")

	_, err := g.Generate(domain.NewVersion(1, 0, 0), "", nil, nil)
	if err == nil {
		t.Error("Generate() with invalid template should return error")
	}
}

func TestTemplateGenerator_Generate_NoCommits(t *testing.T) {
	g := changelog.NewTemplateGenerator("")
	version := domain.NewVersion(1, 0, 0)

	out, err := g.Generate(version, "", nil, domain.DefaultChangelogSections())
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if !strings.Contains(out, "1.0.0") {
		t.Errorf("output %q should contain version even with no commits", out)
	}
}
