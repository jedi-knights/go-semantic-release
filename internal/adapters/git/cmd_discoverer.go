package git

import (
	"context"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// CmdDiscoverer discovers projects in a single-module Go monorepo that follows
// the cmd/<service>/main.go layout. It is only activated when:
//   - A go.mod file exists at the repository root (single module).
//   - No go.work file is present (not a Go workspace — use WorkspaceDiscoverer for those).
//   - A cmd/ directory exists at the repository root.
//
// For each immediate subdirectory of cmd/ that contains a main.go, a
// ProjectTypeCmdService project is created. Additionally, any pkg/<name> import
// path that is used by more than one service becomes a ProjectTypeCmdLibrary
// project and is listed as a dependency of all consuming services.
type CmdDiscoverer struct {
	fs ports.FileSystem
}

// NewCmdDiscoverer creates a CmdDiscoverer that reads from the given filesystem.
func NewCmdDiscoverer(fsys ports.FileSystem) *CmdDiscoverer {
	return &CmdDiscoverer{fs: fsys}
}

// Discover returns discovered projects. Returns (nil, nil) when the repository
// does not match the cmd/ monorepo pattern, or when the cmd/ directory exists
// but contains no subdirectory with a main.go.
//
// Output order: services (sorted by name) followed by shared libraries (sorted
// by name). This ordering is stable but is an implementation detail — callers
// should not depend on the relative position of a specific project.
func (d *CmdDiscoverer) Discover(ctx context.Context, rootPath string) ([]domain.Project, error) {
	// Guard 1: single-module — go.mod must exist at root.
	if !d.fs.Exists(filepath.Join(rootPath, "go.mod")) {
		return nil, nil
	}
	// Guard 2: not a workspace — go.work must NOT exist.
	if d.fs.Exists(filepath.Join(rootPath, "go.work")) {
		return nil, nil
	}
	// Guard 3: cmd/ directory must exist.
	cmdDir := filepath.Join(rootPath, "cmd")
	if !d.fs.Exists(cmdDir) {
		return nil, nil
	}

	moduleName, err := readModuleName(d.fs, filepath.Join(rootPath, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("cmd discoverer: reading module name: %w", err)
	}

	// Walk cmd/ to find immediate subdirectories that contain a main.go file.
	// We collect (serviceName → importedPkgNames) per service.
	type serviceInfo struct {
		name string   // e.g. "api"
		pkgs []string // pkg/ sub-package names imported (e.g. "queue")
	}

	var services []serviceInfo

	walkErr := d.fs.Walk(cmdDir, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}

		rel, relErr := filepath.Rel(cmdDir, path)
		if relErr != nil {
			return relErr
		}
		parts := strings.Split(rel, string(filepath.Separator))

		// Only interested in main.go files exactly one level below cmd/:
		// rel = "<service>/main.go" → parts = ["<service>", "main.go"]
		// Service name is always derived from parts[0] so that discovery does
		// not depend on the directory entry arriving before its children.
		if !de.IsDir() && de.Name() == "main.go" && len(parts) == 2 {
			svcName := parts[0]
			pkgs, parseErr := d.parsePkgImports(path, moduleName)
			if parseErr != nil {
				return parseErr
			}
			services = append(services, serviceInfo{name: svcName, pkgs: pkgs})
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("cmd discoverer: walking %s: %w", cmdDir, walkErr)
	}

	if len(services) == 0 {
		return nil, nil
	}

	// Count how many services use each pkg/ package.
	// parsePkgImports already deduplicates within a file, so each entry in
	// services[i].pkgs is unique — no additional per-service dedup is needed.
	pkgUsage := make(map[string]int) // pkgName → usage count
	for i := range services {
		for _, pkg := range services[i].pkgs {
			pkgUsage[pkg]++
		}
	}

	// Packages used by more than one service become library projects.
	sharedPkgs := make(map[string]bool)
	for pkg, count := range pkgUsage {
		if count > 1 {
			sharedPkgs[pkg] = true
		}
	}

	var projects []domain.Project

	// Service projects first, sorted by name for determinism.
	slices.SortFunc(services, func(a, b serviceInfo) int { return strings.Compare(a.name, b.name) })
	for i := range services {
		svc := &services[i]
		var deps []string
		for _, pkg := range svc.pkgs {
			if sharedPkgs[pkg] {
				deps = append(deps, pkg)
			}
		}
		slices.Sort(deps)

		projects = append(projects, domain.Project{
			Name:         svc.name,
			Path:         filepath.Join("cmd", svc.name),
			Type:         domain.ProjectTypeCmdService,
			ModulePath:   moduleName,
			TagPrefix:    svc.name + "/",
			Dependencies: deps,
		})
	}

	// Library projects, sorted by name.
	var libNames []string
	for name := range sharedPkgs {
		libNames = append(libNames, name)
	}
	slices.Sort(libNames)

	for _, name := range libNames {
		projects = append(projects, domain.Project{
			Name:       name,
			Path:       filepath.Join("pkg", name),
			Type:       domain.ProjectTypeCmdLibrary,
			ModulePath: moduleName,
			TagPrefix:  name + "/",
		})
	}

	return projects, nil
}

// parsePkgImports reads a Go source file and returns the names of any
// pkg/<name> sub-packages it imports within the given module.
func (d *CmdDiscoverer) parsePkgImports(filePath, moduleName string) ([]string, error) {
	data, err := d.fs.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filePath, err)
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, data, parser.ImportsOnly)
	if err != nil {
		return nil, fmt.Errorf("parsing imports in %s: %w", filePath, err)
	}

	pkgPrefix := moduleName + "/pkg/"
	seen := make(map[string]bool)
	var pkgs []string
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		if rest, ok := strings.CutPrefix(path, pkgPrefix); ok {
			// Extract the immediate sub-package name: "module/pkg/queue/sub" → "queue".
			name := strings.SplitN(rest, "/", 2)[0]
			if name != "" && !seen[name] {
				seen[name] = true
				pkgs = append(pkgs, name)
			}
		}
	}
	return pkgs, nil
}
