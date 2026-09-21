package domain

import (
	"strings"
	"testing"
)

func TestRenderGitCommitMessage_EmptyTemplate(t *testing.T) {
	t.Parallel()
	got := RenderGitCommitMessage("", "svc/v1.0.0", NewVersion(1, 0, 0), "")
	want := "chore(release): svc/v1.0.0"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderGitCommitMessage_VersionPlaceholder(t *testing.T) {
	t.Parallel()
	got := RenderGitCommitMessage("chore(release): {{.Version}}", "svc/v1.2.3", NewVersion(1, 2, 3), "")
	want := "chore(release): 1.2.3"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderGitCommitMessage_TagPlaceholder(t *testing.T) {
	t.Parallel()
	got := RenderGitCommitMessage("release {{.Tag}}", "svc/v1.0.0", NewVersion(1, 0, 0), "")
	want := "release svc/v1.0.0"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRenderGitCommitMessage_NotesPlaceholder(t *testing.T) {
	t.Parallel()
	notes := "## 1.0.0\n\n- feat: something"
	got := RenderGitCommitMessage("{{.Notes}}", "svc/v1.0.0", NewVersion(1, 0, 0), notes)
	// text/template does not HTML-escape '>' — assert the raw content is preserved.
	if !strings.Contains(got, "feat: something") {
		t.Errorf("expected notes content in output, got %q", got)
	}
}

func TestRenderGitCommitMessage_AllPlaceholders(t *testing.T) {
	t.Parallel()
	tmpl := "chore(release): {{.Version}} [skip ci]\n\n{{.Tag}}"
	got := RenderGitCommitMessage(tmpl, "svc/v2.1.0", NewVersion(2, 1, 0), "notes body")
	if !strings.Contains(got, "2.1.0") {
		t.Errorf("expected version in output, got %q", got)
	}
	if !strings.Contains(got, "svc/v2.1.0") {
		t.Errorf("expected tag in output, got %q", got)
	}
	if !strings.Contains(got, "[skip ci]") {
		t.Errorf("expected static text in output, got %q", got)
	}
}

func TestRenderGitCommitMessage_InvalidTemplate_FallsBack(t *testing.T) {
	t.Parallel()
	// "{{.Invalid" is malformed — Parse returns an error → fallback.
	got := RenderGitCommitMessage("{{.Invalid", "svc/v1.0.0", NewVersion(1, 0, 0), "")
	want := "chore(release): svc/v1.0.0"
	if got != want {
		t.Errorf("expected fallback for invalid template, got %q", got)
	}
}

func TestRenderGitCommitMessage_ExecuteError_FallsBack(t *testing.T) {
	t.Parallel()
	// References an undefined named template — Execute returns an error → fallback.
	got := RenderGitCommitMessage(`{{template "nonexistent"}}`, "svc/v1.0.0", NewVersion(1, 0, 0), "")
	want := "chore(release): svc/v1.0.0"
	if got != want {
		t.Errorf("expected fallback for execute error, got %q", got)
	}
}

func TestRenderGitCommitMessage_TagContainsSlash(t *testing.T) {
	t.Parallel()
	// Slash in tag names is valid (monorepo prefix format).
	got := RenderGitCommitMessage("chore(release): {{.Tag}}", "my-svc/v3.0.0", NewVersion(3, 0, 0), "")
	want := "chore(release): my-svc/v3.0.0"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
