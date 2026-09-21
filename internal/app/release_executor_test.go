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

	executor := app.MustNewReleaseExecutor(mockGit, mockTag, mockChangelog, mockPublisher, mockLogger, sections, domain.GitConfig{})

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

	executor := app.MustNewReleaseExecutor(mockGit, mockTag, mockChangelog, mockPublisher, mockLogger, sections, domain.GitConfig{})

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

// TestReleaseExecutor_Execute_StagesCommitsAndPushesReleaseAssets covers the
// regression this test guards against: when git.assets is configured, the
// prepared release files (CHANGELOG.md, version-bumped manifests, …) must be
// staged, committed, and pushed to the branch BEFORE the tag is created —
// otherwise the tag points at a pre-existing commit and the prepared changes
// are silently discarded on the runner instead of landing in the repository.
func TestReleaseExecutor_Execute_StagesCommitsAndPushesReleaseAssets(t *testing.T) {
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

	assets := []string{"CHANGELOG.md", "package.json"}

	// Stage -> Commit -> Push (branch) must happen, in order, before the tag
	// is created at the resulting HEAD.
	gomock.InOrder(
		mockGit.EXPECT().Stage(gomock.Any(), assets).Return(nil),
		mockGit.EXPECT().Commit(gomock.Any(), "chore(release): 1.1.0 [skip ci]").Return(nil),
		mockGit.EXPECT().Push(gomock.Any()).Return(nil),
		mockGit.EXPECT().HeadHash(gomock.Any()).Return("cafef00d", nil),
		mockGit.EXPECT().CreateTag(gomock.Any(), "v1.1.0", "cafef00d", gomock.Any()).Return(nil),
		mockGit.EXPECT().PushTag(gomock.Any(), "v1.1.0").Return(nil),
	)

	mockPublisher.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(
		domain.ProjectReleaseResult{Published: true}, nil,
	)

	gitConfig := domain.GitConfig{
		Assets:  assets,
		Message: "chore(release): {{.Version}} [skip ci]",
	}
	executor := app.MustNewReleaseExecutor(mockGit, mockTag, mockChangelog, mockPublisher, mockLogger, sections, gitConfig)

	plan := &domain.ReleasePlan{
		DryRun: false,
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
	if len(result.Projects) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Projects))
	}
	if !result.Projects[0].TagCreated {
		t.Error("tag should be created")
	}
}

// TestReleaseExecutor_Execute_NoAssetsConfigured_SkipsCommit verifies that
// with no git.assets configured, the executor does not stage/commit/push at
// all — it only tags at the pre-existing HEAD. This is the existing
// (single-repo, no prepared files) behavior and must not regress.
func TestReleaseExecutor_Execute_NoAssetsConfigured_SkipsCommit(t *testing.T) {
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
	).Return("## 1.1.0", nil)

	mockTag.EXPECT().FormatTag("", domain.NewVersion(1, 1, 0)).Return("v1.1.0", nil)

	// No Stage/Commit/Push expectations set up at all — gomock fails the test
	// if the executor calls them unexpectedly with an empty GitConfig.
	mockGit.EXPECT().HeadHash(gomock.Any()).Return("deadbeef", nil)
	mockGit.EXPECT().CreateTag(gomock.Any(), "v1.1.0", "deadbeef", gomock.Any()).Return(nil)
	mockGit.EXPECT().PushTag(gomock.Any(), "v1.1.0").Return(nil)

	mockPublisher.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(
		domain.ProjectReleaseResult{Published: true}, nil,
	)

	executor := app.MustNewReleaseExecutor(mockGit, mockTag, mockChangelog, mockPublisher, mockLogger, sections, domain.GitConfig{})

	plan := &domain.ReleasePlan{
		DryRun: false,
		Projects: []domain.ProjectReleasePlan{{
			Project:        domain.Project{Name: "", Path: "."},
			CurrentVersion: domain.NewVersion(1, 0, 0),
			NextVersion:    domain.NewVersion(1, 1, 0),
			ReleaseType:    domain.ReleaseMinor,
			Commits:        []domain.Commit{{Type: "feat", Description: "add feature"}},
			ShouldRelease:  true,
		}},
	}

	if _, err := executor.Execute(context.Background(), plan); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

// TestReleaseExecutor_Execute_PushBranchFailure_IsHardFailure verifies that a
// failure to push the release commit aborts the release (matching the
// documented error model: tag/push failures are hard failures) instead of
// proceeding to tag a commit that was never pushed.
func TestReleaseExecutor_Execute_PushBranchFailure_IsHardFailure(t *testing.T) {
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
	).Return("## 1.1.0", nil)

	mockTag.EXPECT().FormatTag("", domain.NewVersion(1, 1, 0)).Return("v1.1.0", nil)

	assets := []string{"CHANGELOG.md"}

	mockGit.EXPECT().Stage(gomock.Any(), assets).Return(nil)
	mockGit.EXPECT().Commit(gomock.Any(), gomock.Any()).Return(nil)
	mockGit.EXPECT().Push(gomock.Any()).Return(errors.New("remote rejected push"))
	// HeadHash/CreateTag/PushTag must never be called once Push fails.

	gitConfig := domain.GitConfig{Assets: assets}
	executor := app.MustNewReleaseExecutor(mockGit, mockTag, mockChangelog, mockPublisher, mockLogger, sections, gitConfig)

	plan := &domain.ReleasePlan{
		DryRun: false,
		Projects: []domain.ProjectReleasePlan{{
			Project:        domain.Project{Name: "", Path: "."},
			CurrentVersion: domain.NewVersion(1, 0, 0),
			NextVersion:    domain.NewVersion(1, 1, 0),
			ReleaseType:    domain.ReleaseMinor,
			Commits:        []domain.Commit{{Type: "feat", Description: "add feature"}},
			ShouldRelease:  true,
		}},
	}

	_, err := executor.Execute(context.Background(), plan)
	if err == nil {
		t.Fatal("Execute() should return an error when pushing the release branch fails")
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

	executor := app.MustNewReleaseExecutor(mockGit, mockTag, mockChangelog, mockPublisher, mockLogger, sections, domain.GitConfig{})

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

// TestReleaseExecutor_Execute_TagAlreadyExists verifies that a re-run where the
// tag was already created (e.g., a previous workflow attempt that failed after
// tagging) completes successfully without treating the existing tag as an error.
// PushTag must still be called because the tag may have been created locally
// but not yet pushed to the remote.
func TestReleaseExecutor_Execute_TagAlreadyExists(t *testing.T) {
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
		domain.NewVersion(1, 1, 0), "api", gomock.Any(), sections,
	).Return("## 1.1.0", nil)

	mockTag.EXPECT().FormatTag("api", domain.NewVersion(1, 1, 0)).Return("api/v1.1.0", nil)

	mockGit.EXPECT().HeadHash(gomock.Any()).Return("deadbeef", nil)
	// CreateTag returns ErrTagAlreadyExists — simulates a re-run where the tag
	// was already created in a previous workflow attempt at the same commit.
	mockGit.EXPECT().CreateTag(gomock.Any(), "api/v1.1.0", "deadbeef", gomock.Any()).
		Return(domain.ErrTagAlreadyExists)
	// PushTag must still be called: the tag may have been created locally but
	// not yet pushed to the remote.
	mockGit.EXPECT().PushTag(gomock.Any(), "api/v1.1.0").Return(nil)

	mockPublisher.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(
		domain.ProjectReleaseResult{Published: true, PublishURL: "https://github.com/org/repo/releases/tag/api/v1.1.0"}, nil,
	)

	executor := app.MustNewReleaseExecutor(mockGit, mockTag, mockChangelog, mockPublisher, mockLogger, sections, domain.GitConfig{})

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

	result, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() should not return error when tag already exists at current commit, got: %v", err)
	}
	if len(result.Projects) != 1 {
		t.Fatalf("expected 1 project result, got %d", len(result.Projects))
	}
	pr := result.Projects[0]
	if !pr.TagCreated {
		t.Error("TagCreated should be true when the tag already existed at current commit")
	}
	if pr.Error != nil {
		t.Errorf("unexpected project error: %v", pr.Error)
	}
}

// TestMustNewReleaseExecutor_PanicsOnNilArgs verifies that each nil argument
// causes an immediate panic so callers catch wiring mistakes at startup rather
// than at runtime.
func TestMustNewReleaseExecutor_PanicsOnNilArgs(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)

	validGit := mocks.NewMockGitRepository(ctrl)
	validTag := mocks.NewMockTagService(ctrl)
	validChangelog := mocks.NewMockChangelogGenerator(ctrl)
	validPublisher := mocks.NewMockReleasePublisher(ctrl)
	validLogger := mocks.NewMockLogger(ctrl)

	cases := []struct {
		name string
		fn   func()
	}{
		{"nil git", func() {
			app.MustNewReleaseExecutor(nil, validTag, validChangelog, validPublisher, validLogger, nil, domain.GitConfig{})
		}},
		{"nil tagService", func() {
			app.MustNewReleaseExecutor(validGit, nil, validChangelog, validPublisher, validLogger, nil, domain.GitConfig{})
		}},
		{"nil changelog", func() {
			app.MustNewReleaseExecutor(validGit, validTag, nil, validPublisher, validLogger, nil, domain.GitConfig{})
		}},
		{"nil publisher", func() {
			app.MustNewReleaseExecutor(validGit, validTag, validChangelog, nil, validLogger, nil, domain.GitConfig{})
		}},
		{"nil logger", func() {
			app.MustNewReleaseExecutor(validGit, validTag, validChangelog, validPublisher, nil, nil, domain.GitConfig{})
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("MustNewReleaseExecutor(%s): expected panic, got none", tc.name)
				}
			}()
			tc.fn()
		})
	}
}

// TestReleaseExecutor_Execute_TagAlreadyExists_PushAlreadyOnRemote covers the
// full re-run scenario: tag was already created AND already pushed to the
// remote in a prior workflow attempt. The adapter is responsible for
// translating the "nothing to push" condition (e.g. go-git's
// NoErrAlreadyUpToDate) into nil before returning from PushTag. At the
// executor level we verify the release completes successfully when both
// CreateTag and PushTag report "already done" — neither is treated as a
// hard failure.
func TestReleaseExecutor_Execute_TagAlreadyExists_PushAlreadyOnRemote(t *testing.T) {
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
		domain.NewVersion(1, 1, 0), "api", gomock.Any(), sections,
	).Return("## 1.1.0", nil)

	mockTag.EXPECT().FormatTag("api", domain.NewVersion(1, 1, 0)).Return("api/v1.1.0", nil)

	mockGit.EXPECT().HeadHash(gomock.Any()).Return("deadbeef", nil)
	mockGit.EXPECT().CreateTag(gomock.Any(), "api/v1.1.0", "deadbeef", gomock.Any()).
		Return(domain.ErrTagAlreadyExists)
	// Adapter has already translated NoErrAlreadyUpToDate → nil; the executor
	// must treat a nil return from PushTag as success regardless of the create path.
	mockGit.EXPECT().PushTag(gomock.Any(), "api/v1.1.0").Return(nil)

	mockPublisher.EXPECT().Publish(gomock.Any(), gomock.Any()).Return(
		domain.ProjectReleaseResult{Published: true}, nil,
	)

	executor := app.MustNewReleaseExecutor(mockGit, mockTag, mockChangelog, mockPublisher, mockLogger, sections, domain.GitConfig{})

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

	result, err := executor.Execute(context.Background(), plan)
	if err != nil {
		t.Fatalf("Execute() should not return error on full re-run, got: %v", err)
	}
	if len(result.Projects) != 1 {
		t.Fatalf("expected 1 project result, got %d", len(result.Projects))
	}
	pr := result.Projects[0]
	if !pr.TagCreated {
		t.Error("TagCreated should be true when tag already existed at current commit")
	}
	if pr.Error != nil {
		t.Errorf("unexpected project error: %v", pr.Error)
	}
}
