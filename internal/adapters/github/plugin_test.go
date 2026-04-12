package github_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/github"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

// noopLogger satisfies ports.Logger without producing any output.
type noopLogger struct{}

func (noopLogger) Debug(string, ...any) {}
func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

func baseConfig(apiURL string) github.PluginConfig {
	return github.PluginConfig{
		Owner:  "owner",
		Repo:   "repo",
		Token:  "test-token",
		APIURL: apiURL,
	}
}

func TestPlugin_Name(t *testing.T) {
	p := github.NewPlugin(baseConfig("http://unused"), noopLogger{})
	if got := p.Name(); got != "github" {
		t.Errorf("Name() = %q, want %q", got, "github")
	}
}

func TestPlugin_NewPlugin_Defaults(t *testing.T) {
	// When no optional fields are provided the constructor should supply sensible defaults.
	cfg := github.PluginConfig{
		Owner: "o",
		Repo:  "r",
		Token: "t",
	}
	p := github.NewPlugin(cfg, noopLogger{})
	if p.Name() != "github" {
		t.Fatal("unexpected plugin name")
	}

	// Verify defaults indirectly by exercising a path that uses them.
	// We can't inspect the struct fields from outside the package, but a successful
	// Fail call that creates an issue exercises ReleasedLabels/FailLabels defaults.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues"):
			// Return an empty list so the plugin falls through to createIssue.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("[]"))
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues"):
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	cfg2 := github.PluginConfig{
		Owner:  "o",
		Repo:   "r",
		Token:  "t",
		APIURL: srv.URL,
	}
	p2 := github.NewPlugin(cfg2, noopLogger{})
	rc := &domain.ReleaseContext{Error: errors.New("boom")}
	if err := p2.Fail(context.Background(), rc); err != nil {
		t.Errorf("Fail() with defaults failed: %v", err)
	}
}

func TestPlugin_NewPlugin_TokenFromEnv(t *testing.T) {
	t.Setenv("GH_TOKEN", "env-token")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// VerifyConditions calls GET /repos/{owner}/{repo}
		auth := r.Header.Get("Authorization")
		if auth != "token env-token" {
			http.Error(w, "unexpected auth: "+auth, http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := github.PluginConfig{
		Owner:  "owner",
		Repo:   "repo",
		APIURL: srv.URL,
		// Token intentionally empty — should be resolved from GH_TOKEN
	}
	p := github.NewPlugin(cfg, noopLogger{})
	if err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{}); err != nil {
		t.Errorf("VerifyConditions() with env token failed: %v", err)
	}
}

func TestPlugin_VerifyConditions_NoToken(t *testing.T) {
	// Ensure no env vars bleed in from the test environment.
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("SEMANTIC_RELEASE_GITHUB_TOKEN", "")

	cfg := github.PluginConfig{Owner: "o", Repo: "r", APIURL: "http://unused"}
	p := github.NewPlugin(cfg, noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Fatal("expected error for missing token, got nil")
	}
}

func TestPlugin_VerifyConditions_NoOwnerRepo(t *testing.T) {
	cfg := github.PluginConfig{Token: "tok", APIURL: "http://unused"}
	p := github.NewPlugin(cfg, noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Fatal("expected error for missing owner/repo, got nil")
	}
}

func TestPlugin_VerifyConditions_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := github.NewPlugin(baseConfig(srv.URL), noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}

func TestPlugin_VerifyConditions_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	p := github.NewPlugin(baseConfig(srv.URL), noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
}

func TestPlugin_VerifyConditions_UnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := github.NewPlugin(baseConfig(srv.URL), noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}

func TestPlugin_VerifyConditions_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := github.NewPlugin(baseConfig(srv.URL), noopLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err != nil {
		t.Errorf("VerifyConditions() unexpected error: %v", err)
	}
}

func TestPlugin_Publish_NilProject(t *testing.T) {
	p := github.NewPlugin(baseConfig("http://unused"), noopLogger{})
	rc := &domain.ReleaseContext{CurrentProject: nil}
	result, err := p.Publish(context.Background(), rc)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestPlugin_Publish_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/releases") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			body, _ := json.Marshal(map[string]any{
				"id":       1,
				"html_url": "https://github.com/org/repo/releases/1",
				"tag_name": "v1.0.0",
			})
			_, _ = w.Write(body)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := github.NewPlugin(baseConfig(srv.URL), noopLogger{})
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
	if result.PublishURL != "https://github.com/org/repo/releases/1" {
		t.Errorf("PublishURL = %q, want %q", result.PublishURL, "https://github.com/org/repo/releases/1")
	}
}

func TestPlugin_Publish_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := github.NewPlugin(baseConfig(srv.URL), noopLogger{})
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

func TestPlugin_AddChannel_EmptyTag(t *testing.T) {
	// When TagName is empty no HTTP request should be made.
	p := github.NewPlugin(baseConfig("http://should-not-be-called"), noopLogger{})
	rc := &domain.ReleaseContext{TagName: ""}
	if err := p.AddChannel(context.Background(), rc); err != nil {
		t.Errorf("AddChannel() unexpected error: %v", err)
	}
}

func TestPlugin_AddChannel_ReleaseNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GET /repos/owner/repo/releases/tags/v1.0.0 → 404 means no release exists
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/releases/tags/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := github.NewPlugin(baseConfig(srv.URL), noopLogger{})
	rc := &domain.ReleaseContext{TagName: "v1.0.0"}
	if err := p.AddChannel(context.Background(), rc); err != nil {
		t.Errorf("AddChannel() unexpected error for missing release: %v", err)
	}
}

func TestPlugin_AddChannel_UpdateSuccess(t *testing.T) {
	release := map[string]any{
		"id":       42,
		"html_url": "https://github.com/org/repo/releases/42",
		"tag_name": "v1.0.0",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/releases/tags/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			body, _ := json.Marshal(release)
			_, _ = w.Write(body)
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/releases/42"):
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p := github.NewPlugin(baseConfig(srv.URL), noopLogger{})
	rc := &domain.ReleaseContext{TagName: "v1.0.0"}
	if err := p.AddChannel(context.Background(), rc); err != nil {
		t.Errorf("AddChannel() unexpected error: %v", err)
	}
}

func TestPlugin_AddChannel_UpdateFailure(t *testing.T) {
	release := map[string]any{
		"id":       42,
		"html_url": "https://github.com/org/repo/releases/42",
		"tag_name": "v1.0.0",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/releases/tags/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			body, _ := json.Marshal(release)
			_, _ = w.Write(body)
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/releases/"):
			w.WriteHeader(http.StatusBadRequest)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p := github.NewPlugin(baseConfig(srv.URL), noopLogger{})
	rc := &domain.ReleaseContext{TagName: "v1.0.0"}
	if err := p.AddChannel(context.Background(), rc); err == nil {
		t.Fatal("expected error for 400 PATCH response, got nil")
	}
}

func TestPlugin_Success_NilProject(t *testing.T) {
	p := github.NewPlugin(baseConfig("http://unused"), noopLogger{})
	rc := &domain.ReleaseContext{CurrentProject: nil}
	if err := p.Success(context.Background(), rc); err != nil {
		t.Errorf("Success() unexpected error: %v", err)
	}
}

func TestPlugin_Success_WithCommits(t *testing.T) {
	prs := []map[string]any{{"number": 7}}
	// Track which requests arrived so we can assert they happened.
	var gotGetCommitPRs, gotPostComment, gotPostLabels bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/commits/") && strings.Contains(r.URL.Path, "/pulls"):
			gotGetCommitPRs = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			body, _ := json.Marshal(prs)
			_, _ = w.Write(body)
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/7/comments"):
			gotPostComment = true
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/7/labels"):
			gotPostLabels = true
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p := github.NewPlugin(baseConfig(srv.URL), noopLogger{})
	rc := &domain.ReleaseContext{
		TagName: "v1.0.0",
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "myapp"},
			NextVersion: domain.NewVersion(1, 0, 0),
			Commits: []domain.Commit{
				{Hash: "abc123"},
			},
		},
		Result: &domain.ReleaseResult{
			Projects: []domain.ProjectReleaseResult{
				{Project: domain.Project{Name: "myapp"}, PublishURL: "https://github.com/org/repo/releases/1"},
			},
		},
	}

	if err := p.Success(context.Background(), rc); err != nil {
		t.Errorf("Success() unexpected error: %v", err)
	}
	if !gotGetCommitPRs {
		t.Error("expected GET commits/pulls request, did not see one")
	}
	if !gotPostComment {
		t.Error("expected POST comment request, did not see one")
	}
	if !gotPostLabels {
		t.Error("expected POST labels request, did not see one")
	}
}

func TestPlugin_Fail_NilError(t *testing.T) {
	p := github.NewPlugin(baseConfig("http://unused"), noopLogger{})
	rc := &domain.ReleaseContext{Error: nil}
	if err := p.Fail(context.Background(), rc); err != nil {
		t.Errorf("Fail() unexpected error: %v", err)
	}
}

func TestPlugin_Fail_FindsExistingIssue(t *testing.T) {
	existingIssues := []map[string]any{
		{"number": 5, "title": "The automated release is failing", "state": "open"},
	}
	var gotComment bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			body, _ := json.Marshal(existingIssues)
			_, _ = w.Write(body)
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues/5/comments"):
			gotComment = true
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p := github.NewPlugin(baseConfig(srv.URL), noopLogger{})
	rc := &domain.ReleaseContext{
		Branch: "main",
		Error:  errors.New("something went wrong"),
	}
	if err := p.Fail(context.Background(), rc); err != nil {
		t.Errorf("Fail() unexpected error: %v", err)
	}
	if !gotComment {
		t.Error("expected POST comment on existing issue, did not see one")
	}
}

func TestPlugin_Fail_CreatesNewIssue(t *testing.T) {
	var gotCreateIssue bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues"):
			// Return empty list — no existing issue.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("[]"))
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/issues"):
			gotCreateIssue = true
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p := github.NewPlugin(baseConfig(srv.URL), noopLogger{})
	rc := &domain.ReleaseContext{
		Branch: "main",
		Error:  errors.New("something went wrong"),
	}
	if err := p.Fail(context.Background(), rc); err != nil {
		t.Errorf("Fail() unexpected error: %v", err)
	}
	if !gotCreateIssue {
		t.Error("expected POST create issue request, did not see one")
	}
}
