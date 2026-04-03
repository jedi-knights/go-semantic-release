# semantic-release

A production-grade semantic release utility written in native Go. Analyzes conventional commits, determines the next semantic version, generates changelogs, creates tags, and publishes GitHub releases.

Supports monorepos with independent project versioning, including Go workspaces and multi-module repositories.

## Features

- **Conventional Commits analysis** — parses commit messages to determine release types (major, minor, patch)
- **Semantic versioning** — calculates next version based on commit impact
- **Monorepo support** — independent versioning per project with three discovery modes:
  - Go workspaces (`go.work`)
  - Multiple nested `go.mod` modules
  - Config-defined projects with path mappings
- **Branch policies** — stable releases on main, prereleases on beta/alpha/next
- **Changelog generation** — Markdown release notes grouped by commit type
- **GitHub Releases** — creates releases via the GitHub API
- **Dry-run mode** — preview releases without any mutations
- **Dependency propagation** — optionally trigger dependent project releases
- **Pluggable architecture** — ports and adapters pattern for full extensibility

## Installation

```bash
go install github.com/jedi-knights/go-semantic-release/cmd/semantic-release@latest
```

Or build from source:

```bash
git clone https://github.com/jedi-knights/go-semantic-release.git
cd go-semantic-release
go build -o bin/semantic-release ./cmd/semantic-release
```

## Quick Start

```bash
# Initialize config
semantic-release config init

# Perform a release (default action, same as original semantic-release)
semantic-release

# Dry run
semantic-release --dry-run

# See what would happen (extended command)
semantic-release plan

# Preview the next version
semantic-release version

# Generate changelog
semantic-release changelog
```

## Usage

Running `semantic-release` with no subcommand performs the release — this matches the original [semantic-release](https://github.com/semantic-release/semantic-release) behavior exactly.

```bash
semantic-release [options]
```

### CLI Flags (compatible with semantic-release)

| Flag | Short | Description |
|------|-------|-------------|
| `--branches` | `-b` | Git branches to release from |
| `--repository-url` | `-r` | Git repository URL |
| `--tag-format` | `-t` | Git tag format |
| `--plugins` | `-p` | Plugins |
| `--extends` | `-e` | Shareable configurations |
| `--dry-run` | `-d` | Skip publishing |
| `--ci` | | Toggle CI verifications |
| `--no-ci` | | Skip CI verifications |
| `--debug` | | Output debugging information |

### Extension Flags (Go-specific)

| Flag | Description |
|------|-------------|
| `--config` | Path to config file (default: `.semantic-release.yaml`) |
| `--project` | Target a specific project in a monorepo |
| `--json` | Output in JSON format |

### Extension Subcommands

These are additional commands beyond the original semantic-release:

| Command | Description |
|---------|-------------|
| `semantic-release plan` | Show the release plan without executing |
| `semantic-release version` | Display current and next version |
| `semantic-release changelog` | Generate release notes |
| `semantic-release detect-projects` | List discovered projects |
| `semantic-release verify` | Check release prerequisites |
| `semantic-release config init` | Create a default config file |

## Configuration

Configuration is loaded from `.semantic-release.yaml`, `.releaserc.yaml`, `.releaserc.json`, or `release.config.yaml`, plus environment variables (prefix `SEMANTIC_RELEASE_`), and CLI flags. CLI flags take precedence over config files.

When not running in CI, dry-run mode is enabled by default. Use `--no-ci` to run locally without dry-run.

### Repository-wide release (default)

```yaml
release_mode: repo
tag_format: "v{{.Version}}"
```

### Independent project versioning

```yaml
release_mode: independent
project_tag_format: "{{.Project}}/v{{.Version}}"

projects:
  - name: api
    path: services/api
    tag_prefix: "api/"
  - name: worker
    path: services/worker
    tag_prefix: "worker/"
    dependencies:
      - shared
  - name: shared
    path: pkg/shared
    tag_prefix: "shared/"

dependency_propagation: true
```

### Auto-discovery with go.work

```yaml
release_mode: independent
# Projects are auto-discovered from go.work
```

### Auto-discovery with nested go.mod

```yaml
release_mode: independent
discover_modules: true
```

### Branch policies

```yaml
branches:
  - name: main
    is_default: true
  - name: next
    prerelease: true
    channel: next
  - name: next-major
    prerelease: true
    channel: next-major
  - name: beta
    prerelease: true
    channel: beta
  - name: alpha
    prerelease: true
    channel: alpha
```

Maintenance branches (e.g., `1.x`, `1.0.x`) are auto-detected by name pattern. You can also configure them explicitly:

```yaml
branches:
  - name: "1.x"
    range: "1.x"
    channel: "release-1.x"
    branch_type: maintenance
  - name: "1.0.x"
    range: "1.0.x"
    channel: "release-1.0.x"
    branch_type: maintenance
  - name: main
    is_default: true
```

Maintenance branches enforce version range constraints — a `1.0.x` branch only allows patch bumps, and a `1.x` branch allows patch and minor bumps but not major.

### Prepare step (file updates)

```yaml
prepare:
  changelog_file: CHANGELOG.md
  version_file: VERSION
```

When configured, the prepare step updates `CHANGELOG.md` (prepending new entries) and `VERSION` (with the new version string) before the release is published.

### Git identity

```yaml
git_author:
  name: semantic-release-bot
  email: semantic-release-bot@users.noreply.github.com
```

### GitHub settings

```yaml
github:
  create_release: true
  owner: jedi-knights
  repo: go-semantic-release
  draft_release: false
  assets:
    - "dist/*.tar.gz"
    - "dist/*.zip"
  success_comment: "🎉 Released in {{.Version}}"
  released_labels:
    - released
  fail_labels:
    - semantic-release
  discussion_category_name: "Announcements"
  # token: set via GH_TOKEN, GITHUB_TOKEN, or SEMANTIC_RELEASE_GITHUB_TOKEN
```

## Architecture

semantic-release follows **Hexagonal Architecture** (Ports and Adapters) with clear separation:

```
cmd/semantic-release/  # CLI entry point
internal/
  domain/              # Pure business logic, no dependencies
  ports/               # Interface definitions (ports)
  app/                 # Application services (use cases)
  adapters/            # Implementations (adapters)
    git/               # Git CLI, commit parser, tag service, project discovery
    github/            # GitHub release publisher
    config/            # Viper config provider
    cli/               # Cobra CLI commands
    fs/                # Filesystem adapter
    changelog/         # Changelog template generator
    template/          # Go template renderer
  di/                  # Dependency injection container
  platform/            # Cross-cutting concerns (logger, clock)
```

### Release Lifecycle

The release pipeline follows the same 9-step lifecycle as [semantic-release](https://github.com/semantic-release/semantic-release):

| Step | Description | Plugins |
|------|-------------|---------|
| **verifyConditions** | Check prerequisites (git access, GitHub token) | git, github |
| **analyzeCommits** | Determine release type from conventional commits | commit-analyzer |
| **verifyRelease** | Validate the pending release | (extensible) |
| **generateNotes** | Create release notes from commits | release-notes-generator |
| **prepare** | Update CHANGELOG.md, VERSION files | prepare-files |
| **publish** | Create git tag, push, create GitHub release | git, github |
| **addChannel** | Update release prerelease status on GitHub | github |
| **success** | Comment on PRs/issues, apply labels | github |
| **fail** | Open/update failure issue on GitHub | github |

Each step is implemented as a plugin interface. Multiple plugins can implement the same step — for `analyzeCommits`, the highest release type wins; for `generateNotes`, outputs are concatenated. In dry-run mode, steps after `generateNotes` are skipped.

### Monorepo Support

| Case | Detection | Tags |
|------|-----------|------|
| Go workspace (`go.work`) | Parses `use` directives | `project/vX.Y.Z` |
| Nested `go.mod` | Recursive file scan | `project/vX.Y.Z` |
| Config-defined | `.semantic-release.yaml` projects | Configurable prefix |
| Single module | Root `go.mod` | `vX.Y.Z` |

## Development

### Prerequisites

- Go 1.21+
- [Task](https://taskfile.dev/) (optional, for task runner)
- [golangci-lint](https://golangci-lint.run/) (for linting)
- [mockgen](https://github.com/uber-go/mock) (for mock generation)

### Common Tasks

```bash
task build          # Build binary
task test           # Run all tests
task test:unit      # Run tests with race detection
task lint           # Run linter
task fmt            # Format code
task generate:mocks # Regenerate mocks
task coverage       # Generate coverage report
task ci             # Run full CI pipeline
```

### Testing Approach

- **Table-driven tests** with subtests throughout
- **Uber Go Mock** for all port interfaces
- Tests cover: version calculation, commit parsing, breaking change detection, project discovery, go.work parsing, tag formatting, impact analysis, dependency propagation, dry-run behavior, branch policies
- All external system access is behind ports for full testability

### Project Structure

```
internal/domain/         # Version, Commit, Project, ReleaseType, etc.
internal/ports/          # GitRepository, CommitParser, TagService, etc.
internal/ports/mocks/    # Generated mocks
internal/app/            # VersionCalculator, ReleasePlanner, ReleaseExecutor
internal/adapters/git/   # Git CLI, conventional commit parser, discoverers
internal/adapters/github # GitHub release publisher
internal/di/             # DI container wiring
```

## Roadmap

- [x] Plugin lifecycle pipeline (9 steps)
- [x] Changelog file writing (CHANGELOG.md) via prepare step
- [x] Version file updates (VERSION) via prepare step
- [x] CI environment auto-detection
- [x] Maintenance branch support with version ranges
- [x] GitHub PR/issue commenting, failure issues, asset uploads
- [x] GitLab/Bitbucket adapters
- [x] go-git adapter (no CLI dependency)
- [x] Glob-based path include/exclude filtering
- [x] Shareable configuration loading (`--extends`)
- [x] External plugin loading (`--plugins`)
- [x] Commit message linting
- [x] Interactive mode for release confirmation

## References

- [Conventional Commits](https://www.conventionalcommits.org/) — the commit message specification used to determine release types
- [Semantic Versioning](https://semver.org/) — the versioning scheme used for releases

## License

MIT
