package app

import (
	"context"
	"fmt"
	"strings"

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
	// Derive the tag-lookup prefix from the project's TagPrefix. Root projects
	// (TagPrefix == "") use unprefixed tags like "v1.0.0" and must look up with
	// "". Named projects with an explicit prefix (e.g. "sun-neovim/") have tags
	// like "sun-neovim/v0.1.1" and must look up with "sun-neovim".
	tagLookupPrefix := ""
	if len(projects) > 0 && projects[0].TagPrefix != "" {
		tagLookupPrefix = strings.TrimSuffix(projects[0].TagPrefix, "/")
	}
	latestTag, _ := p.tagService.FindLatestTag(tags, tagLookupPrefix)
	currentVersion := domain.ZeroVersion()

	if latestTag != nil {
		currentVersion = latestTag.Version
		// Trim commits to only those newer than the last release tag so that
		// commits already counted in a prior release are not re-analyzed.
		commits = commitsAfterHash(commits, buildCommitIndex(commits), latestTag.Hash)
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
	// Build a position index once for all per-project filtering below.
	// commits is ordered newest-first (git log default), so lower index = newer.
	commitIndex := buildCommitIndex(commits)

	impactMap := p.impactAnalyzer.Analyze(projects, commits)

	for _, proj := range projects {
		projectCommits := impactMap[proj.Name]

		latestTag, _ := p.tagService.FindLatestTag(tags, proj.Name)
		currentVersion := domain.ZeroVersion()
		if latestTag != nil {
			currentVersion = latestTag.Version
			// Trim to only commits newer than the last release tag so that
			// commits already counted in a prior release are not re-analyzed.
			projectCommits = commitsAfterHash(projectCommits, commitIndex, latestTag.Hash)
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

// buildCommitIndex constructs a position map for a newest-first commit slice.
// Lower index means more recent; this is the natural order from git log.
func buildCommitIndex(commits []domain.Commit) map[string]int {
	idx := make(map[string]int, len(commits))
	for i := range commits {
		idx[commits[i].Hash] = i
	}
	return idx
}

// commitsAfterHash returns the subset of commits that are newer than the commit
// identified by sinceHash, as determined by position in the globally-ordered
// newest-first index. Commits at a lower index than sinceHash's position are
// newer; commits at the same or higher index were already included in the
// release that created sinceHash.
//
// If sinceHash is empty or not present in the index (first release or the tag
// commit is outside the fetched window), all commits are returned unchanged so
// the first-release path is handled correctly.
func commitsAfterHash(commits []domain.Commit, index map[string]int, sinceHash string) []domain.Commit {
	if sinceHash == "" {
		return commits
	}
	cutoff, ok := index[sinceHash]
	if !ok {
		// Tag commit not in the fetched window — treat every commit as new.
		return commits
	}
	result := make([]domain.Commit, 0, cutoff)
	for i := range commits {
		if pos, exists := index[commits[i].Hash]; exists && pos < cutoff {
			result = append(result, commits[i])
		}
	}
	return result
}
