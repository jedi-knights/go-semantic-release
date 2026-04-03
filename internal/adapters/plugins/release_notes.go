package plugins

import (
	"context"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance checks.
var (
	_ ports.Plugin              = (*ReleaseNotesPlugin)(nil)
	_ ports.GenerateNotesPlugin = (*ReleaseNotesPlugin)(nil)
)

// ReleaseNotesPlugin implements GenerateNotesPlugin using a changelog generator.
type ReleaseNotesPlugin struct {
	generator ports.ChangelogGenerator
	sections  []domain.ChangelogSectionConfig
}

// NewReleaseNotesPlugin creates the default release notes generator plugin.
func NewReleaseNotesPlugin(generator ports.ChangelogGenerator, sections []domain.ChangelogSectionConfig) *ReleaseNotesPlugin {
	return &ReleaseNotesPlugin{generator: generator, sections: sections}
}

func (p *ReleaseNotesPlugin) Name() string { return "release-notes-generator" }

func (p *ReleaseNotesPlugin) GenerateNotes(_ context.Context, rc *domain.ReleaseContext) (string, error) {
	if rc.CurrentProject == nil {
		return "", nil
	}
	return p.generator.Generate(
		rc.CurrentProject.NextVersion,
		rc.CurrentProject.Project.Name,
		rc.CurrentProject.Commits,
		p.sections,
	)
}
