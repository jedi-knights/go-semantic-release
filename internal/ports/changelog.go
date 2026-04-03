package ports

import "github.com/jedi-knights/go-semantic-release/internal/domain"

// ChangelogGenerator generates release notes from commits.
type ChangelogGenerator interface {
	Generate(version domain.Version, project string, commits []domain.Commit, sections []domain.ChangelogSectionConfig) (string, error)
}
