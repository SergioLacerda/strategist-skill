# Governance Compatibility Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make Strategist provider-agnostic for governance while supporting Providence as an explicit reference adapter and retiring implicit SDD compatibility.

**Architecture:** Keep the consumer-owned governance contracts in `internal/governance`, expose a normalized source/snapshot boundary, and isolate Providence filesystem parsing in an adapter package. Keep the CLI, bridge, telemetry, catalogs, templates, and generated mirrors dependent on the generic vocabulary; invalid explicit sources fail closed and no named directory is auto-detected.

**Tech Stack:** Go 1.27.1, Cobra, JSON/YAML, OpenTelemetry, embedded defaults, shell generation gates, Go tests.

---

### Task 1: Record the breaking boundary

**Files:**
- Create: `docs/adr/0062-governance-compatibility-implementation.md`
- Modify: `CHANGELOG.md`
- Modify: `docs/adr/README.md`

**Step 1:** Record the repository's next flat release label (`v1.0.28`), the removal of SDD-specific flags/aliases/fallbacks, and the required CHANGELOG `Removed`/`Changed` entries. ADR-0055 is already occupied by an unrelated accepted record, so this implementation ADR uses the next free number.

**Step 2:** Run `bash scripts/check-doc-links.sh` and inspect the ADR/index diff.

### Task 2: Add the generic governance source contract

**Files:**
- Create: `internal/governance/source.go`
- Create: `internal/governance/source_test.go`
- Modify: `internal/governance/sync.go`
- Modify: `internal/governance/sync_governance_source.go`

**Step 1:** Add normalized source, snapshot, scope/correlation, validation, and fail-closed error types.

**Step 2:** Write tests for standalone absence, valid explicit sources, malformed metadata, missing fingerprint, missing required mandates, uncorrelated input, and insufficient authority.

**Step 3:** Refactor sync to consume the generic contract without implicit filesystem discovery or legacy fingerprint fallback.

### Task 3: Implement the Providence adapter

**Files:**
- Create: `internal/governance/providence/adapter.go`
- Create: `internal/governance/providence/adapter_test.go`

**Step 1:** Parse only explicit Providence input, `fingerprints.combined`, and required `MANDATE` entries from `source/governance-core.json`.

**Step 2:** Add deterministic ordering and rejection tests for malformed, uncorrelated, and insufficient-authority sources.

### Task 4: Generalize CLI, bridge, and telemetry

**Files:**
- Modify: `cmd/strategist/sync_governance.go`
- Modify: `cmd/strategist/sync_governance_test.go`
- Modify: `cmd/strategist/sync_governance_report.go`
- Modify: `internal/governance/bridge_adapter.go`
- Modify: `internal/governance/bridge_adapter_test.go`
- Modify: `internal/telemetry/*` only where SDD authority labels are emitted

**Step 1:** Replace `--sdd` and its default with an explicit generic governance input.

**Step 2:** Preserve `GovernanceBridge` and `governance_injection`, but remove SDD-specific adapter names and use `external:providence` only at the Providence boundary.

**Step 3:** Update tests and command snapshots for rejection of retired SDD inputs.

### Task 5: Remove legacy catalog/template/runtime surfaces

**Files:**
- Modify: `internal/embed/defaults/plugins/catalog.yaml`
- Modify: `internal/embed/defaults/templates/known-providers.yaml`
- Delete: `internal/embed/defaults/templates/epic-sdd.yaml`
- Modify: `internal/embed/defaults/skill.yaml`
- Modify: affected contracts, narrative, generated docs, fixtures, and retirement tests

**Step 1:** Remove SDD-specific catalog entries, aliases, defaults, and dual-read references while preserving generic governance injection.

**Step 2:** Rename arbitrary test fixtures to neutral providers and add retirement pins forbidding `--sdd`, `sdd_injection`, and implicit `.sdd` probing outside historical documentation.

**Step 3:** Regenerate embedded/runtime mirrors from canonical `internal/embed/defaults/` sources.

### Task 6: Validate and close the implementation

**Files:**
- Modify: refined task checklist only after evidence is collected

**Step 1:** Run focused governance, CLI, bridge, telemetry, and generated-parity tests with isolated `GOCACHE`.

**Step 2:** Run `make embed-skills-check`, `make docs-links-gate`, `make convergence-check`, coverage, lint, and the repository test gates.

**Step 3:** Mark only evidence-backed implementation tasks complete; report any environment-only failures separately.
