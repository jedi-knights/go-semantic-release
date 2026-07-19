package cargo

import (
	"os"
	"path/filepath"
	"testing"

	adapterfs "github.com/jedi-knights/go-semantic-release/internal/adapters/fs"
)

// writeFile writes content to dir/rel, creating parent directories.
func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetect(t *testing.T) {
	fsys := adapterfs.NewOSFileSystem()

	t.Run("no Cargo.toml returns nil", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "go.mod", "module example.com/x\n")
		info, err := Detect(fsys, dir)
		if err != nil {
			t.Fatal(err)
		}
		if info != nil {
			t.Fatalf("expected nil, got %+v", info)
		}
	})

	t.Run("single crate", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Cargo.toml", `[package]
name = "plug-audit"
version = "0.1.0"
edition = "2024"
`)
		info, err := Detect(fsys, dir)
		if err != nil {
			t.Fatal(err)
		}
		if info == nil {
			t.Fatal("expected info, got nil")
		}
		if info.Kind != KindCrate {
			t.Errorf("Kind = %v, want KindCrate", info.Kind)
		}
		if info.VersionKeyPath != "package.version" {
			t.Errorf("VersionKeyPath = %q, want package.version", info.VersionKeyPath)
		}
		assertNames(t, info.CrateNames, []string{"plug-audit"})
	})

	t.Run("workspace with explicit members", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Cargo.toml", `[workspace]
resolver = "2"
members = [
    "crates/vcl-core",
    "crates/vcl-cli",
]

[workspace.package]
version = "0.1.0"
edition = "2021"
`)
		writeFile(t, dir, "crates/vcl-core/Cargo.toml", "[package]\nname = \"vcl-core\"\nversion.workspace = true\n")
		writeFile(t, dir, "crates/vcl-cli/Cargo.toml", "[package]\nname = \"vcl-cli\"\nversion.workspace = true\n")

		info, err := Detect(fsys, dir)
		if err != nil {
			t.Fatal(err)
		}
		if info == nil {
			t.Fatal("expected info, got nil")
		}
		if info.Kind != KindWorkspace {
			t.Errorf("Kind = %v, want KindWorkspace", info.Kind)
		}
		if info.VersionKeyPath != "workspace.package.version" {
			t.Errorf("VersionKeyPath = %q, want workspace.package.version", info.VersionKeyPath)
		}
		assertNames(t, info.CrateNames, []string{"vcl-cli", "vcl-core"})
	})

	t.Run("workspace with glob members", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Cargo.toml", `[workspace]
members = ["crates/*"]

[workspace.package]
version = "0.1.0"
`)
		writeFile(t, dir, "crates/a/Cargo.toml", "[package]\nname = \"crate-a\"\n")
		writeFile(t, dir, "crates/b/Cargo.toml", "[package]\nname = \"crate-b\"\n")

		info, err := Detect(fsys, dir)
		if err != nil {
			t.Fatal(err)
		}
		if info == nil {
			t.Fatal("expected info, got nil")
		}
		assertNames(t, info.CrateNames, []string{"crate-a", "crate-b"})
	})

	t.Run("workspace root that is also a package includes the root crate", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Cargo.toml", `[package]
name = "root-crate"

[workspace]
members = ["sub"]

[workspace.package]
version = "0.1.0"
`)
		writeFile(t, dir, "sub/Cargo.toml", "[package]\nname = \"sub-crate\"\n")

		info, err := Detect(fsys, dir)
		if err != nil {
			t.Fatal(err)
		}
		if info.Kind != KindWorkspace {
			t.Errorf("Kind = %v, want KindWorkspace", info.Kind)
		}
		assertNames(t, info.CrateNames, []string{"root-crate", "sub-crate"})
	})
}

func assertNames(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("CrateNames = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("CrateNames = %v, want %v", got, want)
		}
	}
}
