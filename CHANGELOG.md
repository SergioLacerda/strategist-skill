# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## Release History Authority

`CHANGELOG.md` is the curated source for unreleased changes and the historical
`1.0.0` baseline. For patch releases after `1.0.0`, GitHub Releases are authoritative
for published release notes and downloadable assets. Do not
backfill `v1.0.x` notes into this file unless they are verified against the
corresponding tag and GitHub Release.

**Note on version numbering (2026-08-31 backfill):** `v1.0.0` and `v1.0.1`
were never tagged or released — the `[1.0.0]` baseline below is a
documentation-only label written into this file the day after the initial
commit, unrelated to the actual git tag sequence, which started
independently at `v0.1.0` and later jumped to `v1.0.2`. Separately, `v0.1.0`,
`v1.0.2`, `v1.0.7`, and `v1.0.12` are real git tags with **no published
GitHub Release** — `v1.0.7` points to the identical commit as `v1.0.8`
(a superseded retry), and `v1.0.12` was superseded the same day by `v1.0.13`;
both are recorded below without a dedicated bullet list where no
independently verifiable content could be separated from the release that
superseded them. `v1.0.2`'s tagged commit is also chronologically *later*
than `v1.0.3`/`v1.0.4`/`v1.0.5` despite its lower number, a topology
artifact of tagging pre-merge branch tips rather than a backdated release.

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

## [1.0.13] - 2026-08-05

### Fixed
- Removed an unused `vi` import from the TabSwitcher test suite

---

## [1.0.12] - 2026-08-04

> Tag exists in git; no GitHub Release was ever published for it (see the
> version-numbering note above). Content below is verified directly from
> the commits in this tag's range, not from a Release body.

### Added
- Automated evaluation harness with contract testing and prompt-based
  artifact validation; Makefile modularization and `strategist eval` CLI
  subcommands, with accompanying ADR documentation
- LM Studio reachability preflight for `eval-promptfoo`
- Jewel evidence quality validation and advisory check CLI command
- Handoff challenge metrics, persistence, and jewel challenge template
  validation
- Runbook execution engine and comprehensive operational runbook
  documentation
- Handoff verification and mission quality domain logic, with associated
  CI/CD validation scripts and tests

### Changed
- Modularized eval harvest logic (selection, copying, content-assertion
  helpers) and mission harvesting/validation logic into separate helper
  files
- Modularized handoff validation and runbook selection logic; added
  treasure chest type definitions
- Modularized validation and calculation logic for readability and
  maintainability

---

## [1.0.11] - 2026-07-30

### Changed
- Centralized CI workflow steps into Makefile targets; added artifact
  release verification scripts

---

## [1.0.10] - 2026-07-29

### Changed
- Reliable artifact collection and verification in the release workflow (CI)

---

## [1.0.9] - 2026-07-29

### Maintenance
- Dependency bumps only (Go, GitHub Actions, and npm groups) — no
  user-facing change

---

## [1.0.8] - 2026-07-29

### Added
- Single-source corpus: retired the `strategist/` authoring tree, added an
  errors catalog, corpus lint, and pointer collapse
- Critic-at-gate, Riposte capture, Keen Senses radar, and Opportunity
  Attack jewel harvest
- Identity check, token-economy loading, dojo/skill-shape docs, and FSM
  gate-bypass guard
- Canonical mission-status vocabulary and FSM alignment
- Handoff-contract and retrieval cascade for Ranger→Archivist context
  management
- New dojo checker pipeline, domain compilation, and comprehensive test
  suite infrastructure
- Robust integrity checks and source metadata tracking for strategist
  configuration
- Robust scanning and governance loading for treasure chests and missions
- Treasure-chest `remove` command and mine `list`; unified flag constants;
  modularized install/validation logic
- Runbook opportunity detection integrated into the quick-draw pipeline,
  with accompanying documentation and tools
- Runbook-opportunity contract; refactored Riposte to use independent
  normalization and capture phases
- Targeted remediation hints for specific treasure-chest drift types

### Changed
- Centralized jewel rendering logic; comprehensive treasure-chest test
  suite
- Enforced role boundaries between Opportunity Attack and Critical Hit;
  formalized ADR telemetry tripwires and documentation-language
  requirements
- Enforced Sniper as the native execution provider; integrated jewel ID
  tracking into outcome reporting
- Migrated agent-awareness logic; added new validation test suites

### Fixed
- Made git-repo detection version/locale-independent in the F3 conflict
  signal
- Registered missing machine contracts; repaired test subject paths
- Made wizard config substitution idempotent
- Closed 6 of 8 implementation-review gaps; surfaced GAP-8

> `v1.0.7` is a duplicate tag pointing at this exact commit — its release
> attempt was superseded by this one; there is no separate content to list.

---

## [1.0.6] - 2026-07-19

### Added
- `--format json` for treasure-chest jewel list; jewel `list`/`show`
  commands (table output, default status filter)
- Post-route discovery capability check, validating weapon subtype support
  before invocation

### Changed
- Improved runtime default parity tracking with manifest-based validation;
  modularized cache and telemetry file handling
- Moved treasure-chest logic into an internal package; strengthened
  protocol documentation
- Removed the legacy `roles_config` field from `active.yaml`

---

## [1.0.5] - 2026-07-14

No user-facing change beyond `v1.0.4` — 5 commits in range, none
changelog-worthy per this file's own curation filter.

---

## [1.0.4] - 2026-07-14

### Added
- Strategist Active header and security role-lock constraints in the agent
  codex seed
- `jewel_retrieval` contract with mandatory `source_cards` fallback, with
  provenance documentation
- Strict agent role isolation, preventing direct execution drift (Role
  Lock contracts)
- Treasure chest management CLI; Scout skill; manifest verification with
  domain modeling

---

## [1.0.3] - 2026-07-10

### Added
- ENTRYPOINT section in `SKILL.md`; `agent-protocol.md` template
- Agent protocol compilation and awareness tracking
- Status banner and integrity checks for active config
- Implementation Short Route and delegated local execution context
  requirements

### Changed
- Renamed "hunter" drift IDs to "sniper"; renamed review gate to approval
  gate
- Simplified mission policy evaluation; enforced canonical provider
  resolution paths
- Streamlined the opportunity attack routine; enhanced governance
  injection and delegation capability checks
- Removed agent awareness/protocol logic from core compilation modules
  (superseded by dedicated protocol compilation)

---

## [1.0.2] - 2026-07-16

> Tag exists in git; no GitHub Release was ever published for it (see the
> version-numbering note above). Its commit shares history with
> `v1.0.3`/`v1.0.4` through pre-merge branch-tip tagging rather than a
> linear sequence, so no content could be separated out here without
> double-counting what `v1.0.4` already lists above — no independent bullet
> list is given.

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
