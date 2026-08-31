# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## Release History Authority

`CHANGELOG.md` is the curated source for unreleased changes and the historical
`1.0.0` baseline. For patch releases after `1.0.0`, GitHub Releases are authoritative
for published release notes and downloadable assets. Do not
backfill `v1.0.x` notes into this file unless they are verified against the
corresponding tag and GitHub Release.

---

## [Unreleased]

### Added
- `strategist upgrade` command: backup-protected file application, expanded
  policy validation, and updated runtime discovery protocols
- `InstallWithReport`, exposing backup directory paths to CLI users
- Native role invocation, automated runtime configuration merging, and
  enhanced telemetry and enforcement policies
- Plugin architecture, telemetry routing, and infrastructure for governance
  system integration and strategist skill management
- Telemetry event sinking system and expanded domain configuration
  validation logic
- `runbook select` CLI command, scoring local runbook sidecars against
  mission signals
- Counterfactual verification and forbidden-claim safety checks in handoff
  logic and CLI
- E2E integration test harness; routing and discovery workflow contracts
- Deterministic golden testing suite with automated governance gates and
  documentation drift prevention
- `CriticalHitTrigger` evaluation logic and corresponding test scenarios
- Integration-style coverage tracking; telemetry and treasure-chest E2E
  scenarios
- Auto-generated contract and schema documentation indices, generated from
  source files

### Changed
- Documentation and generation scripts now point to source files instead
  of gitignored build artifacts
- Migrated treasure CLI logic into `internal/treasurecli`, decomposed into
  smaller helpers
- Introduced `internal/check` package; migrated test helpers for
  modularity and coverage
- Propagated context and added OpenTelemetry instrumentation to
  installation and wizard workflows
- Expanded gated CI metrics; added treasure chest grading evals and
  Critical Hit closure specs

### Fixed
- Sorted and deduplicated test suite references in the contract index
  generation script

---

## [1.0.0] - 2026-05-28

### Added
- Core compilation scripts and contract definitions for strategist configuration and indexing management
- Dungeon documentation pages and GitHub Pages deployment workflow (`pages.yml`)
- Architecture and integration flow diagrams (`docs/`)
- `validate_provider()` function and context hints to install wizard
- Slot write contracts: `write_pending` (Ranger) and `write_analysis` (Archivist)
- `opportunity_attack` phase and side quest pipeline (phases 5b–5d)
- Design spec for side quest ataque de oportunidade pipeline
- Design spec for slot risk contract fix and known-providers registry
- Curl installer (`bootstrap.sh` / `bootstrap.ps1`) with GitHub Actions release workflow (`release.yml`)
- Design spec for curl installer and GitHub Actions release CI/CD
- `install.sh` generates `.strategist/` runtime and registers agent shims across Claude, Gemini, Codex
- Implementation plan and design spec for multi-agent skill registration via `.strategist install`
- `.analysis/` workspace directories tracked in git (`pending/`, `refined/`, `archived/`)

### Changed
- Renamed core persona roles: Scout → Ranger, Engineer → Archivist, Hunter → Sniper
- Removed legacy analysis directory and simplified bootstrap installation flow
- Replaced verbose documentation with concise technical overview of mission orchestration

### Fixed
- Pinned all GitHub Actions to commit SHA to prevent supply-chain attacks
- Set executable permission on `bootstrap.sh` and `bootstrap.ps1`
- `.gitignore` updated to allow tracking `.analysis/pending/`, `refined/`, `archived/`

### Removed
- Obsolete strategist-mission-pipeline design docs and specifications
- Pending design specs (superseded by implementations)
