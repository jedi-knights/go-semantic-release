package app_test

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/jedi-knights/go-semantic-release/internal/app"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports/mocks"
)

// fakeTagService is a test double for ports.TagService. It decouples counter
// tests from the number and order of ParseTag calls, verifying output
// (correct counter value) rather than internal call sequence.
type fakeTagService struct {
	parsed map[string]fakeParsed  // tag name → parse result
	latest map[string]*domain.Tag // project → latest tag
}

type fakeParsed struct {
	project string
	version domain.Version
}

func (f *fakeTagService) ParseTag(name string) (string, domain.Version, error) {
	if p, ok := f.parsed[name]; ok {
		return p.project, p.version, nil
	}
	return "", domain.Version{}, fmt.Errorf("unknown tag %q", name)
}

func (f *fakeTagService) FindLatestTag(_ []domain.Tag, project string) (*domain.Tag, error) {
	if t, ok := f.latest[project]; ok {
		return t, nil
	}
	return nil, nil
}

func (f *fakeTagService) FormatTag(_ string, _ domain.Version) (string, error) {
	return "", fmt.Errorf("fakeTagService.FormatTag not implemented")
}

func TestReleasePlanner_Plan_RepoMode(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)
	mockVersion := mocks.NewMockVersionCalculator(ctrl)
	mockImpact := mocks.NewMockProjectImpactAnalyzer(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	tags := []domain.Tag{{Name: "v1.0.0", Hash: "abc123"}}
	mockGit.EXPECT().ListTags(gomock.Any()).Return(tags, nil)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("main", nil)

	latestTag := &domain.Tag{Name: "v1.0.0", Version: domain.NewVersion(1, 0, 0), Hash: "abc123"}
	mockTag.EXPECT().FindLatestTag(tags, "").Return(latestTag, nil)

	commits := []domain.Commit{{Type: "feat", Description: "add feature"}}
	mockVersion.EXPECT().Calculate(
		domain.NewVersion(1, 0, 0), commits, gomock.Nil(), gomock.Any(), 0,
	).Return(domain.NewVersion(1, 1, 0), domain.ReleaseMinor, nil)

	planner := app.NewReleasePlanner(mockGit, mockTag, mockVersion, mockImpact, mockLogger, domain.DefaultCommitTypeMapping())

	projects := []domain.Project{{Name: "root", Path: "."}}
	plan, err := planner.Plan(context.Background(), projects, commits, domain.ReleaseModeRepo, nil, false)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if !plan.HasReleasableProjects() {
		t.Error("expected releasable projects")
	}
	if len(plan.Projects) != 1 {
		t.Fatalf("expected 1 project plan, got %d", len(plan.Projects))
	}
	if !plan.Projects[0].NextVersion.Equal(domain.NewVersion(1, 1, 0)) {
		t.Errorf("next version = %v, want 1.1.0", plan.Projects[0].NextVersion)
	}
}

// TestReleasePlanner_Plan_RepoMode_WithTagPrefix covers the case where a single
// project in repo mode uses prefixed tags (e.g. "sun-neovim/v0.1.1"). The
// planner must derive the lookup prefix from TagPrefix, not use "" (which would
// miss the existing tag and recompute from scratch).
func TestReleasePlanner_Plan_RepoMode_WithTagPrefix(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)
	mockVersion := mocks.NewMockVersionCalculator(ctrl)
	mockImpact := mocks.NewMockProjectImpactAnalyzer(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	tags := []domain.Tag{{Name: "sun-neovim/v0.1.0", Hash: "abc123"}}
	mockGit.EXPECT().ListTags(gomock.Any()).Return(tags, nil)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("main", nil)

	latestTag := &domain.Tag{Name: "sun-neovim/v0.1.0", Version: domain.NewVersion(0, 1, 0), Hash: "abc123"}
	// Planner must call FindLatestTag with "sun-neovim" (derived from TagPrefix
	// "sun-neovim/"), not "" (which would miss the existing tag entirely).
	mockTag.EXPECT().FindLatestTag(tags, "sun-neovim").Return(latestTag, nil)

	commits := []domain.Commit{{Type: "fix", Description: "fix crash"}}
	mockVersion.EXPECT().Calculate(
		domain.NewVersion(0, 1, 0), commits, gomock.Nil(), gomock.Any(), 0,
	).Return(domain.NewVersion(0, 1, 1), domain.ReleasePatch, nil)

	planner := app.NewReleasePlanner(mockGit, mockTag, mockVersion, mockImpact, mockLogger, domain.DefaultCommitTypeMapping())

	projects := []domain.Project{{Name: "sun-neovim", Path: ".", TagPrefix: "sun-neovim/"}}
	plan, err := planner.Plan(context.Background(), projects, commits, domain.ReleaseModeRepo, nil, false)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if !plan.HasReleasableProjects() {
		t.Error("expected releasable projects")
	}
	if len(plan.Projects) != 1 {
		t.Fatalf("expected 1 project plan, got %d", len(plan.Projects))
	}
	if !plan.Projects[0].NextVersion.Equal(domain.NewVersion(0, 1, 1)) {
		t.Errorf("next version = %v, want 0.1.1", plan.Projects[0].NextVersion)
	}
	if !plan.Projects[0].CurrentVersion.Equal(domain.NewVersion(0, 1, 0)) {
		t.Errorf("current version = %v, want 0.1.0 — tag prefix lookup failed to find baseline", plan.Projects[0].CurrentVersion)
	}
}

func TestReleasePlanner_Plan_IndependentMode(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)
	mockVersion := mocks.NewMockVersionCalculator(ctrl)
	mockImpact := mocks.NewMockProjectImpactAnalyzer(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	tags := []domain.Tag{
		{Name: "api/v1.0.0", Hash: "aaa"},
		{Name: "worker/v2.0.0", Hash: "bbb"},
	}
	mockGit.EXPECT().ListTags(gomock.Any()).Return(tags, nil)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("main", nil)

	apiTag := &domain.Tag{Name: "api/v1.0.0", Version: domain.NewVersion(1, 0, 0)}
	workerTag := &domain.Tag{Name: "worker/v2.0.0", Version: domain.NewVersion(2, 0, 0)}
	mockTag.EXPECT().FindLatestTag(tags, "api").Return(apiTag, nil)
	mockTag.EXPECT().FindLatestTag(tags, "worker").Return(workerTag, nil)

	projects := []domain.Project{
		{Name: "api", Path: "services/api"},
		{Name: "worker", Path: "services/worker"},
	}

	commits := []domain.Commit{
		{Hash: "c1", Type: "feat", FilesChanged: []string{"services/api/main.go"}},
		{Hash: "c2", Type: "fix", FilesChanged: []string{"services/worker/main.go"}},
	}

	impactMap := map[string][]domain.Commit{
		"api":    {commits[0]},
		"worker": {commits[1]},
	}
	mockImpact.EXPECT().Analyze(projects, commits).Return(impactMap)

	mockVersion.EXPECT().Calculate(
		domain.NewVersion(1, 0, 0), impactMap["api"], gomock.Nil(), gomock.Any(), 0,
	).Return(domain.NewVersion(1, 1, 0), domain.ReleaseMinor, nil)

	mockVersion.EXPECT().Calculate(
		domain.NewVersion(2, 0, 0), impactMap["worker"], gomock.Nil(), gomock.Any(), 0,
	).Return(domain.NewVersion(2, 0, 1), domain.ReleasePatch, nil)

	planner := app.NewReleasePlanner(mockGit, mockTag, mockVersion, mockImpact, mockLogger, domain.DefaultCommitTypeMapping())

	plan, err := planner.Plan(context.Background(), projects, commits, domain.ReleaseModeIndependent, nil, false)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if len(plan.Projects) != 2 {
		t.Fatalf("expected 2 project plans, got %d", len(plan.Projects))
	}

	if !plan.Projects[0].ShouldRelease {
		t.Error("api should release")
	}
	if !plan.Projects[1].ShouldRelease {
		t.Error("worker should release")
	}
}

// TestReleasePlanner_Plan_IndependentMode_DoesNotReanalyzeReleasedCommits verifies
// that planIndependent only considers commits newer than each project's last release
// tag. This guards against the regression where every push re-analyzed the full
// commit history and produced unnecessary version bumps.
func TestReleasePlanner_Plan_IndependentMode_DoesNotReanalyzeReleasedCommits(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)
	mockVersion := mocks.NewMockVersionCalculator(ctrl)
	mockImpact := mocks.NewMockProjectImpactAnalyzer(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	// Simulate: three commits in history (newest first).
	//   newCommit  – docs: change README (non-releasable, touches no project path)
	//   tagCommit  – the exact commit that was HEAD when api/v1.0.0 was created
	//   oldFeat    – feat: initial api work (already counted in v1.0.0)
	newCommit := domain.Commit{Hash: "new-sha", Type: "docs", FilesChanged: []string{"README.md"}}
	tagCommit := domain.Commit{Hash: "tag-sha", Type: "chore", FilesChanged: []string{}}
	oldFeat := domain.Commit{Hash: "old-sha", Type: "feat", FilesChanged: []string{"services/api/main.go"}}
	commits := []domain.Commit{newCommit, tagCommit, oldFeat}

	tags := []domain.Tag{{Name: "api/v1.0.0", Hash: "tag-sha"}}
	mockGit.EXPECT().ListTags(gomock.Any()).Return(tags, nil)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("main", nil)

	// FindLatestTag returns the tag whose commit hash is "tag-sha" (v1.0.0).
	apiTag := &domain.Tag{Name: "api/v1.0.0", Version: domain.NewVersion(1, 0, 0), Hash: "tag-sha"}
	mockTag.EXPECT().FindLatestTag(tags, "api").Return(apiTag, nil)

	projects := []domain.Project{{Name: "api", Path: "services/api"}}

	// The impact analyzer sees all three commits but only oldFeat touches services/api.
	// newCommit and tagCommit have no api-scoped files.
	impactMap := map[string][]domain.Commit{
		"api": {oldFeat}, // path-filtered: only oldFeat touches services/api
	}
	mockImpact.EXPECT().Analyze(projects, commits).Return(impactMap)

	// After commitsAfterHash filters oldFeat (at index 2, which is >= cutoff index 1
	// for tag-sha), the version calculator must receive an empty commit slice.
	// An empty slice means no releasable changes → no bump.
	mockVersion.EXPECT().Calculate(
		domain.NewVersion(1, 0, 0), []domain.Commit{}, gomock.Nil(), gomock.Any(), 0,
	).Return(domain.NewVersion(1, 0, 0), domain.ReleaseNone, nil)

	planner := app.NewReleasePlanner(mockGit, mockTag, mockVersion, mockImpact, mockLogger, domain.DefaultCommitTypeMapping())

	plan, err := planner.Plan(context.Background(), projects, commits, domain.ReleaseModeIndependent, nil, false)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if plan.HasReleasableProjects() {
		t.Error("expected no release: all releasable commits are older than the last tag")
	}
	if len(plan.Projects[0].Commits) != 0 {
		t.Errorf("expected 0 commits after filtering, got %d", len(plan.Projects[0].Commits))
	}
}

func TestReleasePlanner_Plan_NoReleasable(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)
	mockVersion := mocks.NewMockVersionCalculator(ctrl)
	mockImpact := mocks.NewMockProjectImpactAnalyzer(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	mockGit.EXPECT().ListTags(gomock.Any()).Return(nil, nil)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("main", nil)
	mockTag.EXPECT().FindLatestTag(gomock.Any(), "").Return(nil, nil)

	commits := []domain.Commit{{Type: "chore"}}
	mockVersion.EXPECT().Calculate(
		domain.ZeroVersion(), commits, gomock.Nil(), gomock.Any(), 0,
	).Return(domain.ZeroVersion(), domain.ReleaseNone, nil)

	planner := app.NewReleasePlanner(mockGit, mockTag, mockVersion, mockImpact, mockLogger, domain.DefaultCommitTypeMapping())

	plan, err := planner.Plan(context.Background(), nil, commits, domain.ReleaseModeRepo, nil, true)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if plan.HasReleasableProjects() {
		t.Error("expected no releasable projects")
	}
}

// TestReleasePlanner_Plan_Prerelease_CounterStartsAtZero verifies that when no
// prerelease tags exist for the next base version, the counter passed to
// VersionCalculator is 0 — producing the first RC tag: v1.1.0-rc.0.
func TestReleasePlanner_Plan_Prerelease_CounterStartsAtZero(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockVersion := mocks.NewMockVersionCalculator(ctrl)
	mockImpact := mocks.NewMockProjectImpactAnalyzer(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	// Only a stable tag exists — no prerelease tags yet.
	tags := []domain.Tag{{Name: "v1.0.0", Hash: "abc123"}}
	mockGit.EXPECT().ListTags(gomock.Any()).Return(tags, nil)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("rc", nil)

	fakeTag := &fakeTagService{
		parsed: map[string]fakeParsed{
			"v1.0.0": {project: "", version: domain.NewVersion(1, 0, 0)},
		},
		latest: map[string]*domain.Tag{
			"": {Name: "v1.0.0", Version: domain.NewVersion(1, 0, 0), Hash: "abc123"},
		},
	}

	policy := &domain.BranchPolicy{Name: "rc", Prerelease: true, Channel: "rc"}
	commits := []domain.Commit{{Hash: "c1", Type: "feat", Description: "new feature"}}

	// feat on v1.0.0 → base v1.1.0; no rc tags for v1.1.0 → counter=0.
	mockVersion.EXPECT().Calculate(
		domain.NewVersion(1, 0, 0), commits, policy, gomock.Any(), 0,
	).Return(domain.Version{Major: 1, Minor: 1, Patch: 0, Prerelease: "rc.0"}, domain.ReleaseMinor, nil)

	planner := app.NewReleasePlanner(mockGit, fakeTag, mockVersion, mockImpact, mockLogger, domain.DefaultCommitTypeMapping())

	projects := []domain.Project{{Name: "root", Path: "."}}
	plan, err := planner.Plan(context.Background(), projects, commits, domain.ReleaseModeRepo, policy, false)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if !plan.HasReleasableProjects() {
		t.Error("expected releasable project")
	}
	want := domain.Version{Major: 1, Minor: 1, Patch: 0, Prerelease: "rc.0"}
	if !plan.Projects[0].NextVersion.Equal(want) {
		t.Errorf("next version = %v, want %v", plan.Projects[0].NextVersion, want)
	}
}

// TestReleasePlanner_Plan_Prerelease_CounterIncrementsWithExistingTags verifies
// that when prerelease tags already exist for the next base version, the counter
// equals the number of existing RC tags — producing the next RC in sequence.
func TestReleasePlanner_Plan_Prerelease_CounterIncrementsWithExistingTags(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockVersion := mocks.NewMockVersionCalculator(ctrl)
	mockImpact := mocks.NewMockProjectImpactAnalyzer(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	// Two existing RC tags for v1.1.0 already exist.
	tags := []domain.Tag{
		{Name: "v1.0.0", Hash: "abc"},
		{Name: "v1.1.0-rc.0", Hash: "def"},
		{Name: "v1.1.0-rc.1", Hash: "ghi"},
	}
	mockGit.EXPECT().ListTags(gomock.Any()).Return(tags, nil)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("rc", nil)

	// FindLatestTag returns the stable baseline (ignores prerelease in ordering).
	fakeTag := &fakeTagService{
		parsed: map[string]fakeParsed{
			"v1.0.0":      {project: "", version: domain.NewVersion(1, 0, 0)},
			"v1.1.0-rc.0": {project: "", version: domain.Version{Major: 1, Minor: 1, Patch: 0, Prerelease: "rc.0"}},
			"v1.1.0-rc.1": {project: "", version: domain.Version{Major: 1, Minor: 1, Patch: 0, Prerelease: "rc.1"}},
		},
		latest: map[string]*domain.Tag{
			"": {Name: "v1.0.0", Version: domain.NewVersion(1, 0, 0), Hash: "abc"},
		},
	}

	policy := &domain.BranchPolicy{Name: "rc", Prerelease: true, Channel: "rc"}
	commits := []domain.Commit{{Hash: "c1", Type: "feat", Description: "another fix"}}

	// feat on v1.0.0 → base v1.1.0; 2 existing rc tags → counter=2.
	mockVersion.EXPECT().Calculate(
		domain.NewVersion(1, 0, 0), commits, policy, gomock.Any(), 2,
	).Return(domain.Version{Major: 1, Minor: 1, Patch: 0, Prerelease: "rc.2"}, domain.ReleaseMinor, nil)

	planner := app.NewReleasePlanner(mockGit, fakeTag, mockVersion, mockImpact, mockLogger, domain.DefaultCommitTypeMapping())

	projects := []domain.Project{{Name: "root", Path: "."}}
	plan, err := planner.Plan(context.Background(), projects, commits, domain.ReleaseModeRepo, policy, false)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	want := domain.Version{Major: 1, Minor: 1, Patch: 0, Prerelease: "rc.2"}
	if !plan.Projects[0].NextVersion.Equal(want) {
		t.Errorf("next version = %v, want %v", plan.Projects[0].NextVersion, want)
	}
}

// TestReleasePlanner_Plan_Prerelease_CounterResetsOnBaseVersionChange verifies
// that when a higher-impact commit changes the base version, the counter resets
// to 0 because no prerelease tags exist for the new base version yet.
func TestReleasePlanner_Plan_Prerelease_CounterResetsOnBaseVersionChange(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockVersion := mocks.NewMockVersionCalculator(ctrl)
	mockImpact := mocks.NewMockProjectImpactAnalyzer(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	// Existing RC tags are for v1.1.0. Incoming commit is a breaking change
	// → base becomes v2.0.0 → counter resets to 0.
	tags := []domain.Tag{
		{Name: "v1.0.0", Hash: "abc"},
		{Name: "v1.1.0-rc.0", Hash: "def"},
	}
	mockGit.EXPECT().ListTags(gomock.Any()).Return(tags, nil)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("rc", nil)

	fakeTag := &fakeTagService{
		parsed: map[string]fakeParsed{
			"v1.0.0":      {project: "", version: domain.NewVersion(1, 0, 0)},
			"v1.1.0-rc.0": {project: "", version: domain.Version{Major: 1, Minor: 1, Patch: 0, Prerelease: "rc.0"}},
		},
		latest: map[string]*domain.Tag{
			"": {Name: "v1.0.0", Version: domain.NewVersion(1, 0, 0), Hash: "abc"},
		},
	}

	policy := &domain.BranchPolicy{Name: "rc", Prerelease: true, Channel: "rc"}
	commits := []domain.Commit{{Hash: "c1", Type: "feat", IsBreakingChange: true, Description: "breaking change"}}

	// breaking change on v1.0.0 → base v2.0.0; no rc tags for v2.0.0 → counter=0.
	mockVersion.EXPECT().Calculate(
		domain.NewVersion(1, 0, 0), commits, policy, gomock.Any(), 0,
	).Return(domain.Version{Major: 2, Minor: 0, Patch: 0, Prerelease: "rc.0"}, domain.ReleaseMajor, nil)

	planner := app.NewReleasePlanner(mockGit, fakeTag, mockVersion, mockImpact, mockLogger, domain.DefaultCommitTypeMapping())

	projects := []domain.Project{{Name: "root", Path: "."}}
	plan, err := planner.Plan(context.Background(), projects, commits, domain.ReleaseModeRepo, policy, false)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	want := domain.Version{Major: 2, Minor: 0, Patch: 0, Prerelease: "rc.0"}
	if !plan.Projects[0].NextVersion.Equal(want) {
		t.Errorf("next version = %v, want %v", plan.Projects[0].NextVersion, want)
	}
}

// TestReleasePlanner_Plan_Prerelease_MaintenanceBranchSkipsCounter verifies
// that a branch which is both prerelease and maintenance does not trigger the
// counter lookup. The counter is undefined for maintenance branches because
// constrainMaintenanceBump may change the base version after nextBaseVersion
// computes it, causing the counter to target the wrong base version.
func TestReleasePlanner_Plan_Prerelease_MaintenanceBranchSkipsCounter(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)
	mockVersion := mocks.NewMockVersionCalculator(ctrl)
	mockImpact := mocks.NewMockProjectImpactAnalyzer(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	tags := []domain.Tag{{Name: "v1.0.0", Hash: "abc"}}
	mockGit.EXPECT().ListTags(gomock.Any()).Return(tags, nil)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("1.x", nil)

	latestTag := &domain.Tag{Name: "v1.0.0", Version: domain.NewVersion(1, 0, 0), Hash: "abc"}
	mockTag.EXPECT().FindLatestTag(tags, "").Return(latestTag, nil)
	// ParseTag must NOT be called — maintenance branches skip the counter path.

	// Policy is both prerelease and maintenance.
	policy := &domain.BranchPolicy{
		Name:       "1.x",
		Prerelease: true,
		Channel:    "rc",
		Range:      "1.x",
		Type:       domain.BranchTypeMaintenance,
	}
	commits := []domain.Commit{{Hash: "c1", Type: "fix", Description: "patch fix"}}

	// Counter must be 0 — no lookup performed.
	mockVersion.EXPECT().Calculate(
		domain.NewVersion(1, 0, 0), commits, policy, gomock.Any(), 0,
	).Return(domain.Version{Major: 1, Minor: 0, Patch: 1, Prerelease: "rc.0"}, domain.ReleasePatch, nil)

	planner := app.NewReleasePlanner(mockGit, mockTag, mockVersion, mockImpact, mockLogger, domain.DefaultCommitTypeMapping())

	projects := []domain.Project{{Name: "root", Path: "."}}
	plan, err := planner.Plan(context.Background(), projects, commits, domain.ReleaseModeRepo, policy, false)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if !plan.HasReleasableProjects() {
		t.Error("expected releasable project")
	}
}

// TestReleasePlanner_Plan_RepoMode_FindLatestTagError verifies that an error
// from FindLatestTag is propagated and Plan returns an error rather than
// silently treating a failed lookup as "no prior tag".
func TestReleasePlanner_Plan_RepoMode_FindLatestTagError(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)
	mockVersion := mocks.NewMockVersionCalculator(ctrl)
	mockImpact := mocks.NewMockProjectImpactAnalyzer(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	tags := []domain.Tag{{Name: "v1.0.0", Hash: "abc"}}
	mockGit.EXPECT().ListTags(gomock.Any()).Return(tags, nil)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("main", nil)

	mockTag.EXPECT().FindLatestTag(tags, "").Return(nil, fmt.Errorf("tag service unavailable"))
	// Calculate must NOT be called when FindLatestTag errors.

	planner := app.NewReleasePlanner(mockGit, mockTag, mockVersion, mockImpact, mockLogger, domain.DefaultCommitTypeMapping())

	projects := []domain.Project{{Name: "root", Path: "."}}
	commits := []domain.Commit{{Type: "feat"}}
	_, err := planner.Plan(context.Background(), projects, commits, domain.ReleaseModeRepo, nil, false)
	if err == nil {
		t.Fatal("expected error from FindLatestTag, got nil")
	}
}

// TestReleasePlanner_Plan_IndependentMode_FindLatestTagError verifies that a
// FindLatestTag error in independent mode is propagated rather than silently
// treated as "no prior tag", which would cause versioning to restart from zero.
func TestReleasePlanner_Plan_IndependentMode_FindLatestTagError(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)
	mockVersion := mocks.NewMockVersionCalculator(ctrl)
	mockImpact := mocks.NewMockProjectImpactAnalyzer(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	tags := []domain.Tag{{Name: "api/v1.0.0", Hash: "abc"}}
	mockGit.EXPECT().ListTags(gomock.Any()).Return(tags, nil)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("main", nil)

	projects := []domain.Project{{Name: "api", Path: "services/api"}}
	commits := []domain.Commit{{Hash: "c1", Type: "feat", FilesChanged: []string{"services/api/main.go"}}}

	impactMap := map[string][]domain.Commit{"api": {commits[0]}}
	mockImpact.EXPECT().Analyze(projects, commits).Return(impactMap)

	mockTag.EXPECT().FindLatestTag(tags, "api").Return(nil, fmt.Errorf("tag service unavailable"))
	// Calculate must NOT be called when FindLatestTag errors.

	planner := app.NewReleasePlanner(mockGit, mockTag, mockVersion, mockImpact, mockLogger, domain.DefaultCommitTypeMapping())

	_, err := planner.Plan(context.Background(), projects, commits, domain.ReleaseModeIndependent, nil, false)
	if err == nil {
		t.Fatal("expected error from FindLatestTag in independent mode, got nil")
	}
}
