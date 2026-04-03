package ports

import (
	"context"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

// Plugin is the base interface for all lifecycle plugins.
// A plugin implements one or more of the step-specific interfaces below.
type Plugin interface {
	Name() string
}

// VerifyConditionsPlugin checks that release prerequisites are met.
type VerifyConditionsPlugin interface {
	Plugin
	VerifyConditions(ctx context.Context, rc *domain.ReleaseContext) error
}

// AnalyzeCommitsPlugin determines the release type from commits.
type AnalyzeCommitsPlugin interface {
	Plugin
	AnalyzeCommits(ctx context.Context, rc *domain.ReleaseContext) (domain.ReleaseType, error)
}

// VerifyReleasePlugin validates the pending release before publishing.
type VerifyReleasePlugin interface {
	Plugin
	VerifyRelease(ctx context.Context, rc *domain.ReleaseContext) error
}

// GenerateNotesPlugin generates release notes content.
type GenerateNotesPlugin interface {
	Plugin
	GenerateNotes(ctx context.Context, rc *domain.ReleaseContext) (string, error)
}

// PreparePlugin prepares artifacts and files before publishing (e.g., update CHANGELOG.md, VERSION).
type PreparePlugin interface {
	Plugin
	Prepare(ctx context.Context, rc *domain.ReleaseContext) error
}

// PublishPlugin publishes the release to an external platform.
type PublishPlugin interface {
	Plugin
	Publish(ctx context.Context, rc *domain.ReleaseContext) (*domain.ProjectReleaseResult, error)
}

// AddChannelPlugin adds a release to a distribution channel.
type AddChannelPlugin interface {
	Plugin
	AddChannel(ctx context.Context, rc *domain.ReleaseContext) error
}

// SuccessPlugin notifies of a successful release.
type SuccessPlugin interface {
	Plugin
	Success(ctx context.Context, rc *domain.ReleaseContext) error
}

// FailPlugin notifies of a failed release.
type FailPlugin interface {
	Plugin
	Fail(ctx context.Context, rc *domain.ReleaseContext) error
}
