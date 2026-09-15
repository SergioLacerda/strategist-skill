# ADR-0035 — Embedded Weapon Roster and Fail-Closed Binding Policy

**Status:** Accepted
**Date:** 2026-09-14
**Context:** `20260913-wizard-hardcoded-fallback-maps-cleanup`

## Context

`20260913-embedded-skill-directory-catalog`'s Task 7.2 asked to delete the
Wizard's hardcoded default/fallback maps (`installableDefaultProviders`,
`knownProviderRisk` in `internal/install/wizard.go`;
`defaultDiscoveryProvider`, `defaultRefinementProvider` in
`internal/install/paths.go`) so the Wizard's option lists would come only
from the generated `plugins/catalog.yaml`. That deletion was never done: the
hardcoded maps are the only source of provider defaults/risk data for
`minimalExtractor{}`-based tests (10+ existing tests) and for any real
environment where the catalog fails to load.

`20260913-wizard-hardcoded-fallback-maps-cleanup` was opened to resolve this
residual. Refinement established that the hardcoded values are not legacy
cruft: they are the concrete embodiment of this project's permanently
embedded weapons. Separately, ADR-0028 already governs mission-time slot
fallback (`provider_resolution_policy`: `block`/`ask`/`native`, with `ask` as
the recommended default) but that policy has a real gap — the Wizard's own
install-time fallback (`loadKnownProviders`/`resolveInstallableDefaultProviders`
in `internal/install/plugin_catalog.go`) substitutes the hardcoded maps "on
any error" with no notification to the operator at all, which is a silent
fallback ADR-0028's `ask`-first posture was meant to prevent elsewhere in the
system.

## Decision

1. **Embedded weapon roster is fixed at two pairings; a third is deferred.**
   `brainstorming↔ranger` (discovery) and `openspec-propose↔archivist`
   (refinement) are the project's permanent, always-embedded weapon/role
   pairings — superseding the earlier `openspec-explore↔archivist` framing
   used during discovery/refinement of this mission: `openspec-propose`
   already produces a proposal/design/tasks package mirroring Archivist's
   own output contract, and is the value the user confirmed at the Approval
   Gate. `openspec-explore` remains registered as a secondary, opt-in
   discovery option, unchanged. The execution slot's embedded weapon
   (paired with `sniper`) is explicitly deferred — not decided by this ADR.

2. **There is no fallback substitution.** This supersedes ADR-0028's
   `ask`-first posture for the active role/weapon path. The Wizard refuses to
   activate a configuration unless discovery and refinement each have one
   compatible external weapon, and `strategist check` applies the same
   invariant at runtime: missing, ambiguous, stale, or incompatible bindings
   are fatal. Native roles are not substituted for weapons.

3. **`strategist check` must positively verify embedded role↔weapon
   linkage.** `strategist check` must add an always-run verification,
   independent of the current
   `active.yaml` configuration, that each permanent pairing from Decision 1
   is structurally intact (the embedded skill's `canonical_role` field
   matches `roles/default.yaml`'s slot map, and the target role file exists
   and validates), reporting pass/fail per pairing.

4. The named default-skill values remain catalog/build metadata, but they are
   not runtime substitutes. The refinement default is
   `openspec-propose`; its absence or incompatibility is a fatal Wizard/check
   error, never a reason to select `archivist`.

## Consequences

### Positive

- The Wizard and `strategist check` fail closed before mission execution when
  a required role/weapon binding is invalid.
- `strategist check` becomes a positive source of truth for whether the two
  permanent embedded pairings are intact, instead of only surfacing a
  fallback candidate reactively.
- `openspec-propose` as the refinement default is a better semantic fit than
  `openspec-explore`: its own output shape (proposal/design/tasks) already
  mirrors Archivist's contract.

### Negative

- `openspec-propose` is not yet a fully onboarded embedded skill (missing
  `canonical_role`/`installable` in `plugins/catalog.yaml`, and no
  `.strategist/skills/openspec-propose/skill.yaml` manifest exists yet) —
  onboarding it is now a prerequisite before it can actually be wired as the
  configured default, adding implementation scope beyond a simple constant
  rename.
- Historical fallback-policy structures may remain for compatibility, but are
  not consulted by Wizard activation or `strategist check`.

## Rejected Alternatives

- **Literal deletion (original Task 7.2 wording):** would require replacing
  `minimalExtractor{}` across 10+ tests with a real minimal catalog fixture
  and leaves undefined what a catalog-less real install should do — rejected
  as materially higher cost/risk than documenting and hardening the existing
  fallback.
- **Leave the Wizard's fallback silent:** consistent with the pre-existing
  code comment ("on any error"), but directly contradicts the user's stated
  "pause and notify, never simulate" principle and the spirit of ADR-0028.

## Validation Requirements

- `strategist check` reports pass/fail for both embedded pairings
  independent of `active.yaml`'s current slot configuration.
- A test asserts the Wizard's install-time fallback notice fires when the
  catalog fails to load, and does not fire when it loads successfully.
- All existing `minimalExtractor{}`-dependent tests continue to pass.
- `openspec-propose` onboarding (catalog `canonical_role`/`installable`
  fields, `.strategist/skills/openspec-propose/skill.yaml`) lands before any
  code wires it as the configured refinement default.

## Scope Boundary

This ADR records the architectural decision only. It does not authorize
runtime, configuration, source-code, or test changes — see
`.analysis/refined/20260913-wizard-hardcoded-fallback-maps-cleanup/tasks.md`
for the implementation handoff, which remains outside Strategist's execution
scope (code/config mutation is not Sniper-executable).
