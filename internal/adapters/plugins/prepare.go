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

	"github.com/jedi-knights/go-semantic-release/internal/adapters/cargo"
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

// WithCargo enables or disables Rust/Cargo awareness. When enabled and a root
// Cargo.toml is present, the prepare step updates the Cargo.toml version key and
// each local crate's version in Cargo.lock. It is disabled by default at
// construction; the DI container enables it based on config (default on).
func WithCargo(enabled bool) PrepareOption {
	return func(p *PreparePlugin) {
		p.cargoEnabled = enabled
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
	cargoEnabled  bool     // update Cargo.toml/Cargo.lock when a root Cargo.toml is present
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

	if err := p.updateCargo(ctx, version, rc.RepositoryRoot); err != nil {
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

	p.previewCargo(version, rc.RepositoryRoot)

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

// previewCargo logs the Cargo file mutations a real prepare would perform,
// without mutating anything. Detection is read-only, so it is safe in dry-run.
func (p *PreparePlugin) previewCargo(version domain.Version, repoRoot string) {
	if !p.cargoEnabled {
		return
	}
	info, err := cargo.Detect(p.fs, repoRoot)
	if err != nil || info == nil {
		return
	}
	if info.VersionKeyPath != "" && !p.versionFilesTarget("Cargo.toml") {
		p.logger.Info("dry run: would update Cargo.toml version", "key", info.VersionKeyPath, "version", version)
	}
	lockPath := filepath.Join(repoRoot, "Cargo.lock")
	if len(info.CrateNames) > 0 && p.fs.Exists(lockPath) {
		p.logger.Info("dry run: would update Cargo.lock", "crates", len(info.CrateNames), "version", version)
	}
}

// updateCargo updates Rust version files when the repository is a Cargo project
// and cargo awareness is enabled: it bumps the shared version key in Cargo.toml
// (unless the user already targets Cargo.toml via version_files) and rewrites
// each local crate's version in Cargo.lock. It is a no-op for non-Rust repos.
//
// Doing the Cargo.lock update natively means the release job needs no Rust
// toolchain — the same flow as Go/Python releases.
func (p *PreparePlugin) updateCargo(_ context.Context, version domain.Version, repoRoot string) error {
	if !p.cargoEnabled {
		return nil
	}
	info, err := cargo.Detect(p.fs, repoRoot)
	if err != nil {
		return fmt.Errorf("detecting cargo project: %w", err)
	}
	if info == nil {
		return nil
	}

	if err := p.updateCargoManifest(info, version, repoRoot); err != nil {
		return err
	}
	return p.updateCargoLock(info, version, repoRoot)
}

// updateCargoManifest bumps the shared version key in Cargo.toml. It is skipped
// when the user already lists Cargo.toml in version_files (that path handled it)
// or when the manifest has no single shared version key.
func (p *PreparePlugin) updateCargoManifest(info *cargo.Info, version domain.Version, repoRoot string) error {
	if info.VersionKeyPath == "" || p.versionFilesTarget("Cargo.toml") {
		return nil
	}
	path := filepath.Join(repoRoot, "Cargo.toml")
	content, err := p.fs.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	updated, err := updateTOMLKey(content, info.VersionKeyPath, version.String())
	if err != nil {
		return fmt.Errorf("updating version in %s: %w", path, err)
	}
	if err := p.fs.WriteFile(path, updated, fs.FileMode(0o644)); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	p.logger.Info("updated Cargo.toml version", "path", path, "key", info.VersionKeyPath, "version", version)
	return nil
}

// updateCargoLock rewrites the version of each local crate in Cargo.lock. It is
// skipped when there is no lock file or no local crates to update.
func (p *PreparePlugin) updateCargoLock(info *cargo.Info, version domain.Version, repoRoot string) error {
	lockPath := filepath.Join(repoRoot, "Cargo.lock")
	if len(info.CrateNames) == 0 || !p.fs.Exists(lockPath) {
		return nil
	}
	content, err := p.fs.ReadFile(lockPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", lockPath, err)
	}
	crateVersions := make(map[string]string, len(info.CrateNames))
	for _, name := range info.CrateNames {
		crateVersions[name] = version.String()
	}
	updated, err := cargo.UpdateLockVersions(content, crateVersions)
	if err != nil {
		return fmt.Errorf("updating %s: %w", lockPath, err)
	}
	if err := p.fs.WriteFile(lockPath, updated, fs.FileMode(0o644)); err != nil {
		return fmt.Errorf("writing %s: %w", lockPath, err)
	}
	p.logger.Info("updated Cargo.lock crate versions", "path", lockPath, "crates", len(info.CrateNames), "version", version)
	return nil
}

// versionFilesTarget reports whether any version_files entry targets a file with
// the given base name (e.g. "Cargo.toml"), so cargo auto-update can defer to an
// explicit user configuration.
func (p *PreparePlugin) versionFilesTarget(name string) bool {
	for _, entry := range p.versionFiles {
		ve := domain.ParseVersionFileEntry(entry)
		if filepath.Base(ve.Path) == name {
			return true
		}
	}
	return false
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
