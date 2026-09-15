# ADR-0037 — Wizard Role Binding Persistence (Discovery + Refinement)

**Status:** Accepted
**Date:** 2026-09-14
**Context:** `20260913-wizard-plugin-lifecycle-persistence-gap`

## Context

`strategist install` (wizard mode) already computes a resolved role↔weapon
binding twice over during a run — once through the legacy
`planPluginOnboarding` → `domain.PluginLock` path, once through the newer,
already-merged `ApplyRoleProviderMigration` (built for
`strategist-papeis-personagens-skills-nativas`'s Role/Provider convergence
work) — and both times discards the result. `applyWizardConfig`, the
function `strategist install` actually runs, only writes `active.yaml` plus
provider/knowledge-index manifests; nothing durable records that a binding
was ever computed or validated. `internal/plugins/lifecycle.Store` (staged
activation, probing, last-known-good rollback) is fully implemented and unit
tested but has zero production callers.

This was invisible at the CLI/UX level because the shallow legacy mechanism
(`active.yaml`'s static `slots:` entries) is what actually governs runtime
behavior — `strategist check`'s SLOTS/READINESS output reads `active.yaml`
directly, never a lock/binding record. The gap surfaced concretely while
inspecting a real `strategist check` run's `WEAPON LINKS` line
(`openspec-explore→ranger ok`): that "ok" certifies only a static manifest
check (`canonical_role` fields agree, per ADR-0035 DEC-003) — it does not
mean the Wizard ever recorded a real, queryable "this installation bound
weapon X to role Y" decision anywhere on disk.

Two prior missions had already surfaced adjacent slices of this gap without
resolving it:
`.analysis/pending/skills_plugaveis/20260913-role-provider-convergence-evaluation/`
found the Role/Provider persistence half of this work unimplemented
(`partially_implemented` verdict), and
`.analysis/refined/20260913-wizard-plugin-lifecycle-persistence-gap/analysis.md`
documented the general lifecycle-persistence gap and left two uncertainties
open: whether persistence should exist at all (UNC-01) and what scope it
should cover (UNC-03). The user resolved both directly.

## Decision

- **DEC-001.** The Wizard must persist a real, on-disk record of the
  resolved discovery (Ranger) and refinement (Archivist) weapon bindings —
not just a validated `active.yaml`. The role remains authoritative during
mission-time invocation, while the Wizard's binding step
  is real product intent, and its output must be written somewhere durable,
  using the already-implemented `lifecycle.Store`/`ApplyRoleProviderMigration`
  primitives instead of computing and discarding them.
- **DEC-002.** Scope is the Role/Provider-scoped slice, applied symmetrically
  to both slots that carry a native-role fallback today — discovery
  (Ranger) and refinement (Archivist) — not the full legacy
  `applyPluginOnboardingPlan` path, and not the execution slot (`sniper`,
  which has no subtype customization and no native-role-fallback
  ambiguity).
- **DEC-003.** A bound weapon is immutable for the lifetime of a single
  mission, for both Ranger and Archivist — a design principle to preserve,
  not a defect to fix. Once persisted, no in-mission code path may
  re-resolve or substitute a different weapon for the same slot mid-mission.

Persistence makes the Wizard's role-to-weapon choice real and observable.
The bound weapon may run within the role boundary, but it cannot replace the
role's normalization, checkpoint, lock, state, or control-log responsibilities.
DEC-003 forbids *switching* the bound weapon mid-mission.

**Deferred (NC-01):** whether `strategist check`'s READINESS output should
start reflecting real lifecycle state (`staged`/`probed`/`active`) once
persistence lands, versus keeping today's `not_ready` placeholders. This
needs its own explicitly-decided follow-up mission, not a mechanical wiring
fix bundled into this one.

**Resolved (NC-02).** The persisted state lives in the dedicated
`.strategist/plugins.lock` YAML file, distinct from the user-hand-edited
`active.yaml` and from the unrelated `.config.lock` tamper-detection seal.
Its envelope contains the resolved `PluginLock` graph plus lifecycle
inventory and discovery/refinement `SlotBinding` records. The writer stamps
`strategist-plugin-lock-file/v1` and writes atomically; a missing file is a
valid fresh-install state. The file is written only after `active.yaml` has
been written successfully, so a failed configuration write cannot leave a
binding artifact without its corresponding active configuration.

## Consequences

### Positive

- The Wizard's weapon-selection step stops being purely cosmetic for
  discovery and refinement — its outcome becomes a durable, inspectable
  record instead of computed-then-dropped state.
- Reuses fully-implemented, already-tested infrastructure
  (`lifecycle.Store`, `ApplyRoleProviderMigration`) instead of introducing a
  new persistence mechanism.
- Keeps the fixed Ranger normalization/checkpoint boundary explicit while
  allowing the persisted weapon to provide flexible discovery input.

### Negative

- `strategist check`'s READINESS output stays disconnected from the new
  persisted state until a separate, future decision (NC-01) addresses it —
  users gain a persisted binding without an immediately visible runtime
  signal for it.
- Adds a second on-disk artifact (the lock file recommended by NC-02)
  alongside `active.yaml`, with its own read/write and drift-detection
  surface to maintain.

## Rejected Alternatives

- **Persist the binding as an additional block inside `active.yaml`
  itself.** Rejected in favor of a dedicated lock file (NC-02's
  recommendation): `active.yaml` is user-hand-edited configuration, while
  the resolved binding is machine-computed, resolved state — mixing the two
  would blur that distinction the way `.config.lock`'s existing
  tamper-seal/config split already avoids.
- **Extend scope to the full legacy `applyPluginOnboardingPlan` path in the
  same mission.** Rejected — the user scoped this to the Role/Provider
  slice only (DEC-002); the legacy path's identical "never invoked" defect
  is a separate, pre-existing gap that can be addressed independently.
- **Bundle the READINESS-output wiring (UNC-02/NC-01) into this decision.**
  Rejected — `check_readiness.go` was deliberately left untouched during
  the prior Role/Provider residual work specifically to avoid an unreviewed
  runtime-behavior change; this ADR preserves that caution and defers it.

## References

- `.analysis/refined/20260913-wizard-plugin-lifecycle-persistence-gap/` — mission that produced this decision
- `.analysis/pending/skills_plugaveis/20260913-role-provider-convergence-evaluation/` — evaluation that found the underlying gap
- `docs/adr/0028-native-role-resilient-baseline.md` — native-role resilient-baseline policy this ADR builds on
- `docs/adr/0035-embedded-weapon-fallback-policy.md` — related fallback-policy context (DEC-003, static WEAPON LINKS check)
