# ADR 003: Monorepo Project Detection

## Status
Accepted

## Context
The tool must support multiple monorepo layouts:
- Go workspaces (`go.work`)
- Multiple nested `go.mod` files without a workspace
- Manually configured projects via YAML

## Decision
Implement four discovery strategies behind the `ProjectDiscoverer` port:
1. **WorkspaceDiscoverer** — parses `go.work` `use` directives
2. **ModuleDiscoverer** — recursively finds `go.mod` files
3. **ConfiguredDiscoverer** — reads project definitions from config
4. **CmdDiscoverer** — discovers services in single-module monorepos with a `cmd/<service>/main.go` layout; promoted shared `pkg/<name>` packages become library projects

These are composed via `CompositeDiscoverer` which tries each in priority order (config > workspace > modules) and returns the first non-empty result. `CmdDiscoverer` is opt-in via the `discover_cmd: true` config flag; it is not included in the default composition because it targets a specific layout that not all single-module repos use.

`CmdDiscoverer` activation conditions:
- `go.mod` exists at the repository root (single-module repo)
- `go.work` does **not** exist (not a Go workspace)
- `cmd/` directory exists at the repository root

## Consequences
- Supports all four monorepo cases without special-casing
- New discovery strategies can be added by implementing the interface
- Strategy priority is configurable by changing the composition order
- Path-based impact analysis maps changed files to projects using file path prefixes
- Opt-in strategies (`discover_cmd`) avoid misidentifying non-matching repos
