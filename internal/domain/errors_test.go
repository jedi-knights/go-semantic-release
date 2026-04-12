package domain_test

import (
	"errors"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestProjectError_Error(t *testing.T) {
	cause := errors.New("something failed")
	err := domain.NewProjectError("my-service", "tag", cause)

	got := err.Error()
	wantSubstrings := []string{"my-service", "tag", "something failed"}
	for _, sub := range wantSubstrings {
		if !containsString(got, sub) {
			t.Errorf("Error() = %q, missing substring %q", got, sub)
		}
	}
}

func TestProjectError_Unwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := domain.NewProjectError("svc", "op", cause)

	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the wrapped cause")
	}
}

func TestProjectError_As(t *testing.T) {
	cause := errors.New("inner")
	err := domain.NewProjectError("proj", "publish", cause)

	var pe *domain.ProjectError
	if !errors.As(err, &pe) {
		t.Fatal("errors.As should match *ProjectError")
	}
	if pe.Project != "proj" {
		t.Errorf("Project = %q, want %q", pe.Project, "proj")
	}
	if pe.Op != "publish" {
		t.Errorf("Op = %q, want %q", pe.Op, "publish")
	}
}

func TestReleaseError_Error(t *testing.T) {
	cause := errors.New("publish failed")
	err := domain.NewReleaseError("publish", cause)

	got := err.Error()
	wantSubstrings := []string{"publish", "publish failed"}
	for _, sub := range wantSubstrings {
		if !containsString(got, sub) {
			t.Errorf("Error() = %q, missing substring %q", got, sub)
		}
	}
}

func TestReleaseError_Unwrap(t *testing.T) {
	cause := errors.New("network error")
	err := domain.NewReleaseError("publish", cause)

	if !errors.Is(err, cause) {
		t.Error("errors.Is should find the wrapped cause")
	}
}

func TestReleaseError_As(t *testing.T) {
	cause := errors.New("inner")
	err := domain.NewReleaseError("verifyConditions", cause)

	var re *domain.ReleaseError
	if !errors.As(err, &re) {
		t.Fatal("errors.As should match *ReleaseError")
	}
	if re.Step != "verifyConditions" {
		t.Errorf("Step = %q, want %q", re.Step, "verifyConditions")
	}
}

func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinels := []error{
		domain.ErrNoReleasableChanges,
		domain.ErrInvalidVersion,
		domain.ErrInvalidCommit,
		domain.ErrProjectNotFound,
		domain.ErrTagAlreadyExists,
		domain.ErrBranchNotAllowed,
		domain.ErrDryRun,
	}

	for i := range sentinels {
		for j := range sentinels {
			if i == j {
				continue
			}
			if errors.Is(sentinels[i], sentinels[j]) {
				t.Errorf("sentinel[%d] should not match sentinel[%d]", i, j)
			}
		}
	}
}

// containsString reports whether s contains substr. Avoids importing strings
// to keep the domain test package lean.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}()
}
