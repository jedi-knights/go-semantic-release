package plugins

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance checks.
var (
	_ ports.Plugin                 = (*GitPlugin)(nil)
	_ ports.VerifyConditionsPlugin = (*GitPlugin)(nil)
	_ ports.PublishPlugin          = (*GitPlugin)(nil)
)

// GitPlugin handles git operations: verifyConditions (git access), publish (stage → commit → push → tag).
type GitPlugin struct {
	git        ports.GitRepository
	tagService ports.TagService
	fs         ports.FileSystem
	logger     ports.Logger
	identity   domain.GitIdentity
	gitConfig  domain.GitConfig
}

// NewGitPlugin creates the built-in git plugin.
func NewGitPlugin(
	git ports.GitRepository,
	tagService ports.TagService,
	fs ports.FileSystem,
	logger ports.Logger,
	identity domain.GitIdentity,
	gitConfig domain.GitConfig,
) *GitPlugin {
	return &GitPlugin{
		git:        git,
		tagService: tagService,
		fs:         fs,
		logger:     logger,
		identity:   identity,
		gitConfig:  gitConfig,
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

	// Stage and commit release assets before tagging so the tag points to the release commit.
	if len(p.gitConfig.Assets) > 0 {
		if err = p.git.Stage(ctx, p.gitConfig.Assets); err != nil {
			return nil, fmt.Errorf("staging release assets: %w", err)
		}
		commitMsg := renderCommitMessage(p.gitConfig.Message, tagName, rc.CurrentProject.NextVersion, rc.Notes)
		if err = p.git.Commit(ctx, commitMsg); err != nil {
			return nil, fmt.Errorf("committing release assets: %w", err)
		}
		if err = p.git.Push(ctx); err != nil {
			return nil, fmt.Errorf("pushing release branch: %w", err)
		}
	}

	headHash, err := p.git.HeadHash(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting HEAD hash: %w", err)
	}

	tagMessage := fmt.Sprintf("chore(release): %s", tagName)
	if rc.Notes != "" {
		tagMessage = rc.Notes
	}

	if err := p.git.CreateTag(ctx, tagName, headHash, tagMessage); err != nil {
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

// renderCommitMessage renders the commit message template with release data.
// Supports {{.Version}}, {{.Tag}}, and {{.Notes}} placeholders.
// Falls back to "chore(release): {tagName}" on empty template or render error.
func renderCommitMessage(tmpl, tagName string, version domain.Version, notes string) string {
	if tmpl == "" {
		return fmt.Sprintf("chore(release): %s", tagName)
	}
	data := struct {
		Version string
		Tag     string
		Notes   string
	}{
		Version: version.String(),
		Tag:     tagName,
		Notes:   notes,
	}
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return fmt.Sprintf("chore(release): %s", tagName)
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return fmt.Sprintf("chore(release): %s", tagName)
	}
	return buf.String()
}
