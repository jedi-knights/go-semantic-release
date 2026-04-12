package bitbucket_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/bitbucket"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

// noopLogger satisfies ports.Logger without producing any output.
type noopLogger struct{}

func (noopLogger) Debug(string, ...any) {}
func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

func baseConfig(apiURL string) bitbucket.PluginConfig {
	return bitbucket.PluginConfig{
		Workspace: "myworkspace",
		RepoSlug:  "myrepo",
		Token:     "test-token",
		APIURL:    apiURL,
	}
}

func TestBitbucketPlugin_Name(t *testing.T) {
	p := bitbucket.NewPlugin(baseConfig("http://unused"), noopLogger{})
	if got := p.Name(); got != "bitbucket" {
		t.Errorf("Name() = %q, want %q", got, "bitbucket")
	}
}

func TestBitbucketPlugin_NewPlugin_Defaults(t *testing.T) {
	// When APIURL is empty NewPlugin should default to https://api.bitbucket.org/2.0.
	// Confirm construction succeeds and returns a functioning plugin.
	cfg := bitbucket.PluginConfig{Workspace: "w", RepoSlug: "r", Token: "t"}
	p := bitbucket.NewPlugin(cfg, noopLogger{})
	if p.Name() != "bitbucket" {
		t.Fatal("unexpected plugin name after default construction")
	}
}

func TestBitbucketPlugin_NewPlugin_TokenFromEnv(t *testing.T) {
	t.Setenv("BB_TOKEN", "env-bb-token")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer env-bb-token" {
			http.Error(w, "unexpected auth: "+auth, http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := bitbucket.PluginConfig{
		Workspace: "myworkspace",
		RepoSlug:  "myrepo",
		APIURL:    srv.URL,
		// Token intentionally empty — should be resolved from BB_TOKEN
	}
	p := bitbucket.NewPlugin(cfg, noopLogger{})
	if err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{}); err != nil {
		t.Errorf("VerifyConditions() with env token failed: %v", err)
	}
}

func TestBitbucketPlugin_VerifyConditions_NoToken(t *testing.T) {
	t.Setenv("BB_TOKEN", "")
	t.Setenv("BITBUCKET_TOKEN", "")
	t.Setenv("SEMANTIC_RELEASE_BITBUCKET_TOKEN", "")

	cfg := bitbucket.PluginConfig{Workspace: "w", RepoSlug: "r", APIURL: "http://unused"}
	p := bitbucket.NewPlugin(cfg, noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}
}

func TestBitbucketPlugin_VerifyConditions_NoWorkspaceRepoSlug(t *testing.T) {
	cfg := bitbucket.PluginConfig{Token: "tok", APIURL: "http://unused"}
	p := bitbucket.NewPlugin(cfg, noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Fatal("expected error for missing workspace/repo_slug, got nil")
	}
}

func TestBitbucketPlugin_VerifyConditions_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := bitbucket.NewPlugin(baseConfig(srv.URL), noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}

func TestBitbucketPlugin_VerifyConditions_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	p := bitbucket.NewPlugin(baseConfig(srv.URL), noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
}

func TestBitbucketPlugin_VerifyConditions_UnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	p := bitbucket.NewPlugin(baseConfig(srv.URL), noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
}

func TestBitbucketPlugin_VerifyConditions_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/repositories/myworkspace/myrepo") {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := bitbucket.NewPlugin(baseConfig(srv.URL), noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err != nil {
		t.Errorf("VerifyConditions() unexpected error: %v", err)
	}
}

func TestBitbucketPlugin_Publish_NilProject(t *testing.T) {
	p := bitbucket.NewPlugin(baseConfig("http://unused"), noopLogger{})
	result, err := p.Publish(context.Background(), &domain.ReleaseContext{CurrentProject: nil})
	if err != nil {
		t.Errorf("Publish() unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestBitbucketPlugin_Publish_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/repositories/myworkspace/myrepo/refs/tags") {
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := bitbucket.NewPlugin(baseConfig(srv.URL), noopLogger{})
	rc := &domain.ReleaseContext{
		TagName: "v1.0.0",
		Notes:   "release notes",
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "myapp", Path: "."},
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
	if result.TagName != "v1.0.0" {
		t.Errorf("TagName = %q, want %q", result.TagName, "v1.0.0")
	}
}

func TestBitbucketPlugin_Publish_SuccessWithStatus200(t *testing.T) {
	// Bitbucket may return 200 OK for an existing tag update — the plugin accepts both 200 and 201.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/refs/tags") {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := bitbucket.NewPlugin(baseConfig(srv.URL), noopLogger{})
	rc := &domain.ReleaseContext{
		TagName: "v2.0.0",
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "myapp"},
			NextVersion: domain.NewVersion(2, 0, 0),
		},
	}
	result, err := p.Publish(context.Background(), rc)
	if err != nil {
		t.Fatalf("Publish() unexpected error for 200 response: %v", err)
	}
	if result == nil || !result.Published {
		t.Error("expected successful result for 200 response")
	}
}

func TestBitbucketPlugin_Publish_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := bitbucket.NewPlugin(baseConfig(srv.URL), noopLogger{})
	rc := &domain.ReleaseContext{
		TagName: "v1.0.0",
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "myapp"},
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}
	_, err := p.Publish(context.Background(), rc)
	if err == nil {
		t.Fatal("expected error for non-2xx response, got nil")
	}
}

func TestBitbucketPlugin_AddChannel_NoOp(t *testing.T) {
	p := bitbucket.NewPlugin(baseConfig("http://unused"), noopLogger{})
	rc := &domain.ReleaseContext{TagName: "v1.0.0"}
	if err := p.AddChannel(context.Background(), rc); err != nil {
		t.Errorf("AddChannel() unexpected error: %v", err)
	}
}

func TestBitbucketPlugin_Success(t *testing.T) {
	p := bitbucket.NewPlugin(baseConfig("http://unused"), noopLogger{})
	rc := &domain.ReleaseContext{TagName: "v1.0.0"}
	if err := p.Success(context.Background(), rc); err != nil {
		t.Errorf("Success() unexpected error: %v", err)
	}
}

func TestBitbucketPlugin_Fail_NilError(t *testing.T) {
	p := bitbucket.NewPlugin(baseConfig("http://unused"), noopLogger{})
	rc := &domain.ReleaseContext{Error: nil}
	if err := p.Fail(context.Background(), rc); err != nil {
		t.Errorf("Fail() unexpected error: %v", err)
	}
}

func TestBitbucketPlugin_Fail_WithError(t *testing.T) {
	// When rc.Error is set the plugin should log and return nil (no HTTP calls).
	p := bitbucket.NewPlugin(baseConfig("http://unused"), noopLogger{})
	rc := &domain.ReleaseContext{Error: errors.New("deployment failed")}
	if err := p.Fail(context.Background(), rc); err != nil {
		t.Errorf("Fail() unexpected error: %v", err)
	}
}
