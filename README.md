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

# See what would happen
semantic-release plan

# Preview the next version
semantic-release version

# Generate changelog
semantic-release changelog

# Perform a release
semantic-release release

# Dry run
semantic-release release --dry-run
```

## Commands

| Command | Description |
|---------|-------------|
| `semantic-release release` | Analyze commits, tag, and publish a release |
| `semantic-release release --project api` | Release a specific project in a monorepo |
| `semantic-release plan` | Show the release plan without executing |
| `semantic-release version` | Display current and next version |
| `semantic-release changelog` | Generate release notes |
| `semantic-release detect-projects` | List discovered projects |
| `semantic-release verify` | Check release prerequisites |
| `semantic-release config init` | Create a default config file |

### Global Flags

| Flag | Description |
|------|-------------|
| `--config` | Path to config file (default: `.semantic-release.yaml`) |
| `--dry-run` | Preview without mutations |
| `--project` | Target a specific project |
| `--json` | Output in JSON format |

## Configuration

Configuration is loaded from `.semantic-release.yaml`, environment variables (prefix `SEMANTIC_RELEASE_`), and CLI flags.

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
  - name: beta
    prerelease: true
    channel: beta
  - name: alpha
    prerelease: true
    channel: alpha
```

### GitHub settings

```yaml
github:
  create_release: true
  owner: jedi-knights
  repo: go-semantic-release
  # token: set via SEMANTIC_RELEASE_GITHUB_TOKEN env var
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

### Release Pipeline

1. **Verify conditions** — check branch policy, GitHub config
2. **Discover projects** — find projects via workspace/modules/config
3. **Analyze commits** — parse conventional commits since last tag
4. **Map impact** — associate changed files to projects
5. **Calculate versions** — determine next version per project
6. **Generate notes** — create markdown changelogs
7. **Create tags** — git tag + push
8. **Publish** — create GitHub release

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

- [ ] Plugin system for custom pipeline steps
- [ ] Changelog file writing (CHANGELOG.md)
- [ ] Version file updates (VERSION, package.json)
- [ ] GitLab/Bitbucket adapters
- [ ] go-git adapter (no CLI dependency)
- [ ] Glob-based path include/exclude filtering
- [ ] Release-please compatibility mode
- [ ] Commit message linting
- [ ] Interactive mode for release confirmation

## License

MIT
