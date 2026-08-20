# ADR-0027 — Refinement (Archivist) as a Native Role, Mission-Scoped Precedent

**Status:** Accepted
**Date:** 2026-08-19
**Context:** `20260819-portable-light-client-eval`

---

## Context

The user asked for an analysis evaluating a migration to, or a portable/light client
analogous to the Markdown-only Hub Centralizado skill — a version of Strategist usable
without the Go CLI. The comparison found Strategist
already CLI-free at mission time for 2 of its 3 pipeline roles: Ranger (discovery) and
Sniper (execution) are `native_role` — the invoking agent embodies them directly by
reading `roles/<role>.yaml` + `internal_skills/<role>/SKILL.md`, with `00-routing.md`
§ "Discovery Weapon Resolution by Subtype" explicitly overriding `active.slots.discovery`
in favor of native Ranger for every subtype.

Refinement had no equivalent override. Its configured weapon, `openspec-explore`, exists
only as a capability-mirror `skill.yaml` (`.strategist/skills/openspec-explore/skill.yaml`)
with no `SKILL.md` and no invocation mechanism in this environment. This had already
blocked mission `20260819-claude-standalone` with `error=role_invocation_failed slot=refinement
provider=openspec-explore`, and reproduced identically at the start of this mission's own
refinement phase.

`.strategist/skill.yaml#orchestration.slot_hierarchy.refinement` already calls Archivist
"the internal refinement persona" and openspec-explore "its default weapon, not a
replacement for the persona" — language parallel to how discovery's native-role override
is framed — and `.strategist/contracts/narrative/04-refinement.md` never once instructs
invoking an external tool; it describes Archivist reading the Ranger artifact and writing
the four refined files directly, structurally identical to Ranger's own contract.

## Decision

For mission `20260819-portable-light-client-eval`, and only after explicit user
confirmation at the point of the block, refinement proceeded with Archivist treated as a
native role: the invoking agent embodied Archivist directly (reading
`roles/archivist.yaml` + `internal_skills/archivist/SKILL.md`), the same mechanism already
used for Ranger and Sniper, instead of hard-stopping on `openspec-explore`'s absence.

This ADR records that decision as precedent. It does **not** itself change
`contracts/narrative/00-routing.md`, `roles/archivist.yaml`, `skill.yaml`, or the
`internal/embed/defaults/` mirror — formalizing a permanent "Refinement Weapon
Resolution" section (mirroring the existing "Discovery Weapon Resolution by Subtype")
is recommended as explicit follow-up work, not adopted here. Until that follow-up lands,
each future occurrence of this block must still surface the same escalation choice to the
user (see the runbook addendum below) rather than assuming this ADR pre-authorizes silent
native-role substitution repo-wide.

## Alternatives Considered

- **Reconfigure `active.slots.refinement` to a different, installed provider** — rejected:
  no alternative refinement provider was available in this environment either, so this
  would only defer the same failure to the next provider choice.
- **Stop at analysis-only, do not refine** — rejected: the user explicitly asked for a
  design evaluation with actionable recommendations, and discovery evidence already
  supported one.

## Consequences

- Unblocks this mission shape without requiring `openspec-explore` (or an equivalent) to
  be installed.
- Leaves a documented asymmetry: discovery's native-role resolution is a permanent,
  formally documented override (`00-routing.md`); refinement's is, as of this ADR, only a
  recorded mission-scoped precedent requiring the same explicit user escalation each time
  it recurs, until the follow-up contract change is separately authored and merged.
- The follow-up items remain open: formalize the routing override and build an additive
  "light mode" `.strategist/` template variant. Both are implementation work outside
  Sniper's documentation-only write scope and require a separately authorized coding task.

## References

- `docs/runbooks/role-invocation-failed.md` § Refinement-Specific Escalation (companion runbook addendum)
- `.strategist/contracts/narrative/00-routing.md` § Discovery Weapon Resolution by Subtype (the pattern this decision mirrors)

---

## Addendum (2026-08-19) — Corrected Root Cause

Mission `20260819-drift-native-refinement-diagnostic` re-evaluated this workspace's refinement-slot drift after `.strategist/active.yaml`'s `slots.refinement` was changed from `openspec-explore` to `archivist`.

This addendum does not change the Decision or Consequences above — it corrects and narrows the *root-cause framing* they were based on:

- `internal/check/check_slots.go#resolveSlotProvider` already, unconditionally, resolves any slot's configured provider to a native role (`roles/<provider>.yaml`) whenever no matching `skills/<provider>/skill.yaml` exists. This is generic, pre-existing behavior — it required **zero Go code changes** to support `slots.refinement: archivist`. `strategist check` in this workspace now reports `refinement archivist kind=native_role`, `STATUS: ok`, confirming the fix.
- The actual cause of the original block was narrower than "refinement has no native-role path": `internal/install/paths.go`'s `defaultRefinementProvider = "openspec-explore"` is what every `strategist install --wizard` run suggests by default. `openspec-explore` passes `strategist check`'s static validation (a valid `skill.yaml` with matching `risk_score`) but fails real invocation, because `check` validates manifest presence/schema only — it cannot detect whether an agent's current runtime actually exposes the provider as an invocable skill.
- Practical effect: **T2** (formalizing a "Refinement Weapon Resolution" section in `00-routing.md`) remains worth doing for documentation clarity, but is no longer a *functional* blocker — the config-only fix (`active.yaml` edit) already works today, in any installation, without it. The higher-leverage follow-up, if pursued, is changing or clarifying the installer's suggested default in `internal/install/paths.go`/`wizard.go` (new item, not part of the original T2/T3) so future installs don't reproduce this drift.
- A `hash_mismatch` integrity warning appeared as a side effect of the `active.yaml` edit being applied outside `strategist install` — non-blocking (`STATUS: ok`), tracked as an open recommendation, not executed by Strategist (CLI action, outside documentation-only scope).

## Addendum (2026-08-19, second) — Restating the Bug: the Manual Workaround Is the Defect

**Source:** user correction following the diagnostic addendum above, same day.

The first addendum still framed the fix as "the wizard's suggested default should be better" — a nice-to-have. That understates it. The actual bug is:

> **Strategist required a manual, out-of-band edit to `active.yaml` — bypassing `strategist install`/`compile` — to unblock refinement once the configured provider (`openspec-explore`) turned out not to be invocable.** That hand-edit is precisely what produced the `hash_mismatch` integrity warning this workspace now carries. The warning is not an unrelated side finding; it is *evidence of the bug* — a trace left by the workaround required to route around a defect that should not require any manual runtime adjustment at all.

Reprioritization:

- **T3** (`internal/install/paths.go` / `wizard.go` — installer default) is now the *primary* fix, not a secondary follow-up: no fresh install should ever need a hand-edited `active.yaml` to reach a working refinement slot.
- Whether a *second*, complementary fix belongs at the preflight/runtime layer (e.g., `strategist check` proactively detecting a `skill_provider`-resolved slot whose provider is not actually present in the current agent's skill listing, and surfacing a suggested `strategist install`-driven correction, rather than a human discovering it only after a mission blocks mid-mission) is an open design question, explicitly **not decided here** — it risks reintroducing the "silently substitute a provider" drift this repository's own contracts (`agent-protocol.md` §1b, §2) deliberately guard against. Any such mechanism must still surface the choice to the user rather than auto-applying it.
- Until the installer-default fix lands, the correct operational stance is: the current hand-edited `active.yaml` is a **known, load-bearing workaround**, not a resolved state — `strategist install` should not be run casually to "clean up" the `hash_mismatch` warning without first confirming it will not silently revert `slots.refinement` back to `openspec-explore`, which would reintroduce the original block.
