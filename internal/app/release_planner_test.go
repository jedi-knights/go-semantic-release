package app_test

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/jedi-knights/go-semantic-release/internal/app"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports/mocks"
)

func TestReleasePlanner_Plan_RepoMode(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)
	mockVersion := mocks.NewMockVersionCalculator(ctrl)
	mockImpact := mocks.NewMockProjectImpactAnalyzer(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	tags := []domain.Tag{{Name: "v1.0.0", Hash: "abc123"}}
	mockGit.EXPECT().ListTags(gomock.Any()).Return(tags, nil)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("main", nil)

	latestTag := &domain.Tag{Name: "v1.0.0", Version: domain.NewVersion(1, 0, 0), Hash: "abc123"}
	mockTag.EXPECT().FindLatestTag(tags, "").Return(latestTag, nil)

	commits := []domain.Commit{{Type: "feat", Description: "add feature"}}
	mockVersion.EXPECT().Calculate(
		domain.NewVersion(1, 0, 0), commits, gomock.Nil(), gomock.Any(),
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

func TestReleasePlanner_Plan_IndependentMode(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)
	mockVersion := mocks.NewMockVersionCalculator(ctrl)
	mockImpact := mocks.NewMockProjectImpactAnalyzer(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
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
		domain.NewVersion(1, 0, 0), impactMap["api"], gomock.Nil(), gomock.Any(),
	).Return(domain.NewVersion(1, 1, 0), domain.ReleaseMinor, nil)

	mockVersion.EXPECT().Calculate(
		domain.NewVersion(2, 0, 0), impactMap["worker"], gomock.Nil(), gomock.Any(),
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

func TestReleasePlanner_Plan_NoReleasable(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)
	mockVersion := mocks.NewMockVersionCalculator(ctrl)
	mockImpact := mocks.NewMockProjectImpactAnalyzer(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Debug(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	mockGit.EXPECT().ListTags(gomock.Any()).Return(nil, nil)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("main", nil)
	mockTag.EXPECT().FindLatestTag(gomock.Any(), "").Return(nil, nil)

	commits := []domain.Commit{{Type: "chore"}}
	mockVersion.EXPECT().Calculate(
		domain.ZeroVersion(), commits, gomock.Nil(), gomock.Any(),
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
