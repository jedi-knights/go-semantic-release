package app_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/jedi-knights/go-semantic-release/internal/app"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
	"github.com/jedi-knights/go-semantic-release/internal/ports/mocks"
)

func TestReleaseExecutor_Execute_DryRun(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)
	mockChangelog := mocks.NewMockChangelogGenerator(ctrl)
	mockPublisher := mocks.NewMockReleasePublisher(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	sections := domain.DefaultChangelogSections()

	mockChangelog.EXPECT().Generate(
		domain.NewVersion(1, 1, 0), "", gomock.Any(), sections,
	).Return("## 1.1.0\n\n### Features\n- add feature", nil)

	mockTag.EXPECT().FormatTag("", domain.NewVersion(1, 1, 0)).Return("v1.1.0", nil)

	// In dry-run mode, no git operations or publishing should happen.

	executor := app.MustNewReleaseExecutor(mockGit, mockTag, mockChangelog, mockPublisher, mockLogger, sections)

	plan := &domain.ReleasePlan{
		DryRun: true,
		Projects: []domain.ProjectReleasePlan{{
			Project:        domain.Project{Name: "", Path: "."},
			CurrentVersion: domain.NewVersion(1, 0, 0),
			NextVersion:    domain.NewVersion(1, 1, 0),
			ReleaseType:    domain.ReleaseMinor,
			Commits:        []domain.Commit{{Type: "feat", Description: "add feature"}},
			ShouldRelease:  true,
		}},
	}

	result, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !result.DryRun {
		t.Error("result should be dry run")
	}
	if len(result.Projects) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Projects))
	}
	if !result.Projects[0].Skipped {
		t.Error("project should be skipped in dry run")
	}
	if result.Projects[0].TagName != "v1.1.0" {
		t.Errorf("tag name = %q, want %q", result.Projects[0].TagName, "v1.1.0")
	}
}

func TestReleaseExecutor_Execute_FullRelease(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)
	mockChangelog := mocks.NewMockChangelogGenerator(ctrl)
	mockPublisher := mocks.NewMockReleasePublisher(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()

	sections := domain.DefaultChangelogSections()

	mockChangelog.EXPECT().Generate(
		domain.NewVersion(2, 0, 0), "api", gomock.Any(), sections,
	).Return("## api 2.0.0\n\n### Breaking\n- new api", nil)

	mockTag.EXPECT().FormatTag("api", domain.NewVersion(2, 0, 0)).Return("api/v2.0.0", nil)

	mockGit.EXPECT().HeadHash(gomock.Any()).Return("deadbeef", nil)
	mockGit.EXPECT().CreateTag(gomock.Any(), "api/v2.0.0", "deadbeef", gomock.Any()).Return(nil)
	mockGit.EXPECT().PushTag(gomock.Any(), "api/v2.0.0").Return(nil)

	mockPublisher.EXPECT().Publish(gomock.Any(), ports.PublishParams{
		TagName:   "api/v2.0.0",
		Version:   domain.NewVersion(2, 0, 0),
		Project:   "api",
		Changelog: "## api 2.0.0\n\n### Breaking\n- new api",
	}).Return(domain.ProjectReleaseResult{
		Published:  true,
		PublishURL: "https://github.com/org/repo/releases/tag/api/v2.0.0",
	}, nil)

	executor := app.MustNewReleaseExecutor(mockGit, mockTag, mockChangelog, mockPublisher, mockLogger, sections)

	plan := &domain.ReleasePlan{
		DryRun: false,
		Projects: []domain.ProjectReleasePlan{{
			Project:        domain.Project{Name: "api", Path: "services/api"},
			CurrentVersion: domain.NewVersion(1, 0, 0),
			NextVersion:    domain.NewVersion(2, 0, 0),
			ReleaseType:    domain.ReleaseMajor,
			Commits:        []domain.Commit{{Type: "feat", IsBreakingChange: true}},
			ShouldRelease:  true,
		}},
	}

	result, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(result.Projects) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Projects))
	}

	pr := result.Projects[0]
	if !pr.TagCreated {
		t.Error("tag should be created")
	}
	if !pr.Published {
		t.Error("release should be published")
	}
	if pr.PublishURL == "" {
		t.Error("publish URL should be set")
	}
}

func TestReleaseExecutor_Execute_PublishFailure(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)
	mockChangelog := mocks.NewMockChangelogGenerator(ctrl)
	mockPublisher := mocks.NewMockReleasePublisher(ctrl)
	mockLogger := mocks.NewMockLogger(ctrl)
	mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
	mockLogger.EXPECT().Warn(gomock.Any(), gomock.Any()).AnyTimes()
	// Execute logs at Error level when a project's publish step fails.
	mockLogger.EXPECT().Error(gomock.Any(), gomock.Any()).AnyTimes()

	sections := domain.DefaultChangelogSections()

	mockChangelog.EXPECT().Generate(
		domain.NewVersion(1, 1, 0), "api", gomock.Any(), sections,
	).Return("## 1.1.0", nil)

	mockTag.EXPECT().FormatTag("api", domain.NewVersion(1, 1, 0)).Return("api/v1.1.0", nil)

	mockGit.EXPECT().HeadHash(gomock.Any()).Return("cafebabe", nil)
	mockGit.EXPECT().CreateTag(gomock.Any(), "api/v1.1.0", "cafebabe", gomock.Any()).Return(nil)
	mockGit.EXPECT().PushTag(gomock.Any(), "api/v1.1.0").Return(nil)

	// Publish fails — this is a soft error: the tag is already pushed.
	mockPublisher.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(
		domain.ProjectReleaseResult{}, errors.New("github api unavailable"),
	)

	executor := app.MustNewReleaseExecutor(mockGit, mockTag, mockChangelog, mockPublisher, mockLogger, sections)

	plan := &domain.ReleasePlan{
		DryRun: false,
		Projects: []domain.ProjectReleasePlan{{
			Project:        domain.Project{Name: "api", Path: "services/api"},
			CurrentVersion: domain.NewVersion(1, 0, 0),
			NextVersion:    domain.NewVersion(1, 1, 0),
			ReleaseType:    domain.ReleaseMinor,
			Commits:        []domain.Commit{{Type: "feat", Description: "add feature"}},
			ShouldRelease:  true,
		}},
	}

	// Execute must not return a hard error — publish failures are soft.
	result, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() should not return hard error on publish failure, got: %v", err)
	}
	if len(result.Projects) != 1 {
		t.Fatalf("expected 1 project result, got %d", len(result.Projects))
	}
	pr := result.Projects[0]
	if pr.Error == nil {
		t.Error("expected per-project error on publish failure, got nil")
	}
	if !result.HasErrors() {
		t.Error("HasErrors() should return true when a project has a publish error")
	}
	if !pr.TagCreated {
		t.Error("tag should still be marked as created despite publish failure")
	}
}
