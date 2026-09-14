# ADR-0035 — Embedded Weapon Roster and Fallback Notification Policy

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
   refinement option, unchanged. The execution slot's embedded weapon
   (paired with `sniper`) is explicitly deferred — not decided by this ADR.

2. **Fallback substitution must always be visible, never silent — and, at
   install time, is a hard block, not a warning.** This extends ADR-0028's
   `ask`-first posture, previously scoped to mission-time slot resolution, to
   the Wizard's install-time fallback path, but resolves the two paths to
   different strengths, confirmed as an intentional dual hard-block:
   - **`strategist install`** (Wizard, install-time): when
     `plugins/catalog.yaml` fails to load, the Wizard refuses to prompt at
     all — a hard stop, not a continue-with-warning. It no longer falls back
     to the hardcoded `installableDefaultProviders`/`knownProviderRisk` maps
     silently or otherwise; those maps are retained only for
     `minimalExtractor{}`-based tests and as the post-catalog manifest-writing
     fallback described in Decision 4.
   - **`strategist check`** (runtime/validation-time): fails the command
     (`check=failed`) when an embedded weapon binding — the `WEAPON LINKS`
     section — is broken. This side was already correctly hard-blocking
     before this revision; only this ADR's text was stale.

   These are independent, complementary gates, not a contradiction: both
   layers hard-block on their own respective failure, rather than one of them
   degrading to a visible-but-permissive warning.

3. **`strategist check` must positively verify embedded role↔weapon
   linkage.** `internal/check/check_slots.go#resolveNativeFallback` only
   discovers a compatible native-role fallback lazily, when a slot happens
   to already be configured as an external skill provider. `strategist
   check` must add an always-run verification, independent of the current
   `active.yaml` configuration, that each permanent pairing from Decision 1
   is structurally intact (the embedded skill's `canonical_role` field
   matches `roles/default.yaml`'s slot map, and the target role file exists
   and validates), reporting pass/fail per pairing.

4. The hardcoded default constants (`installableDefaultProviders`,
   `knownProviderRisk`, `defaultDiscoveryProvider`, and the refinement
   default) are retained, not deleted — this supersedes Task 7.2's literal
   "delete" wording. The refinement default becomes a named default-skill
   variable per slot (not an inline literal), whose refinement value is
   `openspec-propose`.

## Consequences

### Positive

- The Wizard keeps working in every environment `minimalExtractor{}`-based
  tests and catalog-less installs rely on.
- No fallback anywhere in the system — Wizard install-time or mission-time
  slot resolution — can substitute a value without the operator/user seeing
  it happen.
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
- Two independent fallback-notification code paths (Wizard install-time,
  ADR-0028 mission-time) now both need to satisfy "always visible," instead
  of one.

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
