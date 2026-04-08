package plugins

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance checks.
var (
	_ ports.Plugin        = (*PreparePlugin)(nil)
	_ ports.PreparePlugin = (*PreparePlugin)(nil)
)

// PreparePlugin updates files (CHANGELOG.md, VERSION) before the release is published.
type PreparePlugin struct {
	fs            ports.FileSystem
	logger        ports.Logger
	changelogFile string // global changelog path relative to repo root, empty to skip
	versionFile   string // path to VERSION file, empty to skip
}

// NewPreparePlugin creates a plugin that updates release files.
func NewPreparePlugin(fsys ports.FileSystem, logger ports.Logger, cfg domain.PrepareConfig) *PreparePlugin {
	return &PreparePlugin{
		fs:            fsys,
		logger:        logger,
		changelogFile: cfg.ChangelogFile,
		versionFile:   cfg.VersionFile,
	}
}

func (p *PreparePlugin) Name() string { return "prepare-files" }

func (p *PreparePlugin) Prepare(ctx context.Context, rc *domain.ReleaseContext) error {
	if rc.CurrentProject == nil {
		return nil
	}

	version := rc.CurrentProject.NextVersion

	if err := p.updateVersionFile(ctx, version, rc.RepositoryRoot); err != nil {
		return err
	}

	return p.updateChangelog(ctx, rc)
}

// updateVersionFile writes the version string to the configured VERSION file.
// ctx is accepted for forward-compatibility; ports.FileSystem does not yet support cancellation.
func (p *PreparePlugin) updateVersionFile(_ context.Context, version domain.Version, repoRoot string) error {
	if p.versionFile == "" {
		return nil
	}

	path := filepath.Join(repoRoot, p.versionFile)
	content := version.String() + "\n"

	if err := p.fs.WriteFile(path, []byte(content), fs.FileMode(0o644)); err != nil {
		return fmt.Errorf("writing version file %s: %w", path, err)
	}
	p.logger.Info("updated version file", "path", path, "version", version)
	return nil
}

// changelogPath returns the resolved absolute path for the changelog file, or empty string if not configured.
// A per-project changelog_file takes precedence and is resolved relative to the project's path inside the repo.
// The global changelog_file falls back and is resolved relative to the repository root.
// Safe to call with a nil rc.CurrentProject: falls through to the global path in that case.
func (p *PreparePlugin) changelogPath(rc *domain.ReleaseContext) string {
	if rc.CurrentProject != nil && rc.CurrentProject.Project.ChangelogFile != "" {
		return filepath.Join(rc.RepositoryRoot, rc.CurrentProject.Project.Path, rc.CurrentProject.Project.ChangelogFile)
	}
	if p.changelogFile == "" {
		return ""
	}
	return filepath.Join(rc.RepositoryRoot, p.changelogFile)
}

// updateChangelog prepends the generated release notes into the changelog file.
// ctx is accepted for forward-compatibility; ports.FileSystem does not yet support cancellation.
func (p *PreparePlugin) updateChangelog(_ context.Context, rc *domain.ReleaseContext) error {
	// Require an absolute RepositoryRoot so the traversal guard below is reliable.
	// A relative root (e.g. ".") would make filepath.Clean produce a relative prefix,
	// causing valid absolute changelog paths to fail the HasPrefix check.
	if !filepath.IsAbs(rc.RepositoryRoot) {
		return fmt.Errorf("RepositoryRoot must be an absolute path, got: %q", rc.RepositoryRoot)
	}

	// Resolve the raw path first; empty means no changelog is configured.
	raw := p.changelogPath(rc)
	if raw == "" {
		return nil
	}
	// Explicitly clean the path so the traversal guard holds regardless of how
	// changelogPath constructs the string in the future.
	path := filepath.Clean(raw)
	if path == "." {
		return nil
	}

	// Guard against user-supplied changelog_file values that escape the repository root
	// via path traversal (e.g. "../../etc/passwd"). The separator suffix is required so
	// that a root of "/repo" does not accidentally allow "/repo-sibling/evil".
	root := filepath.Clean(rc.RepositoryRoot)
	if !strings.HasPrefix(path, root+string(filepath.Separator)) {
		return fmt.Errorf("changelog_file path escapes repository root: %s", path)
	}

	newEntry := rc.Notes
	if newEntry == "" {
		// Nothing to prepend — skip silently rather than writing a blank entry.
		return nil
	}

	// TODO(ports/filesystem): replace Exists+ReadFile with a single ReadFile call
	// that treats ErrNotExist as an empty file, once ports.FileSystem exposes
	// that sentinel. There is a TOCTOU window between Exists returning false and
	// the subsequent WriteFile: a concurrent process could create the file in
	// between. This is acceptable in practice because CI environments run one
	// release process at a time, but the single-call approach would close the
	// window entirely.
	existing := ""
	if p.fs.Exists(path) {
		data, err := p.fs.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading changelog %s: %w", path, err)
		}
		existing = string(data)
	}

	// Prepend new entry after the title line if it exists, otherwise at the top.
	updated := prependChangelog(existing, newEntry)

	if err := p.fs.WriteFile(path, []byte(updated), fs.FileMode(0o644)); err != nil {
		return fmt.Errorf("writing changelog %s: %w", path, err)
	}
	p.logger.Info("updated changelog", "path", path)
	return nil
}

func prependChangelog(existing, newEntry string) string {
	if existing == "" {
		return "# Changelog\n\n" + newEntry + "\n"
	}

	// If there's a title line (# Changelog), insert after it.
	lines := strings.SplitN(existing, "\n", 2)
	if strings.HasPrefix(lines[0], "# ") {
		rest := ""
		if len(lines) > 1 {
			rest = lines[1]
		}
		// TrimLeft removes any leading newlines from the remainder so that repeated
		// prepend operations do not accumulate blank lines between entries.
		return lines[0] + "\n\n" + newEntry + "\n\n" + strings.TrimLeft(rest, "\n")
	}

	return newEntry + "\n\n" + existing
}
