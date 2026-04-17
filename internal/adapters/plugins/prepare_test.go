// Package plugins_test uses go.uber.org/mock generated mocks rather than hand-written
// port fakes. This is an intentional exception to the project's general preference for
// fakes: the port interfaces here have many methods, and maintaining full hand-written
// fakes would add significant ongoing cost with little benefit at this layer.
package plugins_test

import (
	"context"
	"errors"
	"io/fs"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/plugins"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports/mocks"
)

// assertChangelogHeader is a shared helper that verifies the written changelog data
// starts with the expected header. t.Helper() ensures failures point to the caller.
func assertChangelogHeader(t *testing.T, data []byte) {
	t.Helper()
	if !strings.HasPrefix(string(data), "# Changelog") {
		t.Errorf("expected changelog to start with '# Changelog', got: %q", string(data))
	}
}

func TestPreparePlugin_Name(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)
	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{})
	if got := plugin.Name(); got != "prepare-files" {
		t.Errorf("Name() = %q, want %q", got, "prepare-files")
	}
}

func TestPreparePlugin_UpdateVersionFile(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	mockFS.EXPECT().WriteFile(
		"/repo/VERSION",
		[]byte("2.0.0\n"),
		fs.FileMode(0o644),
	).Return(nil)

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		VersionFile: "VERSION",
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(2, 0, 0),
		},
	}

	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
}

func TestPreparePlugin_UpdateChangelog(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	// Existing changelog.
	mockFS.EXPECT().Exists("/repo/CHANGELOG.md").Return(true)
	mockFS.EXPECT().ReadFile("/repo/CHANGELOG.md").Return(
		[]byte("# Changelog\n\n## 1.0.0\n\n- Initial release\n"), nil)

	mockFS.EXPECT().WriteFile(
		"/repo/CHANGELOG.md",
		gomock.Any(),
		fs.FileMode(0o644),
	).DoAndReturn(func(_ string, data []byte, _ fs.FileMode) error {
		assertChangelogHeader(t, data)
		s := string(data)
		if !strings.Contains(s, "## 2.0.0") {
			t.Errorf("expected new entry '## 2.0.0' in changelog, got: %q", s)
		}
		if !strings.Contains(s, "## 1.0.0") {
			t.Errorf("expected old entry '## 1.0.0' still present in changelog, got: %q", s)
		}
		return nil
	})

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		ChangelogFile: "CHANGELOG.md",
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		Notes:          "## 2.0.0\n\n### Features\n- new stuff",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(2, 0, 0),
		},
	}

	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
}

func TestPreparePlugin_NewChangelog(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	mockFS.EXPECT().Exists("/repo/CHANGELOG.md").Return(false)
	mockFS.EXPECT().WriteFile(
		"/repo/CHANGELOG.md",
		gomock.Any(),
		fs.FileMode(0o644),
	).DoAndReturn(func(_ string, data []byte, _ fs.FileMode) error {
		assertChangelogHeader(t, data)
		return nil
	})

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		ChangelogFile: "CHANGELOG.md",
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		Notes:          "## 1.0.0\n\n- first release",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}

	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
}

func TestPreparePlugin_ProjectChangelogFile(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	// Per-project changelog resolves to repoRoot/project.Path/changelog_file.
	mockFS.EXPECT().Exists("/repo/services/auth-server/CHANGELOG.md").Return(false)
	mockFS.EXPECT().WriteFile(
		"/repo/services/auth-server/CHANGELOG.md",
		gomock.Any(),
		fs.FileMode(0o644),
	).DoAndReturn(func(_ string, data []byte, _ fs.FileMode) error {
		assertChangelogHeader(t, data)
		return nil
	})

	// No global changelog_file configured — only per-project should be used.
	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		Notes:          "## 1.0.0\n\n- initial release",
		CurrentProject: &domain.ProjectReleasePlan{
			Project: domain.Project{
				Name:          "auth-server",
				Path:          "services/auth-server",
				ChangelogFile: "CHANGELOG.md",
			},
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}

	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
}

func TestPreparePlugin_ProjectChangelogOverridesGlobal(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	// The per-project path should win; the global root path must never be touched.
	mockFS.EXPECT().Exists("/repo/services/worker/CHANGELOG.md").Return(false)
	mockFS.EXPECT().WriteFile(
		"/repo/services/worker/CHANGELOG.md",
		gomock.Any(),
		fs.FileMode(0o644),
	).Return(nil)

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		ChangelogFile: "CHANGELOG.md", // global — must not be written
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		Notes:          "## 2.0.0\n\n- breaking change",
		CurrentProject: &domain.ProjectReleasePlan{
			Project: domain.Project{
				Name:          "worker",
				Path:          "services/worker",
				ChangelogFile: "CHANGELOG.md", // per-project wins
			},
			NextVersion: domain.NewVersion(2, 0, 0),
		},
	}

	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
}

func TestPreparePlugin_GlobalChangelogWhenNoProjectOverride(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	// No per-project changelog_file — global path at repo root is used.
	mockFS.EXPECT().Exists("/repo/CHANGELOG.md").Return(false)
	mockFS.EXPECT().WriteFile(
		"/repo/CHANGELOG.md",
		gomock.Any(),
		fs.FileMode(0o644),
	).Return(nil)

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		ChangelogFile: "CHANGELOG.md",
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		Notes:          "## 1.1.0\n\n- patch",
		CurrentProject: &domain.ProjectReleasePlan{
			Project: domain.Project{
				Name: "api",
				Path: "services/api",
				// ChangelogFile intentionally empty — no per-project override
			},
			NextVersion: domain.NewVersion(1, 1, 0),
		},
	}

	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
}

func TestPreparePlugin_NilProject(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		VersionFile:   "VERSION",
		ChangelogFile: "CHANGELOG.md",
	})

	rc := &domain.ReleaseContext{CurrentProject: nil}

	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() with nil project should not error, got: %v", err)
	}
}

func TestPreparePlugin_NoChangelogConfigured(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)
	// No filesystem calls expected — gomock will fail the test if any occur.

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		Notes:          "## 1.0.0\n\n- something",
		CurrentProject: &domain.ProjectReleasePlan{
			Project: domain.Project{
				Name: "api",
				Path: "services/api",
				// ChangelogFile intentionally empty
			},
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}

	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() with no changelog config should not error, got: %v", err)
	}
}

func TestPreparePlugin_PathTraversal(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		ChangelogFile: "../../etc/passwd",
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		Notes:          "## 1.0.0",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}

	err := plugin.Prepare(context.Background(), rc)
	if err == nil {
		t.Fatal("expected error for path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "escapes repository root") {
		t.Errorf("expected traversal guard error, got: %v", err)
	}
}

// TestPreparePlugin_ProjectPathTraversal verifies that a traversal attack via
// a per-project ChangelogFile (rather than the global PrepareConfig.ChangelogFile)
// is also caught by the path-guard in updateChangelog.
func TestPreparePlugin_ProjectPathTraversal(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)
	// No filesystem calls expected — the guard must reject before any I/O.

	// No global changelog configured; only the per-project override is set.
	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		Notes:          "## 1.0.0",
		CurrentProject: &domain.ProjectReleasePlan{
			Project: domain.Project{
				Name: "evil",
				// Path has two components (services/evil), so three "../" are
				// needed to escape past /repo: evil→services→/repo→/
				Path:          "services/evil",
				ChangelogFile: "../../../etc/passwd",
			},
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}

	err := plugin.Prepare(context.Background(), rc)
	if err == nil {
		t.Fatal("expected error for per-project path traversal, got nil")
	}
	if !strings.Contains(err.Error(), "escapes repository root") {
		t.Errorf("expected traversal guard error, got: %v", err)
	}
}

func TestPreparePlugin_WriteVersionFileError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	writeErr := errors.New("disk full")
	mockFS.EXPECT().WriteFile("/repo/VERSION", gomock.Any(), fs.FileMode(0o644)).Return(writeErr)

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		VersionFile: "VERSION",
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}

	err := plugin.Prepare(context.Background(), rc)
	if err == nil {
		t.Fatal("expected error from WriteFile, got nil")
	}
	if !strings.Contains(err.Error(), "writing version file") {
		t.Errorf("expected wrapped version-file error, got: %v", err)
	}
}

func TestPreparePlugin_WriteChangelogError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	writeErr := errors.New("permission denied")
	mockFS.EXPECT().Exists("/repo/CHANGELOG.md").Return(false)
	mockFS.EXPECT().WriteFile("/repo/CHANGELOG.md", gomock.Any(), fs.FileMode(0o644)).Return(writeErr)

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		ChangelogFile: "CHANGELOG.md",
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		Notes:          "## 1.0.0\n\n- something",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}

	err := plugin.Prepare(context.Background(), rc)
	if err == nil {
		t.Fatal("expected error from WriteFile, got nil")
	}
	if !strings.Contains(err.Error(), "writing changelog") {
		t.Errorf("expected wrapped changelog error, got: %v", err)
	}
}

func TestPreparePlugin_EmptyNotesSkipsChangelogWrite(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)
	// No filesystem calls expected when Notes is empty — gomock will fail if any occur.

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		ChangelogFile: "CHANGELOG.md",
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		Notes:          "", // empty — changelog write must be skipped
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}

	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() with empty Notes should not error, got: %v", err)
	}
}

func TestPreparePlugin_NoBlankLineAccumulationOnRepeatedPrepare(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	// Track what was written so the second call can read it back.
	var written []byte

	// First call: changelog does not yet exist.
	mockFS.EXPECT().Exists("/repo/CHANGELOG.md").Return(false)
	mockFS.EXPECT().WriteFile(
		"/repo/CHANGELOG.md",
		gomock.Any(),
		fs.FileMode(0o644),
	).DoAndReturn(func(_ string, data []byte, _ fs.FileMode) error {
		written = data
		return nil
	})

	// Second call: changelog now exists with what was written above.
	mockFS.EXPECT().Exists("/repo/CHANGELOG.md").Return(true)
	mockFS.EXPECT().ReadFile("/repo/CHANGELOG.md").DoAndReturn(func(_ string) ([]byte, error) {
		return written, nil
	})
	mockFS.EXPECT().WriteFile(
		"/repo/CHANGELOG.md",
		gomock.Any(),
		fs.FileMode(0o644),
	).DoAndReturn(func(_ string, data []byte, _ fs.FileMode) error {
		// Structural check: split on lines, locate the title, verify exactly one
		// blank line separates it from the first entry. A triple-newline check
		// would produce a false positive if the notes body itself contained one.
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "# ") {
				// Line after title must be blank; the entry must follow on the next line.
				if i+1 >= len(lines) || lines[i+1] != "" {
					t.Errorf("expected blank line after title, got line[%d]=%q", i+1, lines[i+1])
				}
				if i+2 >= len(lines) || lines[i+2] == "" {
					t.Errorf("expected entry immediately after blank line, got blank at line[%d]", i+2)
				}
				break
			}
		}
		return nil
	})

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		ChangelogFile: "CHANGELOG.md",
	})

	calls := []struct {
		notes   string
		version domain.Version
	}{
		{"## 2.0.0\n\n### Features\n- new feature", domain.NewVersion(2, 0, 0)},
		{"## 3.0.0\n\n### Features\n- another feature", domain.NewVersion(3, 0, 0)},
	}
	for i, c := range calls {
		rc := &domain.ReleaseContext{
			RepositoryRoot: "/repo",
			Notes:          c.notes,
			CurrentProject: &domain.ProjectReleasePlan{
				NextVersion: c.version,
			},
		}
		if err := plugin.Prepare(context.Background(), rc); err != nil {
			t.Fatalf("Prepare() call %d error = %v", i+1, err)
		}
	}
}

func TestPreparePlugin_RelativeRepositoryRoot(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)
	// VersionFile is intentionally absent; the IsAbs guard lives in updateChangelog,
	// so the error originates there (updateVersionFile returns early on empty versionFile).

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		ChangelogFile: "CHANGELOG.md",
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "relative/path", // not absolute — must be rejected
		Notes:          "## 1.0.0",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}

	err := plugin.Prepare(context.Background(), rc)
	if err == nil {
		t.Fatal("expected error for relative RepositoryRoot, got nil")
	}
	if !strings.Contains(err.Error(), "RepositoryRoot must be an absolute path") {
		t.Errorf("expected absolute-path error, got: %v", err)
	}
}

func TestPreparePlugin_RunsCommandDuringPrepare(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	var capturedCmd string
	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		Command: "uv lock",
	}, plugins.WithCommandRunner(func(_ context.Context, cmd string, _ domain.Version) error {
		capturedCmd = cmd
		return nil
	}))

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}

	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if capturedCmd != "uv lock" {
		t.Errorf("capturedCmd = %q, want %q", capturedCmd, "uv lock")
	}
}

func TestPreparePlugin_CommandErrorPropagates(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		Command: "fail",
	}, plugins.WithCommandRunner(func(_ context.Context, _ string, _ domain.Version) error {
		return errors.New("command failed with exit status 1")
	}))

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}

	err := plugin.Prepare(context.Background(), rc)
	if err == nil {
		t.Fatal("expected error when command fails, got nil")
	}
	if !strings.Contains(err.Error(), "prepare command failed") {
		t.Errorf("expected 'prepare command failed' in error, got: %v", err)
	}
}

func TestPreparePlugin_SkipsCommandWhenEmpty(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	called := false
	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		Command: "", // no command configured
	}, plugins.WithCommandRunner(func(_ context.Context, _ string, _ domain.Version) error {
		called = true
		return nil
	}))

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}

	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if called {
		t.Error("command runner should not be called when Command is empty")
	}
}

func TestPreparePlugin_UpdatesTOMLVersionFile(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	pyproject := []byte("[tool.poetry]\nname = \"myproject\"\nversion = \"1.0.0\"\n")

	mockFS.EXPECT().ReadFile("/repo/pyproject.toml").Return(pyproject, nil)
	mockFS.EXPECT().WriteFile(
		"/repo/pyproject.toml",
		gomock.Any(),
		fs.FileMode(0o644),
	).DoAndReturn(func(_ string, data []byte, _ fs.FileMode) error {
		if !strings.Contains(string(data), `version = "2.0.0"`) {
			t.Errorf("expected version = \"2.0.0\" in updated file, got:\n%s", data)
		}
		if strings.Contains(string(data), `version = "1.0.0"`) {
			t.Error("old version should be replaced")
		}
		return nil
	})

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		VersionFiles: []string{"pyproject.toml:tool.poetry.version"},
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(2, 0, 0),
		},
	}

	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
}

func TestPreparePlugin_VersionFilesReadError(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	mockFS.EXPECT().ReadFile("/repo/pyproject.toml").Return(nil, errors.New("file not found"))

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		VersionFiles: []string{"pyproject.toml:tool.poetry.version"},
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(1, 0, 0),
		},
	}

	err := plugin.Prepare(context.Background(), rc)
	if err == nil {
		t.Fatal("expected error when file cannot be read")
	}
}

func TestPreparePlugin_MultipleVersionFiles(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	mockFS := mocks.NewMockFileSystem(ctrl)

	pyproject := []byte("[tool.poetry]\nversion = \"1.0.0\"\n")

	mockFS.EXPECT().ReadFile("/repo/pyproject.toml").Return(pyproject, nil)
	mockFS.EXPECT().WriteFile("/repo/pyproject.toml", gomock.Any(), fs.FileMode(0o644)).Return(nil)
	mockFS.EXPECT().WriteFile("/repo/VERSION", []byte("2.0.0\n"), fs.FileMode(0o644)).Return(nil)

	plugin := plugins.NewPreparePlugin(mockFS, noopLogger{}, domain.PrepareConfig{
		VersionFiles: []string{
			"pyproject.toml:tool.poetry.version",
			"VERSION",
		},
	})

	rc := &domain.ReleaseContext{
		RepositoryRoot: "/repo",
		CurrentProject: &domain.ProjectReleasePlan{
			NextVersion: domain.NewVersion(2, 0, 0),
		},
	}

	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
}
