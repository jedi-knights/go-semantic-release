package ports

import "github.com/jedi-knights/go-semantic-release/internal/domain"

// TagService manages tag creation and formatting.
type TagService interface {
	// FormatTag renders a tag name from project and version using the configured template.
	FormatTag(project string, version domain.Version) (string, error)

	// ParseTag extracts project name and version from a tag string.
	ParseTag(tagName string) (project string, version domain.Version, err error)

	// FindLatestTag finds the most recent tag matching the project scope.
	// For repo-level releases, project is empty.
	FindLatestTag(tags []domain.Tag, project string) (*domain.Tag, error)
}
