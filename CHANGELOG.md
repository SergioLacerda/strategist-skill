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
- SHA256 checksum verification in `bootstrap.sh` for versioned releases
- Security warning when installing from branch ref without integrity check
- Install rollback mechanism in `strategist/install.sh` (`INSTALL_MANIFEST` + `trap ERR`)
- YAML config validation step in Strategist preflight (`2a.validate`)
- New schemas: `active.schema.yaml`, `roles.schema.yaml`, `slot-output.schema.yaml`
- Slot output contract validation after Ranger and Archivist phases
- Test harness with 5 golden-file fixtures for critical contract scenarios
- CI workflow `test.yml` with shellcheck, fixture tests, and schema validation
- `SHA256SUMS` asset generation in `release.yml`
- Promptfoo-based external artifact quality review harness (`promptfoo/`),
  with a Makefile target guarded by a reachability preflight so a missing
  local LM Studio server fails fast with a clear message instead of a raw
  fetch error
- Automated Go-native evaluation harness (`tests/evals/`) with contract
  testing and prompt-based artifact content validation
- `strategist eval harvest`/`select`/`copy` CLI subcommands for building
  eval fixtures from real mission artifacts, with accompanying ADR
  documentation
- Jewel evidence quality validation and advisory check CLI command
- Handoff challenge metrics, persistence, and jewel challenge template
  validation
- Runbook execution engine and operational runbook documentation, including
  a local CI/CD release gate validation runbook
- Handoff verification and mission quality domain logic, with associated
  CI/CD validation scripts and tests

### Changed
- `protocol.md`: normalized `risk_score` vocabulary to `write_pending` / `write_analysis` / `controlled`
- `readme.md`: added security callout for curl pipe installation
- Modularized handoff validation and runbook selection logic; added
  treasure chest type definitions
- Modularized eval harvest logic (selection, copying, content-assertion
  helpers) and mission harvesting/validation logic into separate helper
  files
- Modularized validation and calculation logic for readability and
  maintainability
- Test execution now uses an explicit project root path instead of
  implicit auto-discovery, for hermetic test runs

### Maintenance
- Dependency bumps: GitHub Actions group, `jsdom` in `web/landing`
- Whitespace formatting fix in jewel evidence quality test cases

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
