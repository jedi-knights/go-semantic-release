package plugins

import (
	"context"
	"fmt"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance checks.
var (
	_ ports.Plugin                 = (*GitPlugin)(nil)
	_ ports.VerifyConditionsPlugin = (*GitPlugin)(nil)
	_ ports.PublishPlugin          = (*GitPlugin)(nil)
)

// GitPlugin handles git operations: verifyConditions (git access), prepare (commit changes), publish (tag + push).
type GitPlugin struct {
	git        ports.GitRepository
	tagService ports.TagService
	fs         ports.FileSystem
	logger     ports.Logger
	identity   domain.GitIdentity
	assets     []string // files to commit in prepare step
}

// NewGitPlugin creates the built-in git plugin.
func NewGitPlugin(
	git ports.GitRepository,
	tagService ports.TagService,
	fs ports.FileSystem,
	logger ports.Logger,
	identity domain.GitIdentity,
	assets []string,
) *GitPlugin {
	return &GitPlugin{
		git:        git,
		tagService: tagService,
		fs:         fs,
		logger:     logger,
		identity:   identity,
		assets:     assets,
	}
}

func (p *GitPlugin) Name() string { return "git" }

func (p *GitPlugin) VerifyConditions(ctx context.Context, rc *domain.ReleaseContext) error {
	_, err := p.git.CurrentBranch(ctx)
	if err != nil {
		return fmt.Errorf("unable to access git repository: %w", err)
	}
	return nil
}

func (p *GitPlugin) Publish(ctx context.Context, rc *domain.ReleaseContext) (*domain.ProjectReleaseResult, error) {
	if rc.CurrentProject == nil {
		return nil, nil
	}

	tagName, err := p.tagService.FormatTag(rc.CurrentProject.Project.Name, rc.CurrentProject.NextVersion)
	if err != nil {
		return nil, fmt.Errorf("formatting tag: %w", err)
	}
	rc.TagName = tagName

	headHash, err := p.git.HeadHash(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting HEAD hash: %w", err)
	}

	message := fmt.Sprintf("chore(release): %s", tagName)
	if rc.Notes != "" {
		message = rc.Notes
	}

	if err := p.git.CreateTag(ctx, tagName, headHash, message); err != nil {
		return nil, fmt.Errorf("creating tag %s: %w", tagName, err)
	}

	if err := p.git.PushTag(ctx, tagName); err != nil {
		return nil, fmt.Errorf("pushing tag %s: %w", tagName, err)
	}

	p.logger.Info("created and pushed tag", "tag", tagName)

	return &domain.ProjectReleaseResult{
		Project:    rc.CurrentProject.Project,
		Version:    rc.CurrentProject.NextVersion,
		TagName:    tagName,
		TagCreated: true,
		Changelog:  rc.Notes,
	}, nil
}
