package app

import (
	"context"
	"fmt"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// ReleasePlanner builds a release plan for the repository.
type ReleasePlanner struct {
	git            ports.GitRepository
	tagService     ports.TagService
	versionCalc    ports.VersionCalculator
	impactAnalyzer ports.ProjectImpactAnalyzer
	logger         ports.Logger
	typeMapping    map[string]domain.ReleaseType
}

// NewReleasePlanner creates a release planner.
func NewReleasePlanner(
	git ports.GitRepository,
	tagService ports.TagService,
	versionCalc ports.VersionCalculator,
	impactAnalyzer ports.ProjectImpactAnalyzer,
	logger ports.Logger,
	typeMapping map[string]domain.ReleaseType,
) *ReleasePlanner {
	return &ReleasePlanner{
		git:            git,
		tagService:     tagService,
		versionCalc:    versionCalc,
		impactAnalyzer: impactAnalyzer,
		logger:         logger,
		typeMapping:    typeMapping,
	}
}

// Plan builds a release plan for the given projects and commits.
func (p *ReleasePlanner) Plan(
	ctx context.Context,
	projects []domain.Project,
	commits []domain.Commit,
	releaseMode domain.ReleaseMode,
	policy *domain.BranchPolicy,
	dryRun bool,
) (*domain.ReleasePlan, error) {
	tags, err := p.git.ListTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	plan := &domain.ReleasePlan{
		DryRun: dryRun,
		Policy: policy,
	}

	if branch, err := p.git.CurrentBranch(ctx); err == nil {
		plan.Branch = branch
	}

	if releaseMode == domain.ReleaseModeIndependent {
		return p.planIndependent(projects, commits, tags, policy, plan)
	}
	return p.planRepo(projects, commits, tags, policy, plan)
}

func (p *ReleasePlanner) planRepo(
	projects []domain.Project,
	commits []domain.Commit,
	tags []domain.Tag,
	policy *domain.BranchPolicy,
	plan *domain.ReleasePlan,
) (*domain.ReleasePlan, error) {
	latestTag, _ := p.tagService.FindLatestTag(tags, "")
	currentVersion := domain.ZeroVersion()
	sinceHash := ""

	if latestTag != nil {
		currentVersion = latestTag.Version
		sinceHash = latestTag.Hash
		_ = sinceHash // used for context, commits already provided
	}

	nextVersion, releaseType, err := p.versionCalc.Calculate(currentVersion, commits, policy, p.typeMapping)
	if err != nil {
		return nil, fmt.Errorf("calculating version: %w", err)
	}

	project := domain.Project{Name: "", Path: ".", Type: domain.ProjectTypeRoot}
	if len(projects) > 0 {
		project = projects[0]
	}

	plan.Projects = []domain.ProjectReleasePlan{{
		Project:        project,
		CurrentVersion: currentVersion,
		NextVersion:    nextVersion,
		ReleaseType:    releaseType,
		Commits:        commits,
		ShouldRelease:  releaseType.IsReleasable(),
		Reason:         buildReason(releaseType, len(commits)),
	}}

	return plan, nil
}

func (p *ReleasePlanner) planIndependent(
	projects []domain.Project,
	commits []domain.Commit,
	tags []domain.Tag,
	policy *domain.BranchPolicy,
	plan *domain.ReleasePlan,
) (*domain.ReleasePlan, error) {
	impactMap := p.impactAnalyzer.Analyze(projects, commits)

	for _, proj := range projects {
		projectCommits := impactMap[proj.Name]

		latestTag, _ := p.tagService.FindLatestTag(tags, proj.Name)
		currentVersion := domain.ZeroVersion()
		if latestTag != nil {
			currentVersion = latestTag.Version
		}

		nextVersion, releaseType, err := p.versionCalc.Calculate(currentVersion, projectCommits, policy, p.typeMapping)
		if err != nil {
			return nil, domain.NewProjectError(proj.Name, "calculate version", err)
		}

		plan.Projects = append(plan.Projects, domain.ProjectReleasePlan{
			Project:        proj,
			CurrentVersion: currentVersion,
			NextVersion:    nextVersion,
			ReleaseType:    releaseType,
			Commits:        projectCommits,
			ShouldRelease:  releaseType.IsReleasable(),
			Reason:         buildReason(releaseType, len(projectCommits)),
		})
	}

	return plan, nil
}

func buildReason(rt domain.ReleaseType, commitCount int) string {
	if !rt.IsReleasable() {
		return "no releasable changes"
	}
	return fmt.Sprintf("%d commit(s) require %s bump", commitCount, rt)
}
