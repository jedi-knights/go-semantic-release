# ADR 003: Monorepo Project Detection

## Status
Accepted

## Context
The tool must support multiple monorepo layouts:
- Go workspaces (`go.work`)
- Multiple nested `go.mod` files without a workspace
- Manually configured projects via YAML

## Decision
Implement three discovery strategies behind the `ProjectDiscoverer` port:
1. **WorkspaceDiscoverer** — parses `go.work` `use` directives
2. **ModuleDiscoverer** — recursively finds `go.mod` files
3. **ConfiguredDiscoverer** — reads project definitions from config

These are composed via `CompositeDiscoverer` which tries each in priority order (config > workspace > modules) and returns the first non-empty result.

## Consequences
- Supports all three monorepo cases without special-casing
- New discovery strategies can be added by implementing the interface
- Strategy priority is configurable by changing the composition order
- Path-based impact analysis maps changed files to projects using file path prefixes
