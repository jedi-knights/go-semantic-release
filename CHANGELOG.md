# Changelog

## 0.8.7 (2026-04-09)

### Bug Fixes

- **ci:** narrow goreleaser trigger to semver tags only (007842d)

## 0.8.6 (2026-04-09)

### Bug Fixes

- **release:** make tag creation idempotent on workflow re-runs (01edac4)

## 0.8.5 (2026-04-08)

### Bug Fixes

- **action:** remove secrets context from input description string (9142b15)

## 0.8.4 (2026-04-08)

### Bug Fixes

- **review:** address post-merge code review findings (a7db2a7)

## 0.8.3 (2026-04-08)

### Bug Fixes

- **ci:** remove invalid goreleaser v2 release.github.repo field (2517c74)

## 0.8.2 (2026-04-08)

### Bug Fixes

- **ci:** pass GH_TOKEN to checkout so tag push triggers goreleaser (c343526)

## 0.8.1 (2026-04-08)

### Bug Fixes

- **ci:** use GH_TOKEN PAT so tag push triggers goreleaser workflow (18502ff)

## 0.8.0 (2026-04-08)

### Features

- **discovery:** add cmd/ layout discoverer for single-module monorepos (e936372)

## 0.7.0 (2026-04-08)

### Features

- **release:** add GoReleaser cross-platform builds and composite GitHub Action (022c709)

## 0.6.1 (2026-04-05)

### Bug Fixes

- **lint:** use index-based ranging to avoid 216-byte Commit copies (6c52cd6)
- **planner:** scope commits to since-last-tag in independent and repo modes (cbc4551)

## 0.6.0 (2026-04-04)

### Features

- implement all remaining roadmap items (28e4308)
- add full semantic-release lifecycle and feature parity (39a452e)
- implement go semantic release utility (bb1264b)

### Bug Fixes

- **ci:** resolve golangci-lint failures (b4287d4)
- run prepare plugins before release execution (c25eace)
- use repo tag format for root project and fetch tags in release (08280f7)
- resolve GitHub token from env in publisher and fix release workflow (acf33fc)
- make CLI 100% compatible with semantic-release (d7a3c7f)

## 0.5.0 (2026-04-04)

### Features

- implement all remaining roadmap items (28e4308)
- add full semantic-release lifecycle and feature parity (39a452e)
- implement go semantic release utility (bb1264b)

### Bug Fixes

- **ci:** resolve golangci-lint failures (b4287d4)
- run prepare plugins before release execution (c25eace)
- use repo tag format for root project and fetch tags in release (08280f7)
- resolve GitHub token from env in publisher and fix release workflow (acf33fc)
- make CLI 100% compatible with semantic-release (d7a3c7f)

## 0.4.0 (2026-04-03)

### Features

- implement all remaining roadmap items (28e4308)
- add full semantic-release lifecycle and feature parity (39a452e)
- implement go semantic release utility (bb1264b)

### Bug Fixes

- run prepare plugins before release execution (c25eace)
- use repo tag format for root project and fetch tags in release (08280f7)
- resolve GitHub token from env in publisher and fix release workflow (acf33fc)
- make CLI 100% compatible with semantic-release (d7a3c7f)

## 0.3.0 (2026-04-03)

### Features

- implement all remaining roadmap items (28e4308)
- add full semantic-release lifecycle and feature parity (39a452e)
- implement go semantic release utility (bb1264b)

### Bug Fixes

- run prepare plugins before release execution (c25eace)
- use repo tag format for root project and fetch tags in release (08280f7)
- resolve GitHub token from env in publisher and fix release workflow (acf33fc)
- make CLI 100% compatible with semantic-release (d7a3c7f)

## 0.2.0 (2026-04-03)

### Features

- implement all remaining roadmap items (28e4308)
- add full semantic-release lifecycle and feature parity (39a452e)
- implement go semantic release utility (bb1264b)

### Bug Fixes

- run prepare plugins before release execution (c25eace)
- use repo tag format for root project and fetch tags in release (08280f7)
- resolve GitHub token from env in publisher and fix release workflow (acf33fc)
- make CLI 100% compatible with semantic-release (d7a3c7f)
