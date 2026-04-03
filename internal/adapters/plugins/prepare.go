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
	fs              ports.FileSystem
	logger          ports.Logger
	changelogFile   string // path to CHANGELOG.md, empty to skip
	versionFile     string // path to VERSION file, empty to skip
	additionalFiles []string
}

// PrepareConfig holds configuration for the prepare plugin.
type PrepareConfig struct {
	ChangelogFile   string   `mapstructure:"changelog_file"`
	VersionFile     string   `mapstructure:"version_file"`
	AdditionalFiles []string `mapstructure:"additional_files"`
}

// NewPreparePlugin creates a plugin that updates release files.
func NewPreparePlugin(fs ports.FileSystem, logger ports.Logger, cfg PrepareConfig) *PreparePlugin {
	return &PreparePlugin{
		fs:              fs,
		logger:          logger,
		changelogFile:   cfg.ChangelogFile,
		versionFile:     cfg.VersionFile,
		additionalFiles: cfg.AdditionalFiles,
	}
}

func (p *PreparePlugin) Name() string { return "prepare-files" }

func (p *PreparePlugin) Prepare(_ context.Context, rc *domain.ReleaseContext) error {
	if rc.CurrentProject == nil {
		return nil
	}

	version := rc.CurrentProject.NextVersion

	if err := p.updateVersionFile(version, rc.RepositoryRoot); err != nil {
		return err
	}

	if err := p.updateChangelog(rc); err != nil {
		return err
	}

	return nil
}

func (p *PreparePlugin) updateVersionFile(version domain.Version, repoRoot string) error {
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

func (p *PreparePlugin) updateChangelog(rc *domain.ReleaseContext) error {
	if p.changelogFile == "" {
		return nil
	}

	path := filepath.Join(rc.RepositoryRoot, p.changelogFile)
	newEntry := rc.Notes

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
		return lines[0] + "\n\n" + newEntry + "\n\n" + strings.TrimLeft(rest, "\n")
	}

	return newEntry + "\n\n" + existing
}
