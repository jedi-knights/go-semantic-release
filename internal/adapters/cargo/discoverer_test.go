package cargo

import (
	"context"
	"testing"

	adapterfs "github.com/jedi-knights/go-semantic-release/internal/adapters/fs"
	"github.com/jedi-knights/go-semantic-release/internal/domain"
)

func TestRustDiscoverer_Discover(t *testing.T) {
	fsys := adapterfs.NewOSFileSystem()
	d := NewRustDiscoverer(fsys)

	t.Run("non-cargo repo yields no projects", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "go.mod", "module example.com/x\n")
		projects, err := d.Discover(context.Background(), dir)
		if err != nil {
			t.Fatal(err)
		}
		if projects != nil {
			t.Fatalf("expected nil, got %v", projects)
		}
	})

	t.Run("single crate is cargo-crate", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Cargo.toml", "[package]\nname = \"plug-audit\"\nversion = \"0.1.0\"\n")
		projects, err := d.Discover(context.Background(), dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(projects) != 1 {
			t.Fatalf("expected 1 project, got %d", len(projects))
		}
		if projects[0].Type != domain.ProjectTypeCargoCrate {
			t.Errorf("Type = %v, want cargo-crate", projects[0].Type)
		}
		if !projects[0].IsRoot() {
			t.Errorf("expected root project, got path %q", projects[0].Path)
		}
		// Name must be empty so repo-mode renders unprefixed "vX.Y.Z" tags,
		// matching the repo-root fallback the org's Rust repos rely on.
		if projects[0].Name != "" {
			t.Errorf("Name = %q, want empty (unprefixed tags)", projects[0].Name)
		}
		if projects[0].TagPrefix != "" {
			t.Errorf("TagPrefix = %q, want empty", projects[0].TagPrefix)
		}
	})

	t.Run("workspace is cargo-workspace", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, "Cargo.toml", "[workspace]\nmembers = [\"crates/a\"]\n\n[workspace.package]\nversion = \"0.1.0\"\n")
		writeFile(t, dir, "crates/a/Cargo.toml", "[package]\nname = \"a\"\n")
		projects, err := d.Discover(context.Background(), dir)
		if err != nil {
			t.Fatal(err)
		}
		if len(projects) != 1 {
			t.Fatalf("expected 1 project, got %d", len(projects))
		}
		if projects[0].Type != domain.ProjectTypeCargoWorkspace {
			t.Errorf("Type = %v, want cargo-workspace", projects[0].Type)
		}
	})
}
