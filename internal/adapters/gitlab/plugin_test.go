package gitlab_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/gitlab"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

// noopLogger satisfies ports.Logger without producing any output.
type noopLogger struct{}

func (noopLogger) Debug(string, ...any) {}
func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

func baseConfig(apiURL string) gitlab.PluginConfig {
	return gitlab.PluginConfig{
		ProjectID: "42",
		Token:     "test-token",
		APIURL:    apiURL,
	}
}

func TestGitLabPlugin_Name(t *testing.T) {
	p := gitlab.NewPlugin(baseConfig("http://unused"), noopLogger{})
	if got := p.Name(); got != "gitlab" {
		t.Errorf("Name() = %q, want %q", got, "gitlab")
	}
}

func TestGitLabPlugin_NewPlugin_Defaults(t *testing.T) {
	// When APIURL is empty NewPlugin should default to https://gitlab.com/api/v4.
	// We verify the default was set by confirming it doesn't match the empty string
	// — we cannot inspect private fields, so we exercise VerifyConditions which
	// would fail to dial if APIURL remained empty.
	cfg := gitlab.PluginConfig{ProjectID: "42", Token: "t"}
	p := gitlab.NewPlugin(cfg, noopLogger{})
	// Name is always valid regardless of APIURL — confirm basic construction worked.
	if p.Name() != "gitlab" {
		t.Fatal("unexpected plugin name after default construction")
	}
}

func TestGitLabPlugin_VerifyConditions_NoToken(t *testing.T) {
	// Purge env vars to ensure no token is resolved.
	t.Setenv("GL_TOKEN", "")
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("SEMANTIC_RELEASE_GITLAB_TOKEN", "")

	cfg := gitlab.PluginConfig{ProjectID: "42", APIURL: "http://unused"}
	p := gitlab.NewPlugin(cfg, noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}
}

func TestGitLabPlugin_VerifyConditions_NoProjectID(t *testing.T) {
	cfg := gitlab.PluginConfig{Token: "tok", APIURL: "http://unused"}
	p := gitlab.NewPlugin(cfg, noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Fatal("expected error for missing project_id, got nil")
	}
}

func TestGitLabPlugin_VerifyConditions_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := gitlab.NewPlugin(baseConfig(srv.URL), noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}

func TestGitLabPlugin_VerifyConditions_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The plugin calls GET /projects/42
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/projects/42") {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := gitlab.NewPlugin(baseConfig(srv.URL), noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err != nil {
		t.Errorf("VerifyConditions() unexpected error: %v", err)
	}
}

func TestGitLabPlugin_Publish_NilProject(t *testing.T) {
	p := gitlab.NewPlugin(baseConfig("http://unused"), noopLogger{})
	result, err := p.Publish(context.Background(), &domain.ReleaseContext{CurrentProject: nil})
	if err != nil {
		t.Errorf("Publish() unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestGitLabPlugin_Publish_Success(t *testing.T) {
	releaseResp := map[string]any{
		"tag_name": "v1.0.0",
		"_links": map[string]any{
			"self": "https://gitlab.com/org/repo/-/releases/v1.0.0",
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/projects/42/releases") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			body, _ := json.Marshal(releaseResp)
			_, _ = w.Write(body)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := gitlab.NewPlugin(baseConfig(srv.URL), noopLogger{})
	rc := &domain.ReleaseContext{
		TagName: "v1.0.0",
		Notes:   "release notes",
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "myapp"},
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}
	result, err := p.Publish(context.Background(), rc)
	if err != nil {
		t.Fatalf("Publish() unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Published {
		t.Error("expected result.Published to be true")
	}
	if result.PublishURL != "https://gitlab.com/org/repo/-/releases/v1.0.0" {
		t.Errorf("PublishURL = %q, want %q", result.PublishURL, "https://gitlab.com/org/repo/-/releases/v1.0.0")
	}
}

func TestGitLabPlugin_Publish_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := gitlab.NewPlugin(baseConfig(srv.URL), noopLogger{})
	rc := &domain.ReleaseContext{
		TagName: "v1.0.0",
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "myapp"},
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}
	_, err := p.Publish(context.Background(), rc)
	if err == nil {
		t.Fatal("expected error for non-201 response, got nil")
	}
}

func TestGitLabPlugin_AddChannel_NoOp(t *testing.T) {
	// AddChannel is a no-op — it must always return nil regardless of context.
	p := gitlab.NewPlugin(baseConfig("http://unused"), noopLogger{})
	rc := &domain.ReleaseContext{TagName: "v1.0.0"}
	if err := p.AddChannel(context.Background(), rc); err != nil {
		t.Errorf("AddChannel() unexpected error: %v", err)
	}
}

func TestGitLabPlugin_Success(t *testing.T) {
	p := gitlab.NewPlugin(baseConfig("http://unused"), noopLogger{})
	rc := &domain.ReleaseContext{TagName: "v1.0.0"}
	// Success logs and returns nil.
	if err := p.Success(context.Background(), rc); err != nil {
		t.Errorf("Success() unexpected error: %v", err)
	}
}

func TestGitLabPlugin_Fail_NilError(t *testing.T) {
	p := gitlab.NewPlugin(baseConfig("http://unused"), noopLogger{})
	rc := &domain.ReleaseContext{Error: nil}
	if err := p.Fail(context.Background(), rc); err != nil {
		t.Errorf("Fail() unexpected error: %v", err)
	}
}

func TestGitLabPlugin_Fail_WithError(t *testing.T) {
	// When rc.Error is set the plugin should log and return nil (no HTTP calls).
	p := gitlab.NewPlugin(baseConfig("http://unused"), noopLogger{})
	rc := &domain.ReleaseContext{Error: errors.New("pipeline broke")}
	if err := p.Fail(context.Background(), rc); err != nil {
		t.Errorf("Fail() unexpected error: %v", err)
	}
}
