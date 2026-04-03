package git

import (
	"bufio"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance checks.
var (
	_ ports.ProjectDiscoverer = (*WorkspaceDiscoverer)(nil)
	_ ports.ProjectDiscoverer = (*ModuleDiscoverer)(nil)
	_ ports.ProjectDiscoverer = (*ConfiguredDiscoverer)(nil)
	_ ports.ProjectDiscoverer = (*CompositeDiscoverer)(nil)
)

// WorkspaceDiscoverer discovers projects from go.work files.
type WorkspaceDiscoverer struct {
	fs ports.FileSystem
}

// NewWorkspaceDiscoverer creates a discoverer for go workspace monorepos.
func NewWorkspaceDiscoverer(fs ports.FileSystem) *WorkspaceDiscoverer {
	return &WorkspaceDiscoverer{fs: fs}
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

	dirs := parseGoWorkUse(string(data))
	projects := make([]domain.Project, 0, len(dirs))

	for _, dir := range dirs {
		modPath := filepath.Join(rootPath, dir, "go.mod")
		moduleName := readModuleName(d.fs, modPath)
		name := dir
		if name == "." {
			name = filepath.Base(rootPath)
		}

		projects = append(projects, domain.Project{
			Name:       name,
			Path:       dir,
			Type:       domain.ProjectTypeGoWorkspace,
			ModulePath: moduleName,
			TagPrefix:  name + "/",
		})
	}
	return projects, nil
}

// parseGoWorkUse extracts "use" directives from a go.work file.
func parseGoWorkUse(content string) []string {
	var dirs []string
	scanner := bufio.NewScanner(strings.NewReader(content))
	inUseBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "use (") || line == "use (" {
			inUseBlock = true
			continue
		}
		if inUseBlock && line == ")" {
			inUseBlock = false
			continue
		}
		if inUseBlock {
			dir := strings.TrimSpace(line)
			if dir != "" && !strings.HasPrefix(dir, "//") {
				dirs = append(dirs, dir)
			}
			continue
		}
		if strings.HasPrefix(line, "use ") && !strings.Contains(line, "(") {
			dir := strings.TrimSpace(strings.TrimPrefix(line, "use"))
			if dir != "" {
				dirs = append(dirs, dir)
			}
		}
	}
	return dirs
}

// ModuleDiscoverer discovers projects by finding go.mod files recursively.
type ModuleDiscoverer struct {
	fs ports.FileSystem
}

// NewModuleDiscoverer creates a discoverer for nested go.mod monorepos.
func NewModuleDiscoverer(fs ports.FileSystem) *ModuleDiscoverer {
	return &ModuleDiscoverer{fs: fs}
}

func (d *ModuleDiscoverer) Discover(ctx context.Context, rootPath string) ([]domain.Project, error) {
	matches, err := d.fs.Glob(rootPath + "/**/go.mod")
	if err != nil {
		return nil, fmt.Errorf("scanning for go.mod files: %w", err)
	}

	projects := make([]domain.Project, 0, len(matches))
	for _, match := range matches {
		rel, err := filepath.Rel(rootPath, filepath.Dir(match))
		if err != nil {
			continue
		}

		moduleName := readModuleName(d.fs, match)
		name := rel
		projType := domain.ProjectTypeGoModule
		tagPrefix := name + "/"

		if rel == "." {
			name = filepath.Base(rootPath)
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

func (d *ConfiguredDiscoverer) Discover(_ context.Context, _ string) ([]domain.Project, error) {
	result := make([]domain.Project, 0, len(d.projects))
	for _, pc := range d.projects {
		prefix := pc.TagPrefix
		if prefix == "" {
			prefix = pc.Name + "/"
		}
		result = append(result, domain.Project{
			Name:         pc.Name,
			Path:         pc.Path,
			Type:         domain.ProjectTypeConfigured,
			Dependencies: pc.Dependencies,
			TagPrefix:    prefix,
		})
	}
	return result, nil
}

// readModuleName reads the module directive from a go.mod file.
func readModuleName(fs ports.FileSystem, modFile string) string {
	data, err := fs.ReadFile(modFile)
	if err != nil {
		return ""
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}
	return ""
}

// CompositeDiscoverer tries multiple discoverers in order and returns the first non-empty result.
type CompositeDiscoverer struct {
	discoverers []ports.ProjectDiscoverer
}

// NewCompositeDiscoverer chains discoverers. First non-empty result wins.
func NewCompositeDiscoverer(discoverers ...ports.ProjectDiscoverer) *CompositeDiscoverer {
	return &CompositeDiscoverer{discoverers: discoverers}
}

func (d *CompositeDiscoverer) Discover(ctx context.Context, rootPath string) ([]domain.Project, error) {
	for _, disc := range d.discoverers {
		projects, err := disc.Discover(ctx, rootPath)
		if err != nil {
			return nil, err
		}
		if len(projects) > 0 {
			return projects, nil
		}
	}
	return nil, nil
}
