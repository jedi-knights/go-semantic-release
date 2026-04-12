package plugins_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/plugins"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports/mocks"
)

func TestReleaseNotesPlugin_Name(t *testing.T) {
	ctrl := gomock.NewController(t)
	p := plugins.NewReleaseNotesPlugin(mocks.NewMockChangelogGenerator(ctrl), nil)
	if p.Name() != "release-notes-generator" {
		t.Errorf("Name() = %q, want release-notes-generator", p.Name())
	}
}

func TestReleaseNotesPlugin_GenerateNotes_NilProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGen := mocks.NewMockChangelogGenerator(ctrl)
	// No calls to mockGen expected since CurrentProject is nil.
	p := plugins.NewReleaseNotesPlugin(mockGen, nil)

	rc := &domain.ReleaseContext{CurrentProject: nil}
	notes, err := p.GenerateNotes(context.Background(), rc)
	if err != nil {
		t.Fatalf("GenerateNotes() error = %v", err)
	}
	if notes != "" {
		t.Errorf("GenerateNotes() with nil project = %q, want empty string", notes)
	}
}

func TestReleaseNotesPlugin_GenerateNotes_WithProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGen := mocks.NewMockChangelogGenerator(ctrl)
	sections := domain.DefaultChangelogSections()

	commits := []domain.Commit{{Type: "feat", Description: "add feature"}}
	version := domain.NewVersion(1, 0, 0)
	mockGen.EXPECT().Generate(version, "my-svc", commits, sections).Return("## 1.0.0\n\n- add feature", nil)

	p := plugins.NewReleaseNotesPlugin(mockGen, sections)
	rc := &domain.ReleaseContext{
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "my-svc"},
			NextVersion: version,
			Commits:     commits,
		},
	}

	notes, err := p.GenerateNotes(context.Background(), rc)
	if err != nil {
		t.Fatalf("GenerateNotes() error = %v", err)
	}
	if notes == "" {
		t.Error("GenerateNotes() returned empty notes with a project")
	}
}

func TestReleaseNotesPlugin_GenerateNotes_PropagatesGeneratorError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGen := mocks.NewMockChangelogGenerator(ctrl)
	mockGen.EXPECT().Generate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return("", errors.New("template parse error"))

	p := plugins.NewReleaseNotesPlugin(mockGen, domain.DefaultChangelogSections())
	rc := &domain.ReleaseContext{
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "svc"},
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}

	_, err := p.GenerateNotes(context.Background(), rc)
	if err == nil {
		t.Error("GenerateNotes() should return error when generator fails")
	}
}
