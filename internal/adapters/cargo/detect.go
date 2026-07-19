package cargo

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/pelletier/go-toml/v2"

	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Kind describes the shape of a detected Cargo project.
type Kind int

const (
	// KindCrate is a single-crate project: a root Cargo.toml with a [package] section.
	KindCrate Kind = iota + 1
	// KindWorkspace is a Cargo workspace: a root Cargo.toml with a [workspace] section.
	KindWorkspace
)

// Info describes a detected Cargo project.
type Info struct {
	// Kind is whether the project is a single crate or a workspace.
	Kind Kind
	// VersionKeyPath is the dot-separated TOML key that carries the shared
	// version (e.g. "package.version" or "workspace.package.version"). It is
	// empty when no single shared version key exists.
	VersionKeyPath string
	// CrateNames are the local crate names (for updating Cargo.lock entries).
	CrateNames []string
}

// manifest is the subset of Cargo.toml fields needed for detection. Reading uses
// a real TOML parser (multi-line arrays, inline tables); writes stay surgical.
type manifest struct {
	Package   *pkgSection `toml:"package"`
	Workspace *wsSection  `toml:"workspace"`
}

type pkgSection struct {
	Name string `toml:"name"`
}

type wsSection struct {
	Members []string `toml:"members"`
	Package *wsPkg   `toml:"package"`
}

type wsPkg struct {
	Version string `toml:"version"`
}

// Detect inspects rootPath for a Cargo project. It returns nil (no error) when
// no root Cargo.toml is present, so callers can treat "not a Rust repo" as a
// harmless no-op.
func Detect(fsys ports.FileSystem, rootPath string) (*Info, error) {
	manifestPath := filepath.Join(rootPath, "Cargo.toml")
	if !fsys.Exists(manifestPath) {
		return nil, nil
	}

	root, err := readManifest(fsys, manifestPath)
	if err != nil {
		return nil, fmt.Errorf("reading root Cargo.toml: %w", err)
	}

	if root.Workspace != nil {
		return detectWorkspace(fsys, rootPath, root)
	}
	if root.Package != nil {
		return &Info{
			Kind:           KindCrate,
			VersionKeyPath: "package.version",
			CrateNames:     namesOf(root.Package.Name),
		}, nil
	}
	// A Cargo.toml with neither [package] nor [workspace] is not something we can
	// version — treat it as not detected rather than erroring.
	return nil, nil
}

// detectWorkspace resolves a workspace manifest: its version key and the local
// crate names (root package if present, plus every member crate).
func detectWorkspace(fsys ports.FileSystem, rootPath string, root *manifest) (*Info, error) {
	info := &Info{Kind: KindWorkspace}
	if root.Workspace.Package != nil {
		// Only a shared [workspace.package] version is auto-bumpable as one key.
		info.VersionKeyPath = "workspace.package.version"
	}

	names := map[string]struct{}{}
	if root.Package != nil && root.Package.Name != "" {
		names[root.Package.Name] = struct{}{}
	}

	for _, member := range root.Workspace.Members {
		// Cargo members may be globs (e.g. "crates/*"); expand via the filesystem.
		// filepath.Glob (behind fsys.Glob) matches a single path segment per "*",
		// which is exactly Cargo's member-glob semantics.
		pattern := filepath.Join(rootPath, member, "Cargo.toml")
		matches, err := fsys.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("expanding workspace member %q: %w", member, err)
		}
		for _, match := range matches {
			m, err := readManifest(fsys, match)
			if err != nil {
				return nil, fmt.Errorf("reading member manifest %s: %w", match, err)
			}
			if m.Package != nil && m.Package.Name != "" {
				names[m.Package.Name] = struct{}{}
			}
		}
	}

	info.CrateNames = sortedKeys(names)
	return info, nil
}

func readManifest(fsys ports.FileSystem, path string) (*manifest, error) {
	data, err := fsys.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m manifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return &m, nil
}

func namesOf(name string) []string {
	if name == "" {
		return nil
	}
	return []string{name}
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
