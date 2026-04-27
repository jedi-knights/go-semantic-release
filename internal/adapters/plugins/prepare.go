package plugins

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
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

// commandRunnerFunc is the function type used to execute prepare commands.
type commandRunnerFunc func(ctx context.Context, cmd string, version domain.Version) error

// PrepareOption configures a PreparePlugin after construction.
type PrepareOption func(*PreparePlugin)

// WithCommandRunner injects a custom command runner. Intended for testing.
func WithCommandRunner(fn commandRunnerFunc) PrepareOption {
	return func(p *PreparePlugin) {
		p.runCmd = fn
	}
}

// PreparePlugin updates files (CHANGELOG.md, VERSION, version_files) before the release is published,
// then optionally runs a prepare command.
type PreparePlugin struct {
	fs            ports.FileSystem
	logger        ports.Logger
	changelogFile string   // global changelog path relative to repo root, empty to skip
	versionFile   string   // path to VERSION file, empty to skip
	command       string   // shell command to run after file updates, empty to skip
	versionFiles  []string // additional version files (format: "path" or "path:key.path")
	runCmd        commandRunnerFunc
}

// NewPreparePlugin creates a plugin that updates release files.
func NewPreparePlugin(fsys ports.FileSystem, logger ports.Logger, cfg domain.PrepareConfig, opts ...PrepareOption) *PreparePlugin {
	p := &PreparePlugin{
		fs:            fsys,
		logger:        logger,
		changelogFile: cfg.ChangelogFile,
		versionFile:   cfg.VersionFile,
		command:       cfg.Command,
		versionFiles:  cfg.VersionFiles,
		runCmd:        defaultCommandRunner,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *PreparePlugin) Name() string { return "prepare-files" }

func (p *PreparePlugin) Prepare(ctx context.Context, rc *domain.ReleaseContext) error {
	if rc.CurrentProject == nil {
		return nil
	}

	// Dry-run skips every mutation in this plugin and logs what would have happened
	// instead. The same path/traversal validations still run so dry-run reports the
	// same configuration errors a real run would surface.
	if rc.DryRun {
		return p.previewPrepare(rc)
	}

	version := rc.CurrentProject.NextVersion

	if err := p.updateVersionFile(ctx, version, rc.RepositoryRoot); err != nil {
		return err
	}

	if err := p.updateVersionFiles(ctx, version, rc.RepositoryRoot); err != nil {
		return err
	}

	if err := p.runCommand(ctx, version); err != nil {
		return err
	}

	return p.updateChangelog(ctx, rc)
}

// previewPrepare logs the file mutations and command execution that a real prepare
// step would perform, without touching the filesystem or running any command. It
// preserves the same path-traversal and absolute-root validations as the real path
// so misconfiguration is reported consistently in dry-run.
func (p *PreparePlugin) previewPrepare(rc *domain.ReleaseContext) error {
	version := rc.CurrentProject.NextVersion

	if p.versionFile != "" {
		path := filepath.Join(rc.RepositoryRoot, p.versionFile)
		p.logger.Info("dry run: would update version file", "path", path, "version", version)
	}

	for _, entry := range p.versionFiles {
		ve := domain.ParseVersionFileEntry(entry)
		path := filepath.Join(rc.RepositoryRoot, ve.Path)
		if ve.KeyPath == "" {
			p.logger.Info("dry run: would update version file", "path", path, "version", version)
		} else {
			p.logger.Info("dry run: would update TOML version key", "path", path, "key", ve.KeyPath, "version", version)
		}
	}

	if p.command != "" {
		p.logger.Info("dry run: would run prepare command", "command", p.command)
	}

	path, err := p.validatedChangelogPath(rc)
	if err != nil || path == "" {
		return err
	}
	if rc.Notes == "" {
		return nil
	}
	p.logger.Info("dry run: would update changelog", "path", path)
	return nil
}

// validatedChangelogPath resolves and validates the changelog path. Returns "" when
// no changelog is configured. Returns an error when RepositoryRoot is not absolute
// or when the resolved path escapes the repository root.
func (p *PreparePlugin) validatedChangelogPath(rc *domain.ReleaseContext) (string, error) {
	if !filepath.IsAbs(rc.RepositoryRoot) {
		return "", fmt.Errorf("RepositoryRoot must be an absolute path, got: %q", rc.RepositoryRoot)
	}
	raw := p.changelogPath(rc)
	if raw == "" {
		return "", nil
	}
	path := filepath.Clean(raw)
	if path == "." {
		return "", nil
	}
	root := filepath.Clean(rc.RepositoryRoot)
	if !strings.HasPrefix(path, root+string(filepath.Separator)) {
		return "", fmt.Errorf("changelog_file path escapes repository root: %s", path)
	}
	return path, nil
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

// updateVersionFiles processes each entry in version_files.
// Entries of the form "path:key.path" update a TOML key; plain "path" entries write the version as plain text.
func (p *PreparePlugin) updateVersionFiles(_ context.Context, version domain.Version, repoRoot string) error {
	for _, entry := range p.versionFiles {
		ve := domain.ParseVersionFileEntry(entry)
		path := filepath.Join(repoRoot, ve.Path)

		if ve.KeyPath == "" {
			if err := p.fs.WriteFile(path, []byte(version.String()+"\n"), fs.FileMode(0o644)); err != nil {
				return fmt.Errorf("writing version file %s: %w", path, err)
			}
			p.logger.Info("updated version file", "path", path, "version", version)
			continue
		}

		content, err := p.fs.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		updated, err := updateTOMLKey(content, ve.KeyPath, version.String())
		if err != nil {
			return fmt.Errorf("updating TOML key in %s: %w", path, err)
		}
		if err := p.fs.WriteFile(path, updated, fs.FileMode(0o644)); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
		p.logger.Info("updated TOML version key", "path", path, "key", ve.KeyPath, "version", version)
	}
	return nil
}

// runCommand executes the configured prepare command, exposing NEXT_RELEASE_VERSION as an env var.
func (p *PreparePlugin) runCommand(ctx context.Context, version domain.Version) error {
	if p.command == "" {
		return nil
	}
	p.logger.Info("running prepare command", "command", p.command)
	if err := p.runCmd(ctx, p.command, version); err != nil {
		return fmt.Errorf("prepare command failed: %w", err)
	}
	return nil
}

// defaultCommandRunner executes a shell command via sh -c.
// The cmd string is executed verbatim as a shell command. Operators are
// responsible for ensuring that extended remote configurations are trusted,
// as a compromised remote extends URL could inject arbitrary shell commands.
func defaultCommandRunner(ctx context.Context, cmd string, version domain.Version) error {
	c := exec.CommandContext(ctx, "sh", "-c", cmd)
	c.Env = append(os.Environ(), "NEXT_RELEASE_VERSION="+version.String())
	var out bytes.Buffer
	c.Stdout = &out
	c.Stderr = &out
	if err := c.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(out.String()))
	}
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
	path, err := p.validatedChangelogPath(rc)
	if err != nil || path == "" {
		return err
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
