## Why

A prior refactor (currently staged, uncommitted) deleted `CheckRoleCompatibility`
and `ResolveProviderBinding` from `internal/domain/role_provider_compatibility.go`
because they had zero production callers. It left two smaller pieces of dead
plumbing behind that a codebase-wide legacy-behavior audit
(`.analysis/refined/20260915-legacy-behavior-audit/analysis.md`) surfaced as
safe, zero-risk removal candidates: an inert handoff-schema lookup in
`internal/check/check_role_compatibility.go`, and an unreferenced constant in
`internal/domain/plugin_types.go`. Removing them now, in the same area of the
codebase that is already mid-refactor, avoids a second pass later and keeps
`CHANGELOG.md` accurate about what left the codebase in this cycle.

Revision (2026-09-15, requested at the Approval Gate): the user asked to also
incorporate SQ-2 from the discovery brief — ADR-0035's own "Context" section
already flags `resolveInstallableDefaultProviders`
(`internal/install/plugin_catalog.go:112-122`) as silently substituting a
hardcoded provider map "on any error" from `loadPluginCatalog`, which
contradicts ADR-0028's ask-first posture and the "no fallback substitution"
principle ADR-0035 Decision 2 already adopted for the sibling `runWizard`
path. Verifying this against the current code: both of `resolveInstallableDefaultProviders`'s
two known callers (`runWizard` via `installer_config.go:81`→`97`, and
`activateSilentRoleProviderBindings` at `installer_silent_role_bindings.go:24-27`)
already hard-block on a `loadPluginCatalog` failure earlier in the same call
stack, so the silent-fallback branch inside `resolveInstallableDefaultProviders`
is unreachable via either path today — the same "unreachable in practice"
situation `wizard_fallback_providers.go`'s doc comment already documents for
its sibling `loadKnownProviders`. This revision hardens it anyway (defense in
depth against a future caller that lacks that upstream guard) and documents
why, rather than leaving an undocumented latent violation of the same
principle ADR-0035 just adopted elsewhere.

## What Changes

- **Removed**: `loadSupportedHandoffSchemas()` and the handoff-schema plumbing
  in `internal/check/check_role_compatibility.go` (lines ~70-74, 85, 98-127) —
  it computes `providerContract.SupportedHandoffSchemas`, but
  `CheckRoleAffinity` (`internal/domain/role_provider_compatibility.go:8-27`)
  never reads that field. No behavior change: the compatibility check result is
  identical with or without this plumbing.
- **Removed**: `SlotExtensionLabel` constant in `internal/domain/plugin_types.go`
  (lines ~40-42) — zero callers anywhere in the tree (its sibling
  `SlotExtensionKindLabel` is used and stays).
- **Doc fix**: correct the stale "Negative consequences" paragraph in
  `docs/adr/0038-docs-information-architecture-and-mandate-layer.md:100-107`,
  which still claims three files reference old doc paths; all three already
  reference the current paths.
- **Changelog**: add a `CHANGELOG.md` `[Unreleased] ### Removed` entry covering
  both this change's deletions and the already-staged
  `CheckRoleCompatibility`/`ResolveProviderBinding` removal.
- **Hardened**: `resolveInstallableDefaultProviders`
  (`internal/install/plugin_catalog.go:112-122`) propagates a
  `loadPluginCatalog` error instead of silently substituting
  `installableDefaultProviders`, matching `runWizard`'s existing hard-block
  (`internal/install/wizard.go:88-91`) and ADR-0035 Decision 2. The doc
  comment gains the same "unreachable in practice via current callers" note
  `loadKnownProviders` already carries, so the reason for the change is
  legible without re-deriving the call-graph analysis.

None of these are **BREAKING** — all five items are dead code, an
unreferenced constant, doc-only corrections, or a defense-in-depth hardening
with no observable behavior change on any currently reachable call path (both
known callers already guard against the error case before reaching this
function).

## Capabilities

No spec-level behavior changes. This is dead-code removal and a documentation
correction; the compatibility check's observable behavior (`CheckRoleAffinity`'s
pass/fail outcome) is unchanged before and after. `skip_specs: true` is set in
this change's `.openspec.yaml` accordingly.

### New Capabilities

(none)

### Modified Capabilities

(none)

## Impact

- `internal/check/check_role_compatibility.go` — simplified, same test coverage
  applies (`internal/check` package tests), no test currently depends on the
  removed plumbing producing a different `CheckRoleAffinity` outcome.
- `internal/domain/plugin_types.go` — one constant removed, zero callers, zero
  test coverage of it being read.
- `docs/adr/0038-...md` — text-only correction, no code impact.
- `CHANGELOG.md` — additive entry, no code impact.
- `internal/install/plugin_catalog.go` — `resolveInstallableDefaultProviders`
  error handling changed from silent substitution to propagation; both known
  callers (`installer_config.go`, `installer_silent_role_bindings.go`) already
  guard the error case earlier in the call stack, so no currently reachable
  test or install path observes different behavior.

### Explicitly out of scope (see audit for rationale, do not re-propose without new evidence)

- `internal/install/plugin_catalog_legacy.go` / `LegacyManifestPath` — live
  write path for installed provider manifests, matches ADR-0030's in-progress
  migration plan (deprecation period not yet closed).
- `internal/domain/jewel_grade.go`'s `jewelStatusLegacyActive` guard —
  permanent migration safety net for externally-installed `jewels.yaml` files,
  backed by ADR-0012 and a runbook.
- `internal/install/wizard.go` / `wizard_fallback_providers.go` hardcoded
  provider maps themselves (`installableDefaultProviders`, `knownProviderRisk`)
  — ADR-0035 already investigated and explicitly rejected removing these; this
  change only hardens error propagation around one of their consumers, it does
  not touch the maps.
- `loadKnownProviders` (`wizard_fallback_providers.go:57-72`) — already
  documents its own "unreachable in practice" status; left unchanged since it
  is not the item the user asked to incorporate.
- `internal/domain/plugin_types.go:40-41` (`SlotExtensionLabel`'s own doc
  comment) / `docs/architecture/governance-hierarchy.md`'s Go-vs-product
  vocabulary split — a deliberate, permanent naming split, not legacy leftover.
- `tests/spec/mission_status_vocabulary_test.go`,
  `tests/spec/legacy_terminology_test.go` — anti-regression guards that enforce
  the *absence* of legacy tokens; removing them would remove the safety net,
  not a legacy behavior.

This change does not touch source code as part of the Strategist mission
itself — Strategist (the orchestrator that produced this proposal) only
analyzes and refines. Materialization of these edits requires a separately
authorized execution step outside Strategist's default `sniper` contract,
which does not permit source-code mutation (see
`tasks.md` — all tasks here are classified `implementation_handoff`, not
Sniper-executable `documentation_target` items).
