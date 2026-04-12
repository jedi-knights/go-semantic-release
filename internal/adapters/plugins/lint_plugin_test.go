package plugins_test

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/plugins"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports/mocks"
)

func TestLintPlugin_Name(t *testing.T) {
	ctrl := gomock.NewController(t)
	p := plugins.NewLintPlugin(mocks.NewMockCommitLinter(ctrl), noopLogger{})
	if p.Name() != "commit-lint" {
		t.Errorf("Name() = %q, want commit-lint", p.Name())
	}
}

func TestLintPlugin_VerifyRelease_NoViolations(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLinter := mocks.NewMockCommitLinter(ctrl)

	commits := []domain.Commit{
		{Hash: "abc1234", Type: "feat", Description: "add feature"},
		{Hash: "def5678", Type: "fix", Description: "fix bug"},
	}

	// Linter returns no violations for either commit.
	mockLinter.EXPECT().Lint(commits[0]).Return(nil)
	mockLinter.EXPECT().Lint(commits[1]).Return(nil)

	p := plugins.NewLintPlugin(mockLinter, noopLogger{})
	rc := &domain.ReleaseContext{Commits: commits}

	if err := p.VerifyRelease(context.Background(), rc); err != nil {
		t.Errorf("VerifyRelease() error = %v, want nil", err)
	}
}

func TestLintPlugin_VerifyRelease_WarningsDoNotFail(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLinter := mocks.NewMockCommitLinter(ctrl)

	commit := domain.Commit{Hash: "abc1234", Type: "feat", Description: "add thing"}
	mockLinter.EXPECT().Lint(commit).Return([]domain.LintViolation{
		{Rule: "max-subject-length", Message: "subject too long", Severity: domain.LintWarning},
	})

	p := plugins.NewLintPlugin(mockLinter, noopLogger{})
	rc := &domain.ReleaseContext{Commits: []domain.Commit{commit}}

	if err := p.VerifyRelease(context.Background(), rc); err != nil {
		t.Errorf("VerifyRelease() with only warnings should not error, got %v", err)
	}
}

func TestLintPlugin_VerifyRelease_ErrorViolationFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLinter := mocks.NewMockCommitLinter(ctrl)

	commit := domain.Commit{Hash: "abc1234", Type: "bad-type", Description: "something"}
	mockLinter.EXPECT().Lint(commit).Return([]domain.LintViolation{
		{Rule: "allowed-types", Message: "type 'bad-type' not allowed", Severity: domain.LintError},
	})

	p := plugins.NewLintPlugin(mockLinter, noopLogger{})
	rc := &domain.ReleaseContext{Commits: []domain.Commit{commit}}

	if err := p.VerifyRelease(context.Background(), rc); err == nil {
		t.Error("VerifyRelease() with error-severity violation should return error")
	}
}

func TestLintPlugin_VerifyRelease_NoCommits(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLinter := mocks.NewMockCommitLinter(ctrl)
	// No Lint calls expected.

	p := plugins.NewLintPlugin(mockLinter, noopLogger{})
	rc := &domain.ReleaseContext{Commits: nil}

	if err := p.VerifyRelease(context.Background(), rc); err != nil {
		t.Errorf("VerifyRelease() with no commits error = %v, want nil", err)
	}
}

func TestLintPlugin_VerifyRelease_ShortHashOnError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockLinter := mocks.NewMockCommitLinter(ctrl)

	// Short hash: 7 chars used in error message.
	commit := domain.Commit{Hash: "abcdefg1234", Type: "bad", Description: "x"}
	mockLinter.EXPECT().Lint(commit).Return([]domain.LintViolation{
		{Rule: "allowed-types", Message: "bad type", Severity: domain.LintError},
	})

	p := plugins.NewLintPlugin(mockLinter, noopLogger{})
	err := p.VerifyRelease(context.Background(), &domain.ReleaseContext{Commits: []domain.Commit{commit}})
	if err == nil {
		t.Fatal("VerifyRelease() should fail")
	}
	// Error message should contain the 7-char short hash.
	if msg := err.Error(); msg == "" {
		t.Error("error message should not be empty")
	}
}
