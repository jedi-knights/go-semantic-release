package ports

import (
	"context"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

// ReleasePublisher creates releases on external platforms (e.g., GitHub).
type ReleasePublisher interface {
	Publish(ctx context.Context, params PublishParams) (domain.ProjectReleaseResult, error)
}

// PublishParams contains everything needed to publish a release.
type PublishParams struct {
	TagName    string
	Version    domain.Version
	Project    string
	Changelog  string
	CommitHash string
	Prerelease bool
}
