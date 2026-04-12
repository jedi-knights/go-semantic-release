package plugins_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/plugins"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestExternalPlugin_Name(t *testing.T) {
	p := plugins.NewExternalPlugin("my-plugin", "/usr/bin/true")
	if p.Name() != "my-plugin" {
		t.Errorf("Name() = %q, want my-plugin", p.Name())
	}
}

// writeScript writes an executable shell script to a temp file and returns its path.
func writeScript(t *testing.T, content string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell script tests not supported on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+content+"\n"), 0o755); err != nil {
		t.Fatalf("writeScript: %v", err)
	}
	return path
}

// skipIfShellUnavailable skips the test when /bin/sh is not present.
func skipIfShellUnavailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
}

func TestExternalPlugin_VerifyConditions_NonExistentExecutable(t *testing.T) {
	p := plugins.NewExternalPlugin("ghost", "/nonexistent/binary/ghost-plugin")
	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Error("VerifyConditions() with non-existent executable should return error")
	}
}

func TestExternalPlugin_VerifyConditions_NoOutput(t *testing.T) {
	skipIfShellUnavailable(t)
	// A plugin that exits 0 with no stdout — treated as "step not implemented".
	script := writeScript(t, "exit 0")
	p := plugins.NewExternalPlugin("noop-plugin", script)

	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err != nil {
		t.Errorf("VerifyConditions() with empty-output plugin should not error, got %v", err)
	}
}

func TestExternalPlugin_VerifyConditions_NonZeroExit(t *testing.T) {
	skipIfShellUnavailable(t)
	script := writeScript(t, "echo 'condition failed' >&2; exit 1")
	p := plugins.NewExternalPlugin("failing-plugin", script)

	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Error("VerifyConditions() should return error on non-zero exit")
	}
}

func TestExternalPlugin_AnalyzeCommits_ReturnsMinor(t *testing.T) {
	skipIfShellUnavailable(t)
	resp, _ := json.Marshal(map[string]string{"release_type": "minor"})
	script := writeScript(t, "echo '"+string(resp)+"'")
	p := plugins.NewExternalPlugin("analyzer", script)

	rt, err := p.AnalyzeCommits(context.Background(), &domain.ReleaseContext{})
	if err != nil {
		t.Fatalf("AnalyzeCommits() error = %v", err)
	}
	if rt != domain.ReleaseMinor {
		t.Errorf("AnalyzeCommits() = %v, want ReleaseMinor", rt)
	}
}

func TestExternalPlugin_AnalyzeCommits_ReturnsMajor(t *testing.T) {
	skipIfShellUnavailable(t)
	resp, _ := json.Marshal(map[string]string{"release_type": "major"})
	script := writeScript(t, "echo '"+string(resp)+"'")
	p := plugins.NewExternalPlugin("analyzer", script)

	rt, err := p.AnalyzeCommits(context.Background(), &domain.ReleaseContext{})
	if err != nil {
		t.Fatalf("AnalyzeCommits() error = %v", err)
	}
	if rt != domain.ReleaseMajor {
		t.Errorf("AnalyzeCommits() = %v, want ReleaseMajor", rt)
	}
}

func TestExternalPlugin_AnalyzeCommits_UnknownTypeReturnsNone(t *testing.T) {
	skipIfShellUnavailable(t)
	resp, _ := json.Marshal(map[string]string{"release_type": "unknown"})
	script := writeScript(t, "echo '"+string(resp)+"'")
	p := plugins.NewExternalPlugin("analyzer", script)

	rt, err := p.AnalyzeCommits(context.Background(), &domain.ReleaseContext{})
	if err != nil {
		t.Fatalf("AnalyzeCommits() error = %v", err)
	}
	if rt != domain.ReleaseNone {
		t.Errorf("AnalyzeCommits() unknown type = %v, want ReleaseNone", rt)
	}
}

func TestExternalPlugin_GenerateNotes_ReturnsNotes(t *testing.T) {
	skipIfShellUnavailable(t)
	resp, _ := json.Marshal(map[string]string{"notes": "## v1.0.0"})
	script := writeScript(t, "echo '"+string(resp)+"'")
	p := plugins.NewExternalPlugin("notes-gen", script)

	notes, err := p.GenerateNotes(context.Background(), &domain.ReleaseContext{})
	if err != nil {
		t.Fatalf("GenerateNotes() error = %v", err)
	}
	if notes != "## v1.0.0" {
		t.Errorf("GenerateNotes() = %q, want ## v1.0.0", notes)
	}
}

func TestExternalPlugin_PluginReturnsErrorField(t *testing.T) {
	skipIfShellUnavailable(t)
	resp, _ := json.Marshal(map[string]string{"error": "plugin-level failure"})
	script := writeScript(t, "echo '"+string(resp)+"'")
	p := plugins.NewExternalPlugin("err-plugin", script)

	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Error("VerifyConditions() should propagate error from plugin response")
	}
}

func TestExternalPlugin_InvalidJSONResponse(t *testing.T) {
	skipIfShellUnavailable(t)
	script := writeScript(t, "echo 'not valid json'")
	p := plugins.NewExternalPlugin("bad-json", script)

	err := p.VerifyConditions(context.Background(), &domain.ReleaseContext{})
	if err == nil {
		t.Error("VerifyConditions() should return error on invalid JSON response")
	}
}
