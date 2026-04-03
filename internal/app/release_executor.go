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

// NewReleaseExecutor creates a release executor.
func NewReleaseExecutor(
	git ports.GitRepository,
	tagService ports.TagService,
	changelog ports.ChangelogGenerator,
	publisher ports.ReleasePublisher,
	logger ports.Logger,
	sections []domain.ChangelogSectionConfig,
) *ReleaseExecutor {
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
func (e *ReleaseExecutor) Execute(ctx context.Context, plan *domain.ReleasePlan) (*domain.ReleaseResult, error) {
	result := &domain.ReleaseResult{DryRun: plan.DryRun}

	releasable := plan.ReleasableProjects()
	for i := range releasable {
		pr, err := e.executeProject(ctx, releasable[i], plan)
		if err != nil {
			pr.Error = err
			e.logger.Error("release failed", "project", releasable[i].Project.Name, "error", err)
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
		Project: pp.Project,
		Version: pp.NextVersion,
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

	// Publish release.
	if e.publisher != nil {
		publishResult, err := e.publish(ctx, pp, tagName, notes, plan.Policy)
		if err != nil {
			return result, err
		}
		result.Published = publishResult.Published
		result.PublishURL = publishResult.PublishURL
	}

	e.logger.Info("release completed", "project", pp.Project.Name, "version", pp.NextVersion, "tag", tagName)
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

func (e *ReleaseExecutor) publish(
	ctx context.Context,
	pp domain.ProjectReleasePlan,
	tagName, notes string,
	policy *domain.BranchPolicy,
) (domain.ProjectReleaseResult, error) {
	isPrerelease := policy != nil && policy.Prerelease

	publishResult, err := e.publisher.Publish(ctx, ports.PublishParams{
		TagName:    tagName,
		Version:    pp.NextVersion,
		Project:    pp.Project.Name,
		Changelog:  notes,
		Prerelease: isPrerelease,
	})
	if err != nil {
		return publishResult, domain.NewReleaseError("publish",
			fmt.Errorf("publishing %s: %w", pp.Project.Name, err))
	}
	return publishResult, nil
}
