package plugins_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/plugins"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports/mocks"
)

func TestGitPlugin_Name(t *testing.T) {
	ctrl := gomock.NewController(t)
	p := plugins.NewGitPlugin(
		mocks.NewMockGitRepository(ctrl),
		mocks.NewMockTagService(ctrl),
		mocks.NewMockFileSystem(ctrl),
		noopLogger{},
		domain.DefaultGitIdentity(),
		domain.GitConfig{},
	)
	if p.Name() != "git" {
		t.Errorf("Name() = %q, want git", p.Name())
	}
}

func TestGitPlugin_VerifyConditions_PassesWhenGitAccessible(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("main", nil)

	p := plugins.NewGitPlugin(
		mockGit,
		mocks.NewMockTagService(ctrl),
		mocks.NewMockFileSystem(ctrl),
		noopLogger{},
		domain.DefaultGitIdentity(),
		domain.GitConfig{},
	)

	if err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{}); err != nil {
		t.Errorf("VerifyConditions() error = %v", err)
	}
}

func TestGitPlugin_VerifyConditions_FailsWhenGitUnaccessible(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("", errors.New("not a git repo"))

	p := plugins.NewGitPlugin(
		mockGit,
		mocks.NewMockTagService(ctrl),
		mocks.NewMockFileSystem(ctrl),
		noopLogger{},
		domain.DefaultGitIdentity(),
		domain.GitConfig{},
	)

	if err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{}); err == nil {
		t.Error("VerifyConditions() should return error when git is not accessible")
	}
}

func TestGitPlugin_Publish_NilProject(t *testing.T) {
	ctrl := gomock.NewController(t)
	p := plugins.NewGitPlugin(
		mocks.NewMockGitRepository(ctrl),
		mocks.NewMockTagService(ctrl),
		mocks.NewMockFileSystem(ctrl),
		noopLogger{},
		domain.DefaultGitIdentity(),
		domain.GitConfig{},
	)

	rc := &domain.ReleaseContext{CurrentProject: nil}
	result, err := p.Publish(context.Background(), rc)
	if err != nil {
		t.Fatalf("Publish() with nil project error = %v", err)
	}
	if result != nil {
		t.Error("Publish() with nil project should return nil result")
	}
}

func TestGitPlugin_Publish_CreatesAndPushesTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)

	version := domain.NewVersion(1, 2, 3)
	project := domain.Project{Name: "my-svc", Path: "./my-svc"}

	mockTag.EXPECT().FormatTag("my-svc", version).Return("my-svc/v1.2.3", nil)
	mockGit.EXPECT().HeadHash(gomock.Any()).Return("deadbeef", nil)
	mockGit.EXPECT().CreateTag(gomock.Any(), "my-svc/v1.2.3", "deadbeef", gomock.Any()).Return(nil)
	mockGit.EXPECT().PushTag(gomock.Any(), "my-svc/v1.2.3").Return(nil)

	p := plugins.NewGitPlugin(
		mockGit,
		mockTag,
		mocks.NewMockFileSystem(ctrl),
		noopLogger{},
		domain.DefaultGitIdentity(),
		domain.GitConfig{},
	)

	rc := &domain.ReleaseContext{
		Notes: "## 1.2.3",
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     project,
			NextVersion: version,
		},
	}

	result, err := p.Publish(context.Background(), rc)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if result == nil {
		t.Fatal("Publish() returned nil result")
	}
	if result.TagName != "my-svc/v1.2.3" {
		t.Errorf("TagName = %q, want my-svc/v1.2.3", result.TagName)
	}
	if !result.TagCreated {
		t.Error("TagCreated should be true")
	}
}

func TestGitPlugin_Publish_HeadHashError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)

	version := domain.NewVersion(1, 0, 0)
	mockTag.EXPECT().FormatTag("svc", version).Return("svc/v1.0.0", nil)
	mockGit.EXPECT().HeadHash(gomock.Any()).Return("", errors.New("HEAD not found"))

	p := plugins.NewGitPlugin(
		mockGit, mockTag,
		mocks.NewMockFileSystem(ctrl),
		noopLogger{},
		domain.DefaultGitIdentity(),
		domain.GitConfig{},
	)

	rc := &domain.ReleaseContext{
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "svc"},
			NextVersion: version,
		},
	}

	_, err := p.Publish(context.Background(), rc)
	if err == nil {
		t.Error("Publish() should return error when HeadHash fails")
	}
}

func TestGitPlugin_Publish_FormatTagError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)

	version := domain.NewVersion(1, 0, 0)
	mockTag.EXPECT().FormatTag("svc", version).Return("", errors.New("invalid template"))

	p := plugins.NewGitPlugin(
		mockGit, mockTag,
		mocks.NewMockFileSystem(ctrl),
		noopLogger{},
		domain.DefaultGitIdentity(),
		domain.GitConfig{},
	)

	rc := &domain.ReleaseContext{
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "svc"},
			NextVersion: version,
		},
	}

	_, err := p.Publish(context.Background(), rc)
	if err == nil {
		t.Error("Publish() should return error when FormatTag fails")
	}
}

func TestGitPlugin_Publish_PushTagError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)

	version := domain.NewVersion(1, 0, 0)
	mockTag.EXPECT().FormatTag("svc", version).Return("svc/v1.0.0", nil)
	mockGit.EXPECT().HeadHash(gomock.Any()).Return("abc", nil)
	mockGit.EXPECT().CreateTag(gomock.Any(), "svc/v1.0.0", "abc", gomock.Any()).Return(nil)
	mockGit.EXPECT().PushTag(gomock.Any(), "svc/v1.0.0").Return(errors.New("push rejected"))

	p := plugins.NewGitPlugin(
		mockGit, mockTag,
		mocks.NewMockFileSystem(ctrl),
		noopLogger{},
		domain.DefaultGitIdentity(),
		domain.GitConfig{},
	)

	rc := &domain.ReleaseContext{
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "svc"},
			NextVersion: version,
		},
	}

	_, err := p.Publish(context.Background(), rc)
	if err == nil {
		t.Error("Publish() should return error when PushTag fails")
	}
}

func TestGitPlugin_Publish_TagCreationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)

	version := domain.NewVersion(1, 0, 0)
	mockTag.EXPECT().FormatTag("svc", version).Return("svc/v1.0.0", nil)
	mockGit.EXPECT().HeadHash(gomock.Any()).Return("abc", nil)
	mockGit.EXPECT().CreateTag(gomock.Any(), "svc/v1.0.0", "abc", gomock.Any()).Return(errors.New("tag exists"))

	p := plugins.NewGitPlugin(
		mockGit, mockTag,
		mocks.NewMockFileSystem(ctrl),
		noopLogger{},
		domain.DefaultGitIdentity(),
		domain.GitConfig{},
	)

	rc := &domain.ReleaseContext{
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "svc"},
			NextVersion: version,
		},
	}

	_, err := p.Publish(context.Background(), rc)
	if err == nil {
		t.Error("Publish() should return error when tag creation fails")
	}
}

func TestGitPlugin_Publish_CommitMessageIncludesNotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)

	version := domain.NewVersion(1, 0, 0)
	mockTag.EXPECT().FormatTag("svc", version).Return("svc/v1.0.0", nil)
	mockGit.EXPECT().Stage(gomock.Any(), gomock.Any()).Return(nil)
	mockGit.EXPECT().Commit(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, msg string) error {
			if !strings.Contains(msg, "chore(release): 1.0.0") {
				t.Errorf("commit message missing version, got: %q", msg)
			}
			if !strings.Contains(msg, "## 1.0.0") {
				t.Errorf("commit message missing release notes, got: %q", msg)
			}
			return nil
		},
	)
	mockGit.EXPECT().Push(gomock.Any()).Return(nil)
	mockGit.EXPECT().HeadHash(gomock.Any()).Return("abc", nil)
	mockGit.EXPECT().CreateTag(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mockGit.EXPECT().PushTag(gomock.Any(), gomock.Any()).Return(nil)

	p := plugins.NewGitPlugin(
		mockGit, mockTag,
		mocks.NewMockFileSystem(ctrl),
		noopLogger{},
		domain.DefaultGitIdentity(),
		domain.GitConfig{
			Assets:  []string{"CHANGELOG.md"},
			Message: "chore(release): {{.Version}} [skip ci]\n\n{{.Notes}}",
		},
	)

	rc := &domain.ReleaseContext{
		Notes: "## 1.0.0\n\n- feat: something",
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "svc"},
			NextVersion: version,
		},
	}

	if _, err := p.Publish(context.Background(), rc); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
}

func TestGitPlugin_Publish_StagesCommitsAndPushesAssetsBeforeTagging(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)

	version := domain.NewVersion(1, 2, 3)
	project := domain.Project{Name: "my-svc"}
	assets := []string{"pyproject.toml", "uv.lock"}

	// Ordering matters: Stage → Commit → Push → HeadHash → CreateTag → PushTag.
	gomock.InOrder(
		mockTag.EXPECT().FormatTag("my-svc", version).Return("my-svc/v1.2.3", nil),
		mockGit.EXPECT().Stage(gomock.Any(), assets).Return(nil),
		mockGit.EXPECT().Commit(gomock.Any(), gomock.Any()).Return(nil),
		mockGit.EXPECT().Push(gomock.Any()).Return(nil),
		mockGit.EXPECT().HeadHash(gomock.Any()).Return("abc123", nil),
		mockGit.EXPECT().CreateTag(gomock.Any(), "my-svc/v1.2.3", "abc123", gomock.Any()).Return(nil),
		mockGit.EXPECT().PushTag(gomock.Any(), "my-svc/v1.2.3").Return(nil),
	)

	p := plugins.NewGitPlugin(
		mockGit, mockTag,
		mocks.NewMockFileSystem(ctrl),
		noopLogger{},
		domain.DefaultGitIdentity(),
		domain.GitConfig{Assets: assets, Message: "chore(release): {{.Version}}"},
	)

	rc := &domain.ReleaseContext{
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     project,
			NextVersion: version,
		},
	}

	result, err := p.Publish(context.Background(), rc)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if result == nil || result.TagName != "my-svc/v1.2.3" {
		t.Errorf("unexpected result: %v", result)
	}
}

func TestGitPlugin_Publish_SkipsCommitWhenNoAssets(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)

	version := domain.NewVersion(1, 0, 0)
	mockTag.EXPECT().FormatTag("svc", version).Return("svc/v1.0.0", nil)
	// Stage/Commit/Push must NOT be called when Assets is empty.
	mockGit.EXPECT().HeadHash(gomock.Any()).Return("abc", nil)
	mockGit.EXPECT().CreateTag(gomock.Any(), "svc/v1.0.0", "abc", gomock.Any()).Return(nil)
	mockGit.EXPECT().PushTag(gomock.Any(), "svc/v1.0.0").Return(nil)

	p := plugins.NewGitPlugin(
		mockGit, mockTag,
		mocks.NewMockFileSystem(ctrl),
		noopLogger{},
		domain.DefaultGitIdentity(),
		domain.GitConfig{}, // no assets
	)

	rc := &domain.ReleaseContext{
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "svc"},
			NextVersion: version,
		},
	}

	if _, err := p.Publish(context.Background(), rc); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
}

func TestGitPlugin_Publish_StageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)

	version := domain.NewVersion(1, 0, 0)
	mockTag.EXPECT().FormatTag("svc", version).Return("svc/v1.0.0", nil)
	mockGit.EXPECT().Stage(gomock.Any(), gomock.Any()).Return(errors.New("stage failed"))

	p := plugins.NewGitPlugin(
		mockGit, mockTag,
		mocks.NewMockFileSystem(ctrl),
		noopLogger{},
		domain.DefaultGitIdentity(),
		domain.GitConfig{Assets: []string{"file.txt"}},
	)

	rc := &domain.ReleaseContext{
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "svc"},
			NextVersion: version,
		},
	}

	_, err := p.Publish(context.Background(), rc)
	if err == nil {
		t.Error("Publish() should return error when Stage fails")
	}
}

func TestGitPlugin_Publish_CommitError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)

	version := domain.NewVersion(1, 0, 0)
	mockTag.EXPECT().FormatTag("svc", version).Return("svc/v1.0.0", nil)
	mockGit.EXPECT().Stage(gomock.Any(), gomock.Any()).Return(nil)
	mockGit.EXPECT().Commit(gomock.Any(), gomock.Any()).Return(errors.New("nothing to commit"))

	p := plugins.NewGitPlugin(
		mockGit, mockTag,
		mocks.NewMockFileSystem(ctrl),
		noopLogger{},
		domain.DefaultGitIdentity(),
		domain.GitConfig{Assets: []string{"file.txt"}},
	)

	rc := &domain.ReleaseContext{
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "svc"},
			NextVersion: version,
		},
	}

	_, err := p.Publish(context.Background(), rc)
	if err == nil {
		t.Error("Publish() should return error when Commit fails")
	}
}

func TestGitPlugin_Publish_PushBranchError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockTag := mocks.NewMockTagService(ctrl)

	version := domain.NewVersion(1, 0, 0)
	mockTag.EXPECT().FormatTag("svc", version).Return("svc/v1.0.0", nil)
	mockGit.EXPECT().Stage(gomock.Any(), gomock.Any()).Return(nil)
	mockGit.EXPECT().Commit(gomock.Any(), gomock.Any()).Return(nil)
	mockGit.EXPECT().Push(gomock.Any()).Return(errors.New("push rejected"))

	p := plugins.NewGitPlugin(
		mockGit, mockTag,
		mocks.NewMockFileSystem(ctrl),
		noopLogger{},
		domain.DefaultGitIdentity(),
		domain.GitConfig{Assets: []string{"file.txt"}},
	)

	rc := &domain.ReleaseContext{
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "svc"},
			NextVersion: version,
		},
	}

	_, err := p.Publish(context.Background(), rc)
	if err == nil {
		t.Error("Publish() should return error when Push fails")
	}
}
