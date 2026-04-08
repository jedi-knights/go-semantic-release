# GitHub Marketplace Publishing

This document explains how to publish the `go-semantic-release` composite action to the GitHub Marketplace and how the release pipeline keeps the listing up to date automatically.

## How Marketplace publishing works

The Marketplace listing is tied to the **repository**, not to individual releases. GitHub watches all tags and releases in the repo. Once the initial listing is published, every new GitHub release automatically appears as a new version on the Marketplace — no manual steps required for subsequent releases.

The current release pipeline already handles this end-to-end:

```
push to main
  → release.yml: semantic-release creates a tag and commits CHANGELOG.md/VERSION
  → goreleaser.yml: builds binaries and creates the GitHub release
  → GitHub Marketplace: picks up the new release automatically
```

## Initial listing (one-time manual step)

The first publication must be done through the GitHub web UI:

1. Go to the repository on GitHub.
2. Click **Releases → Draft a new release**.
3. Check **"Publish this Action to the GitHub Marketplace"**.
4. Accept the GitHub Marketplace developer agreement.
5. Publish the release.

GitHub validates `action.yml` at this point. The required fields are all present:

| Field | Value |
|-------|-------|
| `name` | `Semantic Release by Jedi Knights` |
| `description` | `Run semantic-release using pre-built binaries — no Go toolchain required` |
| `branding.icon` | `tag` |
| `branding.color` | `blue` |

After this one-time step, every GoReleaser release is automatically reflected on the Marketplace. No changes to `.goreleaser.yml` or `release.yml` are needed.

## Floating major-version tags

Marketplace users typically pin to a floating major tag (`@v1`) rather than a specific patch (`@v1.2.3`). This lets them receive non-breaking updates automatically without changing their workflow files.

GoReleaser does not manage floating tags automatically. Add the following step to `.github/workflows/release.yml` after the GoReleaser run to maintain the `@v<major>` tag on every release:

```yaml
- name: Update floating major tag
  if: steps.plan.outputs.has_release == 'true'
  env:
    GITHUB_TOKEN: ${{ secrets.GH_TOKEN }}
  run: |
    VERSION=$(cat VERSION)
    MAJOR="v${VERSION%%.*}"
    git tag -f "$MAJOR"
    git push origin "$MAJOR" --force
```

This force-updates the `@v1` tag to point at the latest `v1.x.x` release. The tag is created locally from the current `VERSION` file (written by semantic-release) and pushed with `--force` to overwrite the previous pointer.

> **Note:** Only update the floating tag after GoReleaser has successfully created the release. Placing the step after GoReleaser ensures the binaries and release notes are already attached before users start pinning to the new version.

## Verifying a Marketplace release

After a release completes, confirm the Marketplace listing is current:

1. Visit `https://github.com/marketplace/actions/<action-slug>`.
2. Check that the latest version matches the new tag.
3. Confirm the version selector includes all expected tags.

If a release does not appear, check that:
- The tag follows semver (`v1.2.3`) — non-semver tags are ignored by the Marketplace.
- The GitHub release is not marked as a draft (`draft: false` in `.goreleaser.yml`).
- The `action.yml` at the tagged commit is valid (name, description, branding all present).
