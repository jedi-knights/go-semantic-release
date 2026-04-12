package github

import (
	"context"
	"encoding/json"
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
	// Ensure the competing env vars are absent so GH_TOKEN wins the resolution loop.
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

	if err := p.uploadAsset(context.Background(), 1, filePath); err != nil {
		t.Errorf("uploadAsset: %v", err)
	}
}

func TestPlugin_UploadAsset_NonCreated(t *testing.T) {
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

	if err := p.uploadAsset(context.Background(), 1, filePath); err == nil {
		t.Fatal("expected error for non-201 response, got nil")
	}
}

func TestPlugin_UploadAsset_MissingFile(t *testing.T) {
	p := NewPlugin(pluginCfg(), testLogger{})
	// Pointing at a non-existent file should fail before any HTTP request.
	if err := p.uploadAsset(context.Background(), 1, "/nonexistent/path/asset.bin"); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestPlugin_UploadAssetGlob_NoMatches(t *testing.T) {
	// A glob that matches nothing must not make any HTTP calls and must return nil.
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	p := NewPlugin(pluginCfg(), testLogger{})
	p.client = &http.Client{Transport: redirectToServer(srv)}

	if err := p.uploadAssetGlob(context.Background(), 1, "/nonexistent/*.xyz"); err != nil {
		t.Errorf("uploadAssetGlob no-matches: %v", err)
	}
	if called {
		t.Error("expected no HTTP call when glob matches nothing")
	}
}

func TestPlugin_UploadAssetGlob_Success(t *testing.T) {
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

	if err := p.uploadAssetGlob(context.Background(), 1, filepath.Join(dir, "*")); err != nil {
		t.Errorf("uploadAssetGlob: %v", err)
	}
	if uploadCount != 2 {
		t.Errorf("expected 2 uploads, got %d", uploadCount)
	}
}
