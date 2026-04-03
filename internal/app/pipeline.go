package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Pipeline orchestrates the semantic release lifecycle.
type Pipeline struct {
	plugins []ports.Plugin
	logger  ports.Logger
}

// NewPipeline creates a new lifecycle pipeline with the given plugins.
func NewPipeline(plugins []ports.Plugin, logger ports.Logger) *Pipeline {
	return &Pipeline{plugins: plugins, logger: logger}
}

// Execute runs all lifecycle steps in order against the release context.
func (p *Pipeline) Execute(ctx context.Context, rc *domain.ReleaseContext) error {
	if err := p.runVerifyConditions(ctx, rc); err != nil {
		return p.handleFailure(ctx, rc, err)
	}

	releaseType, err := p.runAnalyzeCommits(ctx, rc)
	if err != nil {
		return p.handleFailure(ctx, rc, err)
	}

	if !releaseType.IsReleasable() {
		p.logger.Info("no releasable changes found")
		return nil
	}

	if verifyErr := p.runVerifyRelease(ctx, rc); verifyErr != nil {
		return p.handleFailure(ctx, rc, verifyErr)
	}

	notes, err := p.runGenerateNotes(ctx, rc)
	if err != nil {
		return p.handleFailure(ctx, rc, err)
	}
	rc.Notes = notes

	if rc.DryRun {
		p.logger.Info("dry run complete, skipping prepare/publish/addChannel/success steps")
		return nil
	}

	if err := p.runPrepare(ctx, rc); err != nil {
		return p.handleFailure(ctx, rc, err)
	}

	if err := p.runPublish(ctx, rc); err != nil {
		return p.handleFailure(ctx, rc, err)
	}

	if err := p.runAddChannel(ctx, rc); err != nil {
		p.logger.Warn("addChannel failed", "error", err)
		// Non-fatal — continue to success notification.
	}

	return p.runSuccess(ctx, rc)
}

func (p *Pipeline) runVerifyConditions(ctx context.Context, rc *domain.ReleaseContext) error {
	for _, plugin := range p.plugins {
		if vp, ok := plugin.(ports.VerifyConditionsPlugin); ok {
			p.logger.Debug("running verifyConditions", "plugin", plugin.Name())
			if err := vp.VerifyConditions(ctx, rc); err != nil {
				return domain.NewReleaseError("verifyConditions:"+plugin.Name(), err)
			}
		}
	}
	return nil
}

func (p *Pipeline) runAnalyzeCommits(ctx context.Context, rc *domain.ReleaseContext) (domain.ReleaseType, error) {
	highest := domain.ReleaseNone
	for _, plugin := range p.plugins {
		if ap, ok := plugin.(ports.AnalyzeCommitsPlugin); ok {
			p.logger.Debug("running analyzeCommits", "plugin", plugin.Name())
			rt, err := ap.AnalyzeCommits(ctx, rc)
			if err != nil {
				return domain.ReleaseNone, domain.NewReleaseError("analyzeCommits:"+plugin.Name(), err)
			}
			highest = highest.Higher(rt)
		}
	}
	return highest, nil
}

func (p *Pipeline) runVerifyRelease(ctx context.Context, rc *domain.ReleaseContext) error {
	for _, plugin := range p.plugins {
		if vp, ok := plugin.(ports.VerifyReleasePlugin); ok {
			p.logger.Debug("running verifyRelease", "plugin", plugin.Name())
			if err := vp.VerifyRelease(ctx, rc); err != nil {
				return domain.NewReleaseError("verifyRelease:"+plugin.Name(), err)
			}
		}
	}
	return nil
}

func (p *Pipeline) runGenerateNotes(ctx context.Context, rc *domain.ReleaseContext) (string, error) {
	var parts []string
	for _, plugin := range p.plugins {
		if gp, ok := plugin.(ports.GenerateNotesPlugin); ok {
			p.logger.Debug("running generateNotes", "plugin", plugin.Name())
			notes, err := gp.GenerateNotes(ctx, rc)
			if err != nil {
				return "", domain.NewReleaseError("generateNotes:"+plugin.Name(), err)
			}
			if notes != "" {
				parts = append(parts, notes)
			}
		}
	}
	return strings.Join(parts, "\n\n"), nil
}

func (p *Pipeline) runPrepare(ctx context.Context, rc *domain.ReleaseContext) error {
	for _, plugin := range p.plugins {
		if pp, ok := plugin.(ports.PreparePlugin); ok {
			p.logger.Debug("running prepare", "plugin", plugin.Name())
			if err := pp.Prepare(ctx, rc); err != nil {
				return domain.NewReleaseError("prepare:"+plugin.Name(), err)
			}
		}
	}
	return nil
}

func (p *Pipeline) runPublish(ctx context.Context, rc *domain.ReleaseContext) error {
	for _, plugin := range p.plugins {
		if pp, ok := plugin.(ports.PublishPlugin); ok {
			p.logger.Debug("running publish", "plugin", plugin.Name())
			result, err := pp.Publish(ctx, rc)
			if err != nil {
				return domain.NewReleaseError("publish:"+plugin.Name(), err)
			}
			if result != nil && rc.Result != nil {
				rc.Result.Projects = append(rc.Result.Projects, *result)
			}
		}
	}
	return nil
}

func (p *Pipeline) runAddChannel(ctx context.Context, rc *domain.ReleaseContext) error {
	for _, plugin := range p.plugins {
		if ap, ok := plugin.(ports.AddChannelPlugin); ok {
			p.logger.Debug("running addChannel", "plugin", plugin.Name())
			if err := ap.AddChannel(ctx, rc); err != nil {
				return fmt.Errorf("addChannel:%s: %w", plugin.Name(), err)
			}
		}
	}
	return nil
}

func (p *Pipeline) runSuccess(ctx context.Context, rc *domain.ReleaseContext) error {
	for _, plugin := range p.plugins {
		if sp, ok := plugin.(ports.SuccessPlugin); ok {
			p.logger.Debug("running success", "plugin", plugin.Name())
			if err := sp.Success(ctx, rc); err != nil {
				p.logger.Warn("success notification failed", "plugin", plugin.Name(), "error", err)
				// Non-fatal — don't fail the release for notification failures.
			}
		}
	}
	return nil
}

func (p *Pipeline) handleFailure(ctx context.Context, rc *domain.ReleaseContext, releaseErr error) error {
	rc.Error = releaseErr
	for _, plugin := range p.plugins {
		if fp, ok := plugin.(ports.FailPlugin); ok {
			p.logger.Debug("running fail", "plugin", plugin.Name())
			if err := fp.Fail(ctx, rc); err != nil {
				p.logger.Warn("fail notification failed", "plugin", plugin.Name(), "error", err)
			}
		}
	}
	return releaseErr
}
