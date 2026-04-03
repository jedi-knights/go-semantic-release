package platform_test

import (
	"os"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/platform"
)

func TestDetectCI_GitHubActions(t *testing.T) {
	os.Setenv("GITHUB_ACTIONS", "true")
	os.Setenv("GITHUB_REF_NAME", "main")
	os.Setenv("GITHUB_SHA", "abc123")
	defer func() {
		os.Unsetenv("GITHUB_ACTIONS")
		os.Unsetenv("GITHUB_REF_NAME")
		os.Unsetenv("GITHUB_SHA")
	}()

	ci := platform.DetectCI()
	if !ci.Detected {
		t.Error("expected CI to be detected")
	}
	if ci.Name != "GitHub Actions" {
		t.Errorf("name = %q, want %q", ci.Name, "GitHub Actions")
	}
	if ci.Branch != "main" {
		t.Errorf("branch = %q, want %q", ci.Branch, "main")
	}
}

func TestDetectCI_NotDetected(t *testing.T) {
	// Clear all CI env vars.
	for _, key := range []string{"GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI", "TRAVIS", "JENKINS_URL", "TF_BUILD", "BITBUCKET_BUILD_NUMBER", "CI"} {
		os.Unsetenv(key)
	}

	ci := platform.DetectCI()
	if ci.Detected {
		t.Errorf("expected CI not detected, got %q", ci.Name)
	}
}

func TestIsCI(t *testing.T) {
	os.Setenv("CI", "true")
	defer os.Unsetenv("CI")

	if !platform.IsCI() {
		t.Error("expected IsCI to return true when CI env var is set")
	}
}
