// White-box test for the unexported toExternalContext conversion function.
// Uses package plugins (not plugins_test) to access internal types directly.
package plugins

import (
	"errors"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestToExternalContext_BasicFields(t *testing.T) {
	rc := &domain.ReleaseContext{
		Branch:        "main",
		DryRun:        true,
		CI:            true,
		RepositoryURL: "https://github.com/org/repo",
		TagName:       "v1.0.0",
		Notes:         "## Release notes",
	}

	ctx := toExternalContext(rc)

	if ctx.Branch != "main" {
		t.Errorf("Branch = %q, want main", ctx.Branch)
	}
	if !ctx.DryRun {
		t.Error("DryRun should be true")
	}
	if !ctx.CI {
		t.Error("CI should be true")
	}
	if ctx.RepositoryURL != "https://github.com/org/repo" {
		t.Errorf("RepositoryURL = %q, want https://github.com/org/repo", ctx.RepositoryURL)
	}
	if ctx.TagName != "v1.0.0" {
		t.Errorf("TagName = %q, want v1.0.0", ctx.TagName)
	}
	if ctx.Notes != "## Release notes" {
		t.Errorf("Notes = %q, want ## Release notes", ctx.Notes)
	}
}

func TestToExternalContext_ErrorField(t *testing.T) {
	rc := &domain.ReleaseContext{
		Error: errors.New("something went wrong"),
	}

	ctx := toExternalContext(rc)

	if ctx.Error != "something went wrong" {
		t.Errorf("Error = %q, want 'something went wrong'", ctx.Error)
	}
}

func TestToExternalContext_NoError(t *testing.T) {
	rc := &domain.ReleaseContext{}
	ctx := toExternalContext(rc)
	if ctx.Error != "" {
		t.Errorf("Error should be empty when rc.Error is nil, got %q", ctx.Error)
	}
}

func TestToExternalContext_CommitsMapped(t *testing.T) {
	rc := &domain.ReleaseContext{
		Commits: []domain.Commit{
			{Hash: "abc1234", Type: "feat", Scope: "auth", Description: "add login", IsBreakingChange: false},
			{Hash: "def5678", Type: "fix", IsBreakingChange: true},
		},
	}

	ctx := toExternalContext(rc)

	if len(ctx.Commits) != 2 {
		t.Fatalf("Commits len = %d, want 2", len(ctx.Commits))
	}
	if ctx.Commits[0].Hash != "abc1234" {
		t.Errorf("Commits[0].Hash = %q, want abc1234", ctx.Commits[0].Hash)
	}
	if ctx.Commits[0].Scope != "auth" {
		t.Errorf("Commits[0].Scope = %q, want auth", ctx.Commits[0].Scope)
	}
	if !ctx.Commits[1].Breaking {
		t.Error("Commits[1].Breaking should be true")
	}
}

func TestToExternalContext_NilCurrentProject(t *testing.T) {
	rc := &domain.ReleaseContext{CurrentProject: nil}
	ctx := toExternalContext(rc)

	if ctx.Project != nil {
		t.Error("Project should be nil when CurrentProject is nil")
	}
	if ctx.NextVersion != "" {
		t.Errorf("NextVersion should be empty, got %q", ctx.NextVersion)
	}
}

func TestToExternalContext_WithCurrentProject(t *testing.T) {
	rc := &domain.ReleaseContext{
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "my-svc", Path: "./my-svc"},
			NextVersion: domain.NewVersion(2, 1, 0),
		},
	}

	ctx := toExternalContext(rc)

	if ctx.Project == nil {
		t.Fatal("Project should not be nil when CurrentProject is set")
	}
	if ctx.Project.Name != "my-svc" {
		t.Errorf("Project.Name = %q, want my-svc", ctx.Project.Name)
	}
	if ctx.NextVersion != "2.1.0" {
		t.Errorf("NextVersion = %q, want 2.1.0", ctx.NextVersion)
	}
}
