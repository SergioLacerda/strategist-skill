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
- Command-tree snapshot test (`cmd/strategist/testdata/command_tree.golden`)
  pinning the path, Use, Short, Long, aliases and flags of every
  `strategist` command; regenerate with
  `go test ./cmd/strategist -run CommandTreeSnapshot -update`

### Changed
- **Stricter `strategist check`:** an absent `Required` normative runtime file
  (`SKILL.md`, `skill.yaml`, `protocol.md`, `templates/agent-protocol.md`, three
  contracts) or an absent generated `agent-protocol.md` is now reported as
  `runtime_missing`, so `check --strict` fails and `check --json` returns
  `status: blocked`. Before, only byte drift was reported and a runtime with
  those files deleted was `ready`. Repair with `strategist install` (or
  `strategist compile` for `agent-protocol.md`)
- **Stricter `strategist validate`:** `active.yaml` is now checked with
  `domain.ActiveConfig.Validate`, the same rules `compile` and `install` enforce,
  so a missing required slot, an unknown `provider_resolution_policy` or an
  invalid `leveling` block now fails validation (they passed before). The
  `pragmatic|epic` mode rule is kept, and every problem is reported in one run.
  Error text for missing `mode`/`base_path`/`slots` changed accordingly
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
- Moved the `plugins` command family (`authorize`, `evaluate-write`,
  `prepare-embedded`) out of `cmd/strategist` `package main` into the
  `cmd/strategist/plugins` adapter package with explicit registration;
  `active.yaml` write-scope resolution now lives in
  `policy.WriteScopeFromActive`. Command paths, flags, output and exit codes
  are unchanged
- Moved the `mission` command family (`start`, `status`, `submit`,
  `context`, `view`, `normalize-openspec`, `report-usage`) out of
  `cmd/strategist` `package main` into the `cmd/strategist/mission` adapter
  package with explicit dependency injection; transitions stay in
  `internal/domain`. Command paths, flags, help text, output and exit codes
  are unchanged
- `mission submit` now rejects the analysis-only terminal events
  `gate_approved_analysis_only` and `handoff_challenge_not_applicable` when
  the mission's `refined/<id>/tasks.md` declares a `documentation_target`,
  so accepted documentation targets reach Sniper instead of being dropped
- Every `mission` subcommand now reports the same `--mission-id` error text
  (`--mission-id is required` / `--mission-id "<id>" is malformed (want
  lowercase letters, digits, and hyphens, …)`) behind its own command
  prefix; previously `start`, `status`, `submit`, `context`, `view` and
  `normalize-openspec` said `must use lowercase letters, digits, and
  hyphens`. Exit codes are unchanged

### Fixed
- `strategist validate` no longer stores the discovered runtime root in its
  `--root` variable, so a later invocation in the same process re-discovers it
- `plugins prepare-embedded --check` drift hint and the embedded-skill
  rollback runbook now name the real command, `strategist plugins
  prepare-embedded` (previously `strategist plugin`)
- Sorted and deduplicated test suite references in the contract index
  generation script
- `resolveInstallableDefaultProviders` now propagates a `loadPluginCatalog`
  failure instead of silently substituting the hardcoded provider fallback
  map, closing a residual ask-first gap flagged in ADR-0035's own context
  (both current callers already guarded this earlier in the call stack, so
  this is a defense-in-depth hardening, not a behavior change on any
  currently reachable path)

### Removed
- Dead handoff-schema lookup plumbing in `check_role_compatibility.go`
  (`loadSupportedHandoffSchemas` and its call sites): computed a value
  `CheckRoleAffinity` never read
- Unreferenced `SlotExtensionLabel` constant in `internal/domain/plugin_types.go`
- `CheckRoleCompatibility` and `ResolveProviderBinding` from
  `internal/domain/role_provider_compatibility.go`: zero production callers
- Legacy `package main` mission shims (`mission_lifecycle.go`, the
  `mission*Cmd` package variables, `runMission*` wrappers and their flag
  helpers): superseded by `cmd/strategist/mission`, test-only callers

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
