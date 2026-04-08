package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// ErrNoModuleDirective is returned by readModuleName when the go.mod file
// contains no module directive. Callers can use errors.Is to distinguish this
// from I/O failures.
var ErrNoModuleDirective = errors.New("no module directive found")

// Compile-time interface compliance checks for all ProjectDiscoverer implementations
// in this package.
var (
	_ ports.ProjectDiscoverer = (*WorkspaceDiscoverer)(nil)
	_ ports.ProjectDiscoverer = (*ModuleDiscoverer)(nil)
	_ ports.ProjectDiscoverer = (*ConfiguredDiscoverer)(nil)
	_ ports.ProjectDiscoverer = (*CompositeDiscoverer)(nil)
	_ ports.ProjectDiscoverer = (*CmdDiscoverer)(nil)
)

// WorkspaceDiscoverer discovers projects from go.work files.
type WorkspaceDiscoverer struct {
	fs ports.FileSystem
}

// NewWorkspaceDiscoverer creates a discoverer for go workspace monorepos.
func NewWorkspaceDiscoverer(fsys ports.FileSystem) *WorkspaceDiscoverer {
	return &WorkspaceDiscoverer{fs: fsys}
}

func (d *WorkspaceDiscoverer) Discover(ctx context.Context, rootPath string) ([]domain.Project, error) {
	workFile := filepath.Join(rootPath, "go.work")
	if !d.fs.Exists(workFile) {
		return nil, nil
	}

	data, err := d.fs.ReadFile(workFile)
	if err != nil {
		return nil, fmt.Errorf("reading go.work: %w", err)
	}

	dirs, err := parseGoWorkUse(data)
	if err != nil {
		return nil, fmt.Errorf("parsing go.work: %w", err)
	}
	projects := make([]domain.Project, 0, len(dirs))

	// Context is only checked between loop iterations. The ReadFile and
	// parseGoWorkUse calls above the loop do not respect cancellation; this is
	// acceptable for the typically small go.work files seen in practice.
	for _, dir := range dirs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		modPath := filepath.Join(rootPath, dir, "go.mod")
		moduleName, err := readModuleName(d.fs, modPath)
		if err != nil {
			return nil, fmt.Errorf("reading module name from %s: %w", modPath, err)
		}
		// filepath.Clean strips the leading "./" from go.work use entries
		// (e.g. "./svc-api" → "svc-api") so that project names and tag prefixes
		// are consistent with ModuleDiscoverer, which uses filepath.Rel for the
		// same normalisation. Without this, tag names would contain an invalid
		// "./" segment (e.g. "./svc-api/v1.0.0").
		cleanDir := filepath.Clean(dir)
		name := cleanDir
		tagPrefix := cleanDir + "/"
		if cleanDir == "." {
			name = filepath.Base(rootPath)
			if name == "/" || name == "." {
				name = "root"
			}
			// Root project has no tag prefix, matching ModuleDiscoverer behavior.
			tagPrefix = ""
		}

		projects = append(projects, domain.Project{
			Name:       name,
			Path:       cleanDir,
			Type:       domain.ProjectTypeGoWorkspace,
			ModulePath: moduleName,
			TagPrefix:  tagPrefix,
		})
	}
	return projects, nil
}

// parseGoWorkUse extracts "use" directives from a go.work file.
// Returns (dirs, error) where error is non-nil only on scanner I/O failure or
// a line exceeding bufio.MaxScanTokenSize (64 KiB) — not on parse/format issues.
func parseGoWorkUse(content []byte) ([]string, error) {
	var dirs []string
	scanner := bufio.NewScanner(bytes.NewReader(content))
	inUseBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// noSpaces is used only to detect "use(" with arbitrary spacing between
		// the keyword and the opening paren. It is NOT used for directory parsing —
		// dir is always derived from the original line so paths with spaces are
		// preserved correctly.
		noSpaces := strings.ReplaceAll(line, " ", "")
		if noSpaces == "use(" {
			inUseBlock = true
			continue
		}
		if inUseBlock && line == ")" {
			inUseBlock = false
			continue
		}
		if inUseBlock {
			dir := strings.TrimSpace(line)
			if strings.HasPrefix(dir, "//") {
				continue
			}
			// Strip inline comments (e.g. "./path // comment" → "./path").
			if idx := strings.Index(dir, "//"); idx >= 0 {
				dir = strings.TrimSpace(dir[:idx])
			}
			if dir != "" {
				dirs = append(dirs, dir)
			}
			continue
		}
		// Single-line form: "use ./path". The go toolchain accepts both a space
		// and a tab after "use", so we check for both. We avoid "used ./path"
		// by requiring that the separator character immediately follows "use".
		if (strings.HasPrefix(line, "use ") || strings.HasPrefix(line, "use\t")) && !strings.Contains(line, "(") {
			dir := strings.TrimSpace(line[len("use"):])
			if dir != "" {
				dirs = append(dirs, dir)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanning go.work: %w", err)
	}
	return dirs, nil
}

// ModuleDiscoverer discovers projects by finding go.mod files recursively.
type ModuleDiscoverer struct {
	fs ports.FileSystem
}

// NewModuleDiscoverer creates a discoverer for nested go.mod monorepos.
func NewModuleDiscoverer(fsys ports.FileSystem) *ModuleDiscoverer {
	return &ModuleDiscoverer{fs: fsys}
}

func (d *ModuleDiscoverer) Discover(ctx context.Context, rootPath string) ([]domain.Project, error) {
	// filepath.Glob does not support recursive "**" patterns — it treats "**" as a
	// literal directory name, not a recursive wildcard. Walk is used instead so that
	// go.mod files at any depth are discovered correctly.
	//
	// Context is checked inside the WalkDirFunc: returning an error from the func is
	// the standard mechanism to abort a Walk early. The Walk return value propagates
	// the cancellation error to the caller below.
	//
	// The returned project order mirrors the Walk order (lexicographic, depth-first).
	// The root go.mod is visited before nested ones, so projects[0].Type == Root
	// holds for any repo with a root module. Tests that rely on this must not sort
	// the result.
	var matches []string
	walkErr := d.fs.Walk(rootPath, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if !de.IsDir() && de.Name() == "go.mod" {
			matches = append(matches, path)
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("scanning for go.mod files: %w", walkErr)
	}

	projects := make([]domain.Project, 0, len(matches))
	for _, match := range matches {
		// No per-iteration ctx check needed here: the Walk callback above already
		// propagates cancellation, so walkErr will be non-nil and the function
		// returns before reaching this loop if the context was cancelled.
		rel, err := filepath.Rel(rootPath, filepath.Dir(match))
		if err != nil {
			return nil, fmt.Errorf("computing relative path for %s: %w", match, err)
		}

		moduleName, err := readModuleName(d.fs, match)
		if err != nil {
			return nil, fmt.Errorf("reading module name from %s: %w", match, err)
		}
		name := rel
		projType := domain.ProjectTypeGoModule
		tagPrefix := name + "/"

		if rel == "." {
			name = filepath.Base(rootPath)
			if name == "/" || name == "." {
				name = "root"
			}
			projType = domain.ProjectTypeRoot
			tagPrefix = ""
		}

		projects = append(projects, domain.Project{
			Name:       name,
			Path:       rel,
			Type:       projType,
			ModulePath: moduleName,
			TagPrefix:  tagPrefix,
		})
	}
	return projects, nil
}

// ConfiguredDiscoverer creates projects from static config definitions.
type ConfiguredDiscoverer struct {
	projects []domain.ProjectConfig
}

// NewConfiguredDiscoverer creates a discoverer from config-defined projects.
func NewConfiguredDiscoverer(projects []domain.ProjectConfig) *ConfiguredDiscoverer {
	return &ConfiguredDiscoverer{projects: projects}
}

// Discover returns the statically configured projects. The context is intentionally
// ignored because this discoverer operates on in-memory data with no I/O.
// rootPath is also ignored: project paths come directly from config and are
// resolved relative to the repo root by callers.
func (d *ConfiguredDiscoverer) Discover(_ context.Context, _ string) ([]domain.Project, error) {
	result := make([]domain.Project, 0, len(d.projects))
	for _, pc := range d.projects {
		prefix := pc.TagPrefix
		if prefix == "" {
			prefix = pc.Name + "/"
		}
		// Clean the path from config so that IsRoot() and tag-prefix logic
		// receive a normalised value. filepath.Clean("./services/api") →
		// "services/api", matching what ModuleDiscoverer produces via filepath.Rel.
		result = append(result, domain.Project{
			Name:          pc.Name,
			Path:          filepath.Clean(pc.Path),
			Type:          domain.ProjectTypeConfigured,
			Dependencies:  pc.Dependencies,
			TagPrefix:     prefix,
			ChangelogFile: pc.ChangelogFile,
		})
	}
	return result, nil
}

// readModuleName reads the module directive from a go.mod file.
// It returns an error if the file cannot be read or has no module directive.
func readModuleName(fsys ports.FileSystem, modFile string) (string, error) {
	data, err := fsys.ReadFile(modFile)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", modFile, err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// HasPrefix and TrimPrefix both use "module " (with a trailing space) so the
		// guard and the trim are consistent: a line like "modulex github.com/..." is
		// not matched, and the returned name has no leading space.
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return "", fmt.Errorf("scanning %s: %w", modFile, scanErr)
	}
	return "", fmt.Errorf("%w in %s", ErrNoModuleDirective, modFile)
}

// CompositeDiscoverer tries multiple discoverers in order and returns the first non-empty result.
type CompositeDiscoverer struct {
	discoverers []ports.ProjectDiscoverer
}

// NewCompositeDiscoverer chains discoverers. First non-empty result wins.
func NewCompositeDiscoverer(discoverers ...ports.ProjectDiscoverer) *CompositeDiscoverer {
	return &CompositeDiscoverer{discoverers: discoverers}
}

// Discover runs each discoverer in order and returns the first non-empty result.
// Returns (nil, nil) when no discoverer finds any projects — callers should treat
// this as "nothing found", not as an error or "unsupported" state.
func (d *CompositeDiscoverer) Discover(ctx context.Context, rootPath string) ([]domain.Project, error) {
	for i, disc := range d.discoverers {
		projects, err := disc.Discover(ctx, rootPath)
		if err != nil {
			return nil, fmt.Errorf("discoverer[%d] %T: %w", i, disc, err)
		}
		if len(projects) > 0 {
			return projects, nil
		}
	}
	return nil, nil
}
