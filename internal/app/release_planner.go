package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// nextBaseVersion computes the bumped Major.Minor.Patch for the given commits
// without applying any prerelease suffix. Used by the planner to determine
// which base version to search for when counting existing prerelease tags.
func nextBaseVersion(current domain.Version, commits []domain.Commit, typeMapping map[string]domain.ReleaseType) domain.Version {
	bump := aggregateBump(commits, typeMapping)
	if !bump.IsReleasable() {
		return current
	}
	return current.Bump(bump)
}

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
	} else {
		p.logger.Warn("could not determine current branch", "error", err)
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
	latestTag, err := p.tagService.FindLatestTag(tags, tagLookupPrefix)
	if err != nil {
		return nil, fmt.Errorf("finding latest tag: %w", err)
	}
	currentVersion := domain.ZeroVersion()

	if latestTag != nil {
		currentVersion = latestTag.Version
		// Trim commits to only those newer than the last release tag so that
		// commits already counted in a prior release are not re-analyzed.
		commits = commitsAfterHash(commits, buildCommitIndex(commits), latestTag.Hash)
	}

	counter := 0
	if policy != nil && policy.IsPrerelease() && !policy.IsMaintenance() {
		base := nextBaseVersion(currentVersion, commits, p.typeMapping)
		counter = p.countPrereleaseTags(tags, tagLookupPrefix, base, policy.Channel)
	}

	nextVersion, releaseType, err := p.versionCalc.Calculate(currentVersion, commits, policy, p.typeMapping, counter)
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

	plan.Projects = make([]domain.ProjectReleasePlan, 0, len(projects))

	for _, proj := range projects {
		projectCommits := impactMap[proj.Name]

		latestTag, err := p.tagService.FindLatestTag(tags, proj.Name)
		if err != nil {
			return nil, domain.NewProjectError(proj.Name, "find latest tag", err)
		}
		currentVersion := domain.ZeroVersion()
		if latestTag != nil {
			currentVersion = latestTag.Version
			// Trim to only commits newer than the last release tag so that
			// commits already counted in a prior release are not re-analyzed.
			projectCommits = commitsAfterHash(projectCommits, commitIndex, latestTag.Hash)
		}

		counter := 0
		if policy != nil && policy.IsPrerelease() && !policy.IsMaintenance() {
			base := nextBaseVersion(currentVersion, projectCommits, p.typeMapping)
			counter = p.countPrereleaseTags(tags, proj.Name, base, policy.Channel)
		}

		nextVersion, releaseType, err := p.versionCalc.Calculate(currentVersion, projectCommits, policy, p.typeMapping, counter)
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

// countPrereleaseTags counts existing prerelease tags for a specific project,
// base version (Major.Minor.Patch), and channel. The result is used as the
// counter N in the {channel}.{N} prerelease suffix so each RC tag in a cycle
// is unique and increments automatically.
//
// Matching rule: the prerelease field must begin with "{channel}." — the dot
// boundary prevents a channel named "rc" from matching a hand-crafted tag
// whose prerelease starts with "rca" or similar. Tags whose prerelease contains
// additional dot-separated segments (e.g. "rc.1.2") are accepted as valid
// counter tags; this is intentional to remain compatible with any tooling that
// writes counters in the legacy format.
func (p *ReleasePlanner) countPrereleaseTags(tags []domain.Tag, project string, base domain.Version, channel string) int {
	prefix := channel + "."
	count := 0
	for _, tag := range tags {
		proj, ver, err := p.tagService.ParseTag(tag.Name)
		if err != nil {
			continue
		}
		if proj != project {
			continue
		}
		if ver.Major != base.Major || ver.Minor != base.Minor || ver.Patch != base.Patch {
			continue
		}
		if strings.HasPrefix(ver.Prerelease, prefix) {
			count++
		}
	}
	return count
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
