# ADR 004: Project-Scoped Tags

## Status
Accepted

## Context
In monorepos with independent versioning, each project needs its own version tag. We need a consistent, configurable tag naming scheme.

## Decision
Use Go templates for tag formatting, with two templates:
- `tag_format` for repo-level tags (default: `v{{.Version}}`)
- `project_tag_format` for project-scoped tags (default: `{{.Project}}/v{{.Version}}`)

The `TagService` port handles both formatting (creating tag names) and parsing (extracting project + version from existing tags). It supports multiple conventions: slash-separated (`api/v1.2.3`), at-sign (`api@1.2.3`), and custom templates.

## Consequences
- Tags are fully configurable per repository
- Both formatting and parsing use the same service, ensuring consistency
- Finding the latest tag for a project involves parsing all tags and filtering by project name
- Tag parsing supports multiple conventions simultaneously for maximum compatibility
