package platform_test

import (
	"os"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/platform"
)

func TestDetectCI_GitHubActions(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_REF_NAME", "main")
	t.Setenv("GITHUB_SHA", "abc123")

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
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unsetenv %s: %v", key, err)
		}
	}

	ci := platform.DetectCI()
	if ci.Detected {
		t.Errorf("expected CI not detected, got %q", ci.Name)
	}
}

func TestIsCI(t *testing.T) {
	t.Setenv("CI", "true")

	if !platform.IsCI() {
		t.Error("expected IsCI to return true when CI env var is set")
	}
}
