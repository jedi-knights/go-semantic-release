package github

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// testLogger satisfies ports.Logger without producing output — defined here
// so that white-box tests can create Plugin values without importing the
// black-box test file's noopLogger.
type testLogger struct{}

func (testLogger) Debug(string, ...any) {}
func (testLogger) Info(string, ...any)  {}
func (testLogger) Warn(string, ...any)  {}
func (testLogger) Error(string, ...any) {}

// roundTripFunc is a function-typed http.RoundTripper used to intercept requests
// in tests without starting a real server on the default transport.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// redirectToServer returns a RoundTripper that rewrites every outgoing request
// so that its host and scheme point to srv, then delegates to http.DefaultTransport.
// This lets us exercise code that hardcodes https://api.github.com without real
// network access.
func redirectToServer(srv *httptest.Server) http.RoundTripper {
	return roundTripFunc(func(req *http.Request) (*http.Response, error) {
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		req.URL.Scheme = "http"
		return http.DefaultTransport.RoundTrip(req)
	})
}

func TestNewPublisher_TokenFromEnv(t *testing.T) {
	t.Setenv("GH_TOKEN", "env-token")
	// Setting to "" is equivalent to unsetting for resolveToken's `v != ""` guard.
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("SEMANTIC_RELEASE_GITHUB_TOKEN", "")

	p := NewPublisher("o", "r", "")
	if p.token != "env-token" {
		t.Errorf("token = %q, want %q", p.token, "env-token")
	}
}

func TestNewPublisher_ExplicitTokenTakesPrecedence(t *testing.T) {
	// Even when GH_TOKEN is set the explicit argument must win.
	t.Setenv("GH_TOKEN", "env-token")

	p := NewPublisher("o", "r", "explicit-token")
	if p.token != "explicit-token" {
		t.Errorf("token = %q, want %q", p.token, "explicit-token")
	}
}

func TestPublisher_Publish_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		body, _ := json.Marshal(map[string]any{
			"html_url": "https://github.com/o/r/releases/v1.0.0",
			"id":       1,
		})
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	p := NewPublisher("o", "r", "test-token")
	p.client = &http.Client{Transport: redirectToServer(srv)}

	params := ports.PublishParams{
		TagName: "v1.0.0",
		Version: domain.NewVersion(1, 0, 0),
	}
	result, err := p.Publish(context.Background(), params)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if !result.Published {
		t.Error("expected result.Published to be true")
	}
	if result.PublishURL != "https://github.com/o/r/releases/v1.0.0" {
		t.Errorf("PublishURL = %q, want %q", result.PublishURL, "https://github.com/o/r/releases/v1.0.0")
	}
}

func TestPublisher_Publish_NonCreated(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewPublisher("o", "r", "test-token")
	p.client = &http.Client{Transport: redirectToServer(srv)}

	params := ports.PublishParams{
		TagName: "v1.0.0",
		Version: domain.NewVersion(1, 0, 0),
	}
	_, err := p.Publish(context.Background(), params)
	if err == nil {
		t.Fatal("expected error for non-201 response, got nil")
	}
}

func TestPublisher_Publish_WithProject(t *testing.T) {
	t.Parallel()
	// When Project is set, the release name is "<project> <version>" instead of the tag.
	// The server should still receive a POST and return 201.
	var gotRequest bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRequest = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		body, _ := json.Marshal(map[string]any{
			"html_url": "https://github.com/o/r/releases/v2.0.0",
			"id":       2,
		})
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	p := NewPublisher("o", "r", "test-token")
	p.client = &http.Client{Transport: redirectToServer(srv)}

	params := ports.PublishParams{
		TagName: "v2.0.0",
		Version: domain.NewVersion(2, 0, 0),
		Project: "myapp",
	}
	result, err := p.Publish(context.Background(), params)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if !gotRequest {
		t.Error("expected the test server to receive a request")
	}
	if !result.Published {
		t.Error("expected result.Published to be true")
	}
}

// ---------------------------------------------------------------------------
// uploadAsset / uploadAssetGlob — Plugin white-box tests
// ---------------------------------------------------------------------------

// pluginCfg returns a minimal PluginConfig for upload tests.
func pluginCfg() PluginConfig {
	return PluginConfig{Owner: "o", Repo: "r", Token: "t"}
}

func TestPlugin_UploadAsset_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "asset.bin")
	if err := os.WriteFile(filePath, []byte("binary-content"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	if err := p.uploadAsset(context.Background(), 1, filePath, ""); err != nil {
		t.Errorf("uploadAsset: %v", err)
	}
}

func TestPlugin_UploadAsset_WithLabel_IncludesLabelInURL(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "asset.bin")
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	if err := p.uploadAsset(context.Background(), 1, filePath, "Source Tarballs"); err != nil {
		t.Fatalf("uploadAsset: %v", err)
	}
	if !strings.Contains(gotQuery, "label=Source+Tarballs") && !strings.Contains(gotQuery, "label=Source%20Tarballs") {
		t.Errorf("expected label in query string, got: %q", gotQuery)
	}
}

func TestPlugin_UploadAsset_WithoutLabel_NoLabelParam(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "asset.bin")
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	if err := p.uploadAsset(context.Background(), 1, filePath, ""); err != nil {
		t.Fatalf("uploadAsset: %v", err)
	}
	if strings.Contains(gotQuery, "label") {
		t.Errorf("expected no label param when label is empty, got: %q", gotQuery)
	}
}

func TestPlugin_UploadAsset_NonCreated(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "asset.bin")
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	if err := p.uploadAsset(context.Background(), 1, filePath, ""); err == nil {
		t.Fatal("expected error for non-201 response, got nil")
	}
}

func TestPlugin_UploadAsset_MissingFile(t *testing.T) {
	t.Parallel()
	p := NewPlugin(pluginCfg(), testLogger{})
	// Pointing at a non-existent file should fail before any HTTP request.
	if err := p.uploadAsset(context.Background(), 1, "/nonexistent/path/asset.bin", ""); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestPlugin_UploadAssetGlob_NoMatches(t *testing.T) {
	t.Parallel()
	// A glob that matches nothing must not make any HTTP calls and must return nil.
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	if err := p.uploadAssetGlob(context.Background(), 1, domain.GitHubAsset{Path: "/nonexistent/*.xyz"}); err != nil {
		t.Errorf("uploadAssetGlob no-matches: %v", err)
	}
	if called {
		t.Error("expected no HTTP call when glob matches nothing")
	}
}

func TestPlugin_UploadAssetGlob_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	for _, name := range []string{"lib.tar.gz", "checksums.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("data"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	var uploadCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		uploadCount++
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	if err := p.uploadAssetGlob(context.Background(), 1, domain.GitHubAsset{Path: filepath.Join(dir, "*")}); err != nil {
		t.Errorf("uploadAssetGlob: %v", err)
	}
	if uploadCount != 2 {
		t.Errorf("expected 2 uploads, got %d", uploadCount)
	}
}

func TestPlugin_UploadAssetGlob_PassesLabelToUpload(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "release.tar.gz")
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	asset := domain.GitHubAsset{Path: filepath.Join(dir, "*.tar.gz"), Label: "Release Tarball"}
	if err := p.uploadAssetGlob(context.Background(), 1, asset); err != nil {
		t.Fatalf("uploadAssetGlob: %v", err)
	}
	if !strings.Contains(gotQuery, "label=") {
		t.Errorf("expected label in query string from glob, got: %q", gotQuery)
	}
}

// ---------------------------------------------------------------------------
// errTransport — always returns a network error from RoundTrip.
// ---------------------------------------------------------------------------

type errTransport struct{}

func (errTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("simulated network error")
}

func pluginWithTransport(tr http.RoundTripper) *Plugin {
	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: tr}
	return p
}

// ---------------------------------------------------------------------------
// VerifyConditions — network-error paths
// ---------------------------------------------------------------------------

func TestPlugin_VerifyConditions_NetworkError(t *testing.T) {
	t.Parallel()
	p := pluginWithTransport(errTransport{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Fatal("expected error from network failure, got nil")
	}
	if !strings.Contains(err.Error(), "verifying GitHub access") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPlugin_VerifyConditions_InvalidAPIURL(t *testing.T) {
	t.Parallel()
	cfg := pluginCfg()
	cfg.APIURL = "://bad-url"
	p := NewPlugin(cfg, testLogger{})
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Fatal("expected error for invalid APIURL, got nil")
	}
	if !strings.Contains(err.Error(), "creating request") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ---------------------------------------------------------------------------
// createGHRelease
// ---------------------------------------------------------------------------

func TestPlugin_CreateGHRelease_NetworkError(t *testing.T) {
	t.Parallel()
	p := pluginWithTransport(errTransport{})
	_, err := p.createGHRelease(context.Background(), ghCreateReleaseRequest{TagName: "v1.0.0"})
	if err == nil {
		t.Fatal("expected error from network failure, got nil")
	}
	if !strings.Contains(err.Error(), "publishing release") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPlugin_CreateGHRelease_InvalidAPIURL(t *testing.T) {
	t.Parallel()
	cfg := pluginCfg()
	cfg.APIURL = "://bad-url"
	p := NewPlugin(cfg, testLogger{})
	_, err := p.createGHRelease(context.Background(), ghCreateReleaseRequest{TagName: "v1.0.0"})
	if err == nil {
		t.Fatal("expected error for invalid APIURL, got nil")
	}
	if !strings.Contains(err.Error(), "creating request") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPlugin_CreateGHRelease_InvalidJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	_, err := p.createGHRelease(context.Background(), ghCreateReleaseRequest{TagName: "v1.0.0"})
	if err == nil {
		t.Fatal("expected error for invalid JSON response, got nil")
	}
	if !strings.Contains(err.Error(), "decoding release response") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ---------------------------------------------------------------------------
// getReleaseByTag
// ---------------------------------------------------------------------------

func TestPlugin_GetReleaseByTag_NetworkError(t *testing.T) {
	t.Parallel()
	p := pluginWithTransport(errTransport{})
	_, err := p.getReleaseByTag(context.Background(), "v1.0.0")
	if err == nil {
		t.Fatal("expected error from network failure, got nil")
	}
}

func TestPlugin_GetReleaseByTag_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	_, err := p.getReleaseByTag(context.Background(), "v1.0.0")
	if err == nil {
		t.Fatal("expected error for unexpected status, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected status") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPlugin_GetReleaseByTag_InvalidJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	_, err := p.getReleaseByTag(context.Background(), "v1.0.0")
	if err == nil {
		t.Fatal("expected error for invalid JSON response, got nil")
	}
}

// ---------------------------------------------------------------------------
// uploadAssetGlob
// ---------------------------------------------------------------------------

func TestPlugin_UploadAssetGlob_BadPattern(t *testing.T) {
	t.Parallel()
	// An unclosed bracket is a syntax error in filepath.Glob.
	p := NewPlugin(pluginCfg(), testLogger{})
	err := p.uploadAssetGlob(context.Background(), 1, domain.GitHubAsset{Path: "[invalid"})
	if err == nil {
		t.Fatal("expected error for bad glob pattern, got nil")
	}
	if !strings.Contains(err.Error(), "globbing") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPlugin_UploadAssetGlob_UploadError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "artifact.bin")
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Upload endpoint returns 400 → uploadAsset errors → uploadAssetGlob propagates it.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	err := p.uploadAssetGlob(context.Background(), 1, domain.GitHubAsset{Path: filePath})
	if err == nil {
		t.Fatal("expected error when upload fails, got nil")
	}
}

// ---------------------------------------------------------------------------
// uploadAsset
// ---------------------------------------------------------------------------

func TestPlugin_UploadAsset_NetworkError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "asset.bin")
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := pluginWithTransport(errTransport{})
	if err := p.uploadAsset(context.Background(), 1, filePath, ""); err == nil {
		t.Fatal("expected error from network failure, got nil")
	}
}

func TestPlugin_UploadAsset_UnknownExtension_DefaultsMIMEType(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// .zzzz has no registered MIME type → falls back to application/octet-stream.
	filePath := filepath.Join(dir, "asset.zzzz")
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	if err := p.uploadAsset(context.Background(), 1, filePath, ""); err != nil {
		t.Errorf("uploadAsset: %v", err)
	}
	if gotContentType != "application/octet-stream" {
		t.Errorf("Content-Type = %q, want application/octet-stream", gotContentType)
	}
}

// ---------------------------------------------------------------------------
// getPRsForCommit
// ---------------------------------------------------------------------------

func TestPlugin_GetPRsForCommit_NetworkError(t *testing.T) {
	t.Parallel()
	p := pluginWithTransport(errTransport{})
	_, err := p.getPRsForCommit(context.Background(), "abc123")
	if err == nil {
		t.Fatal("expected error from network failure, got nil")
	}
}

func TestPlugin_GetPRsForCommit_InvalidAPIURL(t *testing.T) {
	t.Parallel()
	cfg := pluginCfg()
	cfg.APIURL = "://bad-url"
	p := NewPlugin(cfg, testLogger{})
	_, err := p.getPRsForCommit(context.Background(), "abc123")
	if err == nil {
		t.Fatal("expected error for invalid APIURL, got nil")
	}
}

func TestPlugin_GetPRsForCommit_NonOK(t *testing.T) {
	t.Parallel()
	// Non-200 status returns nil, nil — not an error — so Success can continue.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	prs, err := p.getPRsForCommit(context.Background(), "abc123")
	if err != nil {
		t.Errorf("getPRsForCommit() non-200 should return nil error, got: %v", err)
	}
	if prs != nil {
		t.Errorf("getPRsForCommit() non-200 should return nil PRs, got: %v", prs)
	}
}

func TestPlugin_GetPRsForCommit_InvalidJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	_, err := p.getPRsForCommit(context.Background(), "abc123")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// ---------------------------------------------------------------------------
// commentOnIssue
// ---------------------------------------------------------------------------

func TestPlugin_CommentOnIssue_NetworkError(t *testing.T) {
	t.Parallel()
	p := pluginWithTransport(errTransport{})
	if err := p.commentOnIssue(context.Background(), 1, "body"); err == nil {
		t.Fatal("expected error from network failure, got nil")
	}
}

func TestPlugin_CommentOnIssue_InvalidAPIURL(t *testing.T) {
	t.Parallel()
	cfg := pluginCfg()
	cfg.APIURL = "://bad-url"
	p := NewPlugin(cfg, testLogger{})
	if err := p.commentOnIssue(context.Background(), 1, "body"); err == nil {
		t.Fatal("expected error for invalid APIURL, got nil")
	}
}

func TestPlugin_CommentOnIssue_NonCreated(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	err := p.commentOnIssue(context.Background(), 7, "comment body")
	if err == nil {
		t.Fatal("expected error for non-201 response, got nil")
	}
	if !strings.Contains(err.Error(), "comment failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ---------------------------------------------------------------------------
// addLabelsToIssue
// ---------------------------------------------------------------------------

func TestPlugin_AddLabelsToIssue_EmptyLabels(t *testing.T) {
	t.Parallel()
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	p.addLabelsToIssue(context.Background(), 1, []string{})
	if called {
		t.Error("expected no HTTP call for empty labels, got one")
	}
}

func TestPlugin_AddLabelsToIssue_InvalidAPIURL(t *testing.T) {
	t.Parallel()
	cfg := pluginCfg()
	cfg.APIURL = "://bad-url"
	p := NewPlugin(cfg, testLogger{})
	// Errors are silently discarded — verify no panic.
	p.addLabelsToIssue(context.Background(), 1, []string{"released"})
}

func TestPlugin_AddLabelsToIssue_NetworkError(t *testing.T) {
	t.Parallel()
	p := pluginWithTransport(errTransport{})
	// Errors are silently discarded — verify no panic.
	p.addLabelsToIssue(context.Background(), 1, []string{"released"})
}

// ---------------------------------------------------------------------------
// findFailureIssue
// ---------------------------------------------------------------------------

func TestPlugin_FindFailureIssue_NetworkError(t *testing.T) {
	t.Parallel()
	p := pluginWithTransport(errTransport{})
	_, err := p.findFailureIssue(context.Background(), "The automated release is failing")
	if err == nil {
		t.Fatal("expected error from network failure, got nil")
	}
}

func TestPlugin_FindFailureIssue_InvalidAPIURL(t *testing.T) {
	t.Parallel()
	cfg := pluginCfg()
	cfg.APIURL = "://bad-url"
	p := NewPlugin(cfg, testLogger{})
	_, err := p.findFailureIssue(context.Background(), "The automated release is failing")
	if err == nil {
		t.Fatal("expected error for invalid APIURL, got nil")
	}
}

func TestPlugin_FindFailureIssue_NonOK(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	_, err := p.findFailureIssue(context.Background(), "The automated release is failing")
	if err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
	if !strings.Contains(err.Error(), "listing issues failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPlugin_FindFailureIssue_InvalidJSON(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json-array"))
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	_, err := p.findFailureIssue(context.Background(), "The automated release is failing")
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// ---------------------------------------------------------------------------
// createIssue
// ---------------------------------------------------------------------------

func TestPlugin_CreateIssue_NetworkError(t *testing.T) {
	t.Parallel()
	p := pluginWithTransport(errTransport{})
	if err := p.createIssue(context.Background(), "title", "body", []string{"bug"}); err == nil {
		t.Fatal("expected error from network failure, got nil")
	}
}

func TestPlugin_CreateIssue_InvalidAPIURL(t *testing.T) {
	t.Parallel()
	cfg := pluginCfg()
	cfg.APIURL = "://bad-url"
	p := NewPlugin(cfg, testLogger{})
	if err := p.createIssue(context.Background(), "title", "body", []string{"bug"}); err == nil {
		t.Fatal("expected error for invalid APIURL, got nil")
	}
}

func TestPlugin_CreateIssue_NonCreated(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte("validation failed"))
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	err := p.createIssue(context.Background(), "title", "body", []string{"bug"})
	if err == nil {
		t.Fatal("expected error for non-201 response, got nil")
	}
	if !strings.Contains(err.Error(), "create issue failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Publish — asset upload path (lines 154-157)
// ---------------------------------------------------------------------------

func TestPlugin_Publish_AssetUploadFailureIsLogged(t *testing.T) {
	// When an asset upload fails, Publish logs a Warn and still returns the result.
	t.Parallel()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "artifact.bin")
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	var releaseCreated bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Release creation: POST /repos/.../releases (no /assets suffix).
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/releases") && !strings.Contains(r.URL.Path, "/assets") {
			releaseCreated = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			body, _ := json.Marshal(map[string]any{"id": 1, "html_url": "https://github.com/o/r/releases/1"})
			_, _ = w.Write(body)
			return
		}
		// Asset upload (redirected from uploads.github.com via redirectToServer): return 400.
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	cfg := pluginCfg()
	cfg.Assets = []domain.GitHubAsset{{Path: filePath}}
	p := NewPlugin(cfg, testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	rc := &domain.ReleaseContext{
		TagName: "v1.0.0",
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "myapp"},
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}
	result, err := p.Publish(context.Background(), rc)
	if err != nil {
		t.Fatalf("Publish() should succeed even when asset upload fails, got: %v", err)
	}
	if !releaseCreated {
		t.Error("expected release to be created")
	}
	if result == nil || !result.Published {
		t.Error("expected Published=true")
	}
}

// ---------------------------------------------------------------------------
// Success — error paths inside the commit loop (lines 243-250)
// ---------------------------------------------------------------------------

func TestPlugin_Success_GetPRsError(t *testing.T) {
	// A transport that errors for /commits/ requests makes getPRsForCommit return an
	// error, exercising the debug-and-continue path in Success.
	t.Parallel()
	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/commits/") {
			return nil, errors.New("network error for commits endpoint")
		}
		return nil, errors.New("unexpected request in test")
	})}

	rc := &domain.ReleaseContext{
		TagName: "v1.0.0",
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "myapp"},
			NextVersion: domain.NewVersion(1, 0, 0),
			Commits:     []domain.Commit{{Hash: "abc123"}},
		},
		Result: &domain.ReleaseResult{
			Projects: []domain.ProjectReleaseResult{
				{Project: domain.Project{Name: "myapp"}, PublishURL: "https://github.com/o/r/releases/1"},
			},
		},
	}
	if err := p.Success(context.Background(), rc); err != nil {
		t.Errorf("Success() should return nil even when getPRsForCommit fails, got: %v", err)
	}
}

func TestPlugin_Success_CommentFails(t *testing.T) {
	// commentOnIssue returns non-201; Success logs the error and continues.
	t.Parallel()
	prs := []ghPR{{Number: 7}}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/commits/") && strings.Contains(r.URL.Path, "/pulls"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			body, _ := json.Marshal(prs)
			_, _ = w.Write(body)
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/comments"):
			w.WriteHeader(http.StatusForbidden)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	rc := &domain.ReleaseContext{
		TagName: "v1.0.0",
		CurrentProject: &domain.ProjectReleasePlan{
			Project:     domain.Project{Name: "myapp"},
			NextVersion: domain.NewVersion(1, 0, 0),
			Commits:     []domain.Commit{{Hash: "abc123"}},
		},
		Result: &domain.ReleaseResult{
			Projects: []domain.ProjectReleaseResult{
				{Project: domain.Project{Name: "myapp"}, PublishURL: "https://github.com/o/r/releases/1"},
			},
		},
	}
	if err := p.Success(context.Background(), rc); err != nil {
		t.Errorf("Success() should return nil even when comment fails, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// getReleaseByTag — NewRequestWithContext error path
// ---------------------------------------------------------------------------

func TestPlugin_GetReleaseByTag_InvalidAPIURL(t *testing.T) {
	t.Parallel()
	cfg := pluginCfg()
	cfg.APIURL = "://bad-url"
	p := NewPlugin(cfg, testLogger{})
	_, err := p.getReleaseByTag(context.Background(), "v1.0.0")
	if err == nil {
		t.Fatal("expected error for invalid APIURL, got nil")
	}
}

// ---------------------------------------------------------------------------
// AddChannel — Do error on the PATCH update request (line 204-206)
// ---------------------------------------------------------------------------

func TestPlugin_AddChannel_PatchNetworkError(t *testing.T) {
	// getReleaseByTag succeeds (GET returns a valid release), but the PATCH update
	// fails at the network level, exercising the Do-error path in AddChannel.
	t.Parallel()
	release := map[string]any{"id": 42, "html_url": "https://github.com/o/r/releases/42", "tag_name": "v1.0.0"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/releases/tags/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			body, _ := json.Marshal(release)
			_, _ = w.Write(body)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method == http.MethodPatch {
			return nil, errors.New("simulated network error on PATCH")
		}
		req.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		req.URL.Scheme = "http"
		return http.DefaultTransport.RoundTrip(req)
	})}

	rc := &domain.ReleaseContext{TagName: "v1.0.0"}
	err := p.AddChannel(context.Background(), rc)
	if err == nil {
		t.Fatal("expected error for network error on PATCH, got nil")
	}
	if !strings.Contains(err.Error(), "updating release") {
		t.Errorf("expected 'updating release' in error, got: %v", err)
	}
}
