package plugins_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	adapterfs "github.com/jedi-knights/go-semantic-release/internal/adapters/fs"
	"github.com/jedi-knights/go-semantic-release/internal/adapters/plugins"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func writeRepoFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readRepoFile(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestPreparePlugin_Cargo_Workspace(t *testing.T) {
	dir := t.TempDir()
	writeRepoFile(t, dir, "Cargo.toml", `[workspace]
members = [
    "crates/vcl-core",
    "crates/vcl-cli",
]

[workspace.package]
version = "0.1.0"
edition = "2021"
`)
	writeRepoFile(t, dir, "crates/vcl-core/Cargo.toml", "[package]\nname = \"vcl-core\"\nversion.workspace = true\n")
	writeRepoFile(t, dir, "crates/vcl-cli/Cargo.toml", "[package]\nname = \"vcl-cli\"\nversion.workspace = true\n")
	writeRepoFile(t, dir, "Cargo.lock", `version = 4

[[package]]
name = "anyhow"
version = "1.0.86"

[[package]]
name = "vcl-core"
version = "0.1.0"

[[package]]
name = "vcl-cli"
version = "0.1.0"
`)

	plugin := plugins.NewPreparePlugin(
		adapterfs.NewOSFileSystem(),
		noopLogger{},
		domain.PrepareConfig{},
		plugins.WithCargo(true),
	)
	rc := &domain.ReleaseContext{
		RepositoryRoot: dir,
		CurrentProject: &domain.ProjectReleasePlan{NextVersion: domain.NewVersion(0, 2, 0)},
	}
	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}

	toml := readRepoFile(t, dir, "Cargo.toml")
	if !strings.Contains(toml, `version = "0.2.0"`) {
		t.Errorf("Cargo.toml workspace version not bumped:\n%s", toml)
	}
	// The [workspace.package] edition and comment structure must be preserved.
	if !strings.Contains(toml, `edition = "2021"`) {
		t.Errorf("Cargo.toml lost surrounding content:\n%s", toml)
	}

	lock := readRepoFile(t, dir, "Cargo.lock")
	if strings.Count(lock, `version = "0.2.0"`) != 2 {
		t.Errorf("expected both local crates bumped in Cargo.lock:\n%s", lock)
	}
	if !strings.Contains(lock, `version = "1.0.86"`) {
		t.Errorf("third-party crate version must be untouched:\n%s", lock)
	}
}

func TestPreparePlugin_Cargo_DisabledByDefault(t *testing.T) {
	// Without WithCargo, a Cargo.toml present in the repo must be left untouched.
	dir := t.TempDir()
	writeRepoFile(t, dir, "Cargo.toml", "[package]\nname = \"x\"\nversion = \"0.1.0\"\n")

	plugin := plugins.NewPreparePlugin(
		adapterfs.NewOSFileSystem(),
		noopLogger{},
		domain.PrepareConfig{},
	)
	rc := &domain.ReleaseContext{
		RepositoryRoot: dir,
		CurrentProject: &domain.ProjectReleasePlan{NextVersion: domain.NewVersion(0, 2, 0)},
	}
	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if got := readRepoFile(t, dir, "Cargo.toml"); !strings.Contains(got, `version = "0.1.0"`) {
		t.Errorf("Cargo.toml should be untouched when cargo disabled:\n%s", got)
	}
}

func TestPreparePlugin_Cargo_DryRunDoesNotMutate(t *testing.T) {
	dir := t.TempDir()
	const manifest = "[package]\nname = \"x\"\nversion = \"0.1.0\"\n"
	writeRepoFile(t, dir, "Cargo.toml", manifest)
	writeRepoFile(t, dir, "Cargo.lock", "[[package]]\nname = \"x\"\nversion = \"0.1.0\"\n")

	plugin := plugins.NewPreparePlugin(
		adapterfs.NewOSFileSystem(),
		noopLogger{},
		domain.PrepareConfig{},
		plugins.WithCargo(true),
	)
	rc := &domain.ReleaseContext{
		RepositoryRoot: dir,
		DryRun:         true,
		CurrentProject: &domain.ProjectReleasePlan{NextVersion: domain.NewVersion(0, 2, 0)},
	}
	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if got := readRepoFile(t, dir, "Cargo.toml"); got != manifest {
		t.Errorf("dry-run mutated Cargo.toml:\n%s", got)
	}
}

func TestPreparePlugin_Cargo_VersionFilesTakePrecedence(t *testing.T) {
	// When the user explicitly lists Cargo.toml in version_files, the cargo step
	// must not double-process the manifest, but it must still refresh Cargo.lock.
	dir := t.TempDir()
	writeRepoFile(t, dir, "Cargo.toml", "[package]\nname = \"plug-audit\"\nversion = \"0.1.0\"\n")
	writeRepoFile(t, dir, "Cargo.lock", "[[package]]\nname = \"plug-audit\"\nversion = \"0.1.0\"\n")

	plugin := plugins.NewPreparePlugin(
		adapterfs.NewOSFileSystem(),
		noopLogger{},
		domain.PrepareConfig{VersionFiles: []string{"Cargo.toml:package.version"}},
		plugins.WithCargo(true),
	)
	rc := &domain.ReleaseContext{
		RepositoryRoot: dir,
		CurrentProject: &domain.ProjectReleasePlan{NextVersion: domain.NewVersion(1, 0, 0)},
	}
	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if got := readRepoFile(t, dir, "Cargo.toml"); !strings.Contains(got, `version = "1.0.0"`) {
		t.Errorf("Cargo.toml not bumped:\n%s", got)
	}
	if got := readRepoFile(t, dir, "Cargo.lock"); !strings.Contains(got, `version = "1.0.0"`) {
		t.Errorf("Cargo.lock not refreshed despite explicit version_files:\n%s", got)
	}
}

func TestPreparePlugin_Cargo_SingleCrate(t *testing.T) {
	dir := t.TempDir()
	writeRepoFile(t, dir, "Cargo.toml", "[package]\nname = \"plug-audit\"\nversion = \"0.1.0\"\n")

	plugin := plugins.NewPreparePlugin(
		adapterfs.NewOSFileSystem(),
		noopLogger{},
		domain.PrepareConfig{},
		plugins.WithCargo(true),
	)
	rc := &domain.ReleaseContext{
		RepositoryRoot: dir,
		CurrentProject: &domain.ProjectReleasePlan{NextVersion: domain.NewVersion(1, 0, 0)},
	}
	if err := plugin.Prepare(context.Background(), rc); err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if got := readRepoFile(t, dir, "Cargo.toml"); !strings.Contains(got, `version = "1.0.0"`) {
		t.Errorf("single-crate Cargo.toml not bumped:\n%s", got)
	}
}
