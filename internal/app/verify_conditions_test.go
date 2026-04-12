package app_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/jedi-knights/go-semantic-release/internal/app"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports/mocks"
)

// cfgNoGitHub returns a DefaultConfig with GitHub release creation disabled
// so branch-only tests are not affected by missing GitHub credentials.
func cfgNoGitHub() domain.Config {
	cfg := domain.DefaultConfig()
	cfg.GitHub.CreateRelease = false
	return cfg
}

func TestConditionVerifier_Verify_PassesOnAllowedBranch(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("main", nil)

	verifier := app.NewConditionVerifier(mockGit, cfgNoGitHub(), noopLogger{})
	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !result.Passed {
		t.Errorf("Verify() failed on allowed branch 'main': %v", result.Failures)
	}
}

func TestConditionVerifier_Verify_FailsOnUnknownBranch(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("feature/unknown", nil)

	verifier := app.NewConditionVerifier(mockGit, cfgNoGitHub(), noopLogger{})
	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Passed {
		t.Error("Verify() should fail on branch not in policy")
	}
	if len(result.Failures) == 0 {
		t.Error("Verify() should record a failure message")
	}
}

func TestConditionVerifier_Verify_GitError(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("", errors.New("not a git repo"))

	verifier := app.NewConditionVerifier(mockGit, domain.DefaultConfig(), noopLogger{})
	_, err := verifier.Verify(context.Background())
	if err == nil {
		t.Error("Verify() should return error when git fails")
	}
}

func TestConditionVerifier_Verify_MissingGitHubOwner(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("main", nil)

	cfg := domain.DefaultConfig()
	cfg.GitHub.CreateRelease = true
	cfg.GitHub.Owner = "" // missing
	cfg.GitHub.Repo = "my-repo"
	cfg.GitHub.Token = "tok"

	verifier := app.NewConditionVerifier(mockGit, cfg, noopLogger{})
	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Passed {
		t.Error("Verify() should fail when GitHub owner is missing")
	}
}

func TestConditionVerifier_Verify_MissingGitHubToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("main", nil)

	cfg := domain.DefaultConfig()
	cfg.GitHub.CreateRelease = true
	cfg.GitHub.Owner = "org"
	cfg.GitHub.Repo = "repo"
	cfg.GitHub.Token = "" // missing

	verifier := app.NewConditionVerifier(mockGit, cfg, noopLogger{})
	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Passed {
		t.Error("Verify() should fail when GitHub token is missing")
	}
}

func TestConditionVerifier_Verify_SkipsGitHubCheckWhenDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("main", nil)

	cfg := domain.DefaultConfig()
	cfg.GitHub.CreateRelease = false

	verifier := app.NewConditionVerifier(mockGit, cfg, noopLogger{})
	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !result.Passed {
		t.Errorf("Verify() failed unexpectedly: %v", result.Failures)
	}
}

func TestConditionVerifier_Verify_AllGitHubFieldsMissing(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockGit := mocks.NewMockGitRepository(ctrl)
	mockGit.EXPECT().CurrentBranch(gomock.Any()).Return("main", nil)

	cfg := domain.DefaultConfig()
	cfg.GitHub.CreateRelease = true
	// owner, repo, and token all empty

	verifier := app.NewConditionVerifier(mockGit, cfg, noopLogger{})
	result, err := verifier.Verify(context.Background())
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if result.Passed {
		t.Error("Verify() should fail when all GitHub fields are missing")
	}
	if len(result.Failures) == 0 {
		t.Error("Verify() should record at least one failure")
	}
}
