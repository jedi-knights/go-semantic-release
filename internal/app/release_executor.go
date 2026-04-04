package app

import (
	"context"
	"fmt"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// ReleaseExecutor executes a release plan by creating tags and publishing releases.
type ReleaseExecutor struct {
	git        ports.GitRepository
	tagService ports.TagService
	changelog  ports.ChangelogGenerator
	publisher  ports.ReleasePublisher
	logger     ports.Logger
	sections   []domain.ChangelogSectionConfig
}

// MustNewReleaseExecutor creates a release executor.
// All parameters are required and must be non-nil. For publisher, pass a
// noopPublisher (available from the DI container via di.Container.ReleasePublisher)
// when publishing is disabled rather than passing nil.
// Panics on any nil argument — these are programming errors, not runtime errors.
func MustNewReleaseExecutor(
	git ports.GitRepository,
	tagService ports.TagService,
	changelog ports.ChangelogGenerator,
	publisher ports.ReleasePublisher,
	logger ports.Logger,
	sections []domain.ChangelogSectionConfig,
) *ReleaseExecutor {
	if git == nil {
		panic("MustNewReleaseExecutor: git must not be nil")
	}
	if tagService == nil {
		panic("MustNewReleaseExecutor: tagService must not be nil")
	}
	if changelog == nil {
		panic("MustNewReleaseExecutor: changelog must not be nil")
	}
	if publisher == nil {
		panic("MustNewReleaseExecutor: publisher must not be nil; use noopPublisher for no-op behavior")
	}
	if logger == nil {
		panic("MustNewReleaseExecutor: logger must not be nil")
	}
	return &ReleaseExecutor{
		git:        git,
		tagService: tagService,
		changelog:  changelog,
		publisher:  publisher,
		logger:     logger,
		sections:   sections,
	}
}

// Execute runs the release for all releasable projects in the plan.
//
// Error model:
//   - Context cancellation and tag/push failures are returned directly and abort
//     the loop immediately. These are hard failures: git state may be partially
//     mutated (e.g. a local tag exists without a corresponding push), so continuing
//     to the next project would compound the inconsistency.
//   - Publish failures (e.g. GitHub release creation) are soft: the tag is already
//     pushed, so the release is technically done. These are collected into
//     result.Projects[i].Error so the caller can report all failures before exiting.
//
// Use result.HasErrors() to check whether any per-project publish error occurred.
func (e *ReleaseExecutor) Execute(ctx context.Context, plan *domain.ReleasePlan) (*domain.ReleaseResult, error) {
	result := &domain.ReleaseResult{DryRun: plan.DryRun}

	releasable := plan.ReleasableProjects()
	for i := range releasable {
		// Cancellation is checked between projects, not during an in-progress
		// executeProject call. If createAndPushTag is blocked on a slow network
		// operation the context is not respected until the current project finishes.
		// This is intentional: aborting mid-tag would leave git state inconsistent.
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("release cancelled: %w", err)
		}
		pr, err := e.executeProject(ctx, releasable[i], plan)
		if err != nil {
			// Hard failure (tag/push): abort immediately rather than continuing to
			// create more tags in an inconsistent state.
			return nil, fmt.Errorf("tagging %s: %w", releasable[i].Project.Name, err)
		}
		if pr.Error != nil {
			e.logger.Error("publish failed", "project", pr.Project.Name, "error", pr.Error)
		}
		result.Projects = append(result.Projects, pr)
	}

	return result, nil
}

func (e *ReleaseExecutor) executeProject(
	ctx context.Context,
	pp domain.ProjectReleasePlan,
	plan *domain.ReleasePlan,
) (domain.ProjectReleaseResult, error) {
	result := domain.ProjectReleaseResult{
		Project:        pp.Project,
		CurrentVersion: pp.CurrentVersion,
		Version:        pp.NextVersion,
	}

	// Generate changelog.
	notes, err := e.changelog.Generate(pp.NextVersion, pp.Project.Name, pp.Commits, e.sections)
	if err != nil {
		return result, domain.NewReleaseError("generate-notes", err)
	}
	result.Changelog = notes

	// Format tag name.
	tagName, err := e.tagService.FormatTag(pp.Project.Name, pp.NextVersion)
	if err != nil {
		return result, domain.NewReleaseError("format-tag", err)
	}
	result.TagName = tagName

	if plan.DryRun {
		result.Skipped = true
		result.SkipReason = "dry run"
		e.logger.Info("dry run: would create tag", "tag", tagName, "version", pp.NextVersion)
		return result, nil
	}

	// Create and push tag.
	if err := e.createAndPushTag(ctx, tagName, notes); err != nil {
		return result, err
	}
	result.TagCreated = true

	// Publish release. Publish failures are soft: the tag is already pushed so
	// the release is technically done. Store the error in result rather than
	// returning it so the caller can continue with remaining projects.
	published, publishURL, publishErr := e.publish(ctx, pp, tagName, notes, plan.Policy)
	if publishErr != nil {
		result.SetError(publishErr)
	} else {
		result.Published = published
		result.PublishURL = publishURL
	}

	// Log at different levels so operators can distinguish a full success from
	// a partial one (tag pushed, publish failed) without parsing the error.
	if result.Error == nil {
		e.logger.Info("release completed", "project", pp.Project.Name, "version", pp.NextVersion, "tag", tagName)
	} else {
		e.logger.Warn("release partially completed (publish failed)", "project", pp.Project.Name, "version", pp.NextVersion, "tag", tagName, "error", result.Error)
	}
	return result, nil
}

func (e *ReleaseExecutor) createAndPushTag(ctx context.Context, tagName, message string) error {
	headHash, err := e.git.HeadHash(ctx)
	if err != nil {
		return domain.NewReleaseError("get-head", err)
	}

	if err := e.git.CreateTag(ctx, tagName, headHash, message); err != nil {
		return domain.NewReleaseError("create-tag", err)
	}

	if err := e.git.PushTag(ctx, tagName); err != nil {
		return domain.NewReleaseError("push-tag", err)
	}
	return nil
}

// publish calls the publisher and returns (published, publishURL, err).
// Returning only the two fields callers actually use avoids implying that the
// other ProjectReleaseResult zero-value fields (TagCreated, Project, …) are meaningful.
func (e *ReleaseExecutor) publish(
	ctx context.Context,
	pp domain.ProjectReleasePlan,
	tagName, notes string,
	policy *domain.BranchPolicy,
) (published bool, publishURL string, err error) {
	isPrerelease := policy != nil && policy.Prerelease

	result, err := e.publisher.Publish(ctx, ports.PublishParams{
		TagName:    tagName,
		Version:    pp.NextVersion,
		Project:    pp.Project.Name,
		Changelog:  notes,
		Prerelease: isPrerelease,
	})
	if err != nil {
		return false, "", domain.NewReleaseError("publish", err)
	}
	return result.Published, result.PublishURL, nil
}
