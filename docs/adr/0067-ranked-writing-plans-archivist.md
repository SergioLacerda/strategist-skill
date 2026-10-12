# ADR-0067 — Ranked embedded Archivist Weapon: `writing-plans`

**Status:** Accepted
**Date:** 2026-10-10
**Mission:** `20261010-writing-plans-ranked-promotion-r2` (refined package `.analysis/refined/20261010-writing-plans-ranked-promotion-r2/`)
**Related:** [ADR-0030](0030-world-class-strategist-plugin-architecture.md), [ADR-0035](0035-embedded-weapon-fallback-policy.md), [ADR-0043](0043-ranked-pipeline-pilot-implementation-decisions.md), [ADR-0045](0045-ranked-testsuitedigest-shared-pin.md), [ADR-0061](0061-weapon-sidecar-generation-and-versioned-catalog.md)

## Context

The Archivist supports both the default `openspec-propose` path and the ranked
embedded `writing-plans` path. The latter is a prompt-only skill carried by the
embedded prompt bridge: its sidecar declares `runtime.kind: embedded` and
`execution_mode: prompt_bridge`, and the generated runtime mirrors that source
contract. Completion enters the shared normalization and publication tail, so
the same confidence, package, gate and provenance rules apply to either
Archivist provider.

## Decision

- **DEC-001. Third Ranked pairing.** Certify `writing-plans@1.0.0` for the `archivist` role by adding
  it to the pin map. Certification stays a deliberate, per-version addition.
- **DEC-002. Embedded prompt bridge.** The sidecar declares `runtime.kind: embedded` with
  `execution_mode: prompt_bridge`, the shape `brainstorming` already uses. No new runtime kind is
  introduced.
- **DEC-003. Explicit completion path.** `mission invoke` emits an Archivist execution contract and
  output contract for `archivist/refinement`; `mission complete` dispatches by role and slot and fails
  closed (`role_invocation_failed`) for an unregistered pair. The registration and the dispatch change
  together.
- **DEC-004. One publishing tail.** The validate, record-confidence, publish-or-promote and
  publication-record sequence of `NormalizeOpenSpec` is shared with the embedded publisher; provenance
  metadata becomes a parameter. Amendment handling stays OpenSpec-only.
- **DEC-005. Single completion object.** The result carries three documents (`proposal.md`,
  `design.md`, `tasks.md`) and the typed `handoff_policy_facts` in its frontmatter, because
  `mission complete` takes one JSON object on stdin and no further flags. Destinations such as
  `docs/plans/`, commit steps and unclassified tasks are rejected before anything is written.
- **DEC-006. Coexistence.** `openspec-propose` remains the default Archivist; `writing-plans` is
  selectable alongside it. Ranger, Sniper and `openspec_root` behavior do not change.
- **DEC-007. Wizard persistence.** A Ranked selection is persisted as a `ranked` binding that matches
  the compiled registry identity.

## Evidence caveat

[ADR-0045](0045-ranked-testsuitedigest-shared-pin.md) accepted a shared, fixed `TestSuiteDigest` pin
whose tests exercise the Ranger's outgoing handoff contract, not the Archivist's. This pairing uses the
same pin, so it adds no role-specific conformance evidence for the Archivist. The certification
digest for `writing-plans` therefore records identity, role, handoff schema and runtime shape; it does
not prove behavior. The embedded completion, publication and containment tests are the behavioral
evidence for this pairing.

## Consequences

- A prompt-only Weapon can fill the Archivist slot through the same invoke/complete boundary as the
  Ranger, at the cost of one more pin and one more completion path to keep correct.
- Two Archivist publishers (OpenSpec and embedded) share a tail, so idempotence and conflict rules stay
  in one place; the existing `normalize-openspec` tests must pass unchanged.
- The catalog, generated runtime and docs gates are regenerated through `prepare-embedded` and
  `compile`; installation still changes only when `writing-plans` is selected.

## Rejected Alternatives

- **A private runtime kind mirroring `openspec_root`.** `writing-plans` has no process to bundle or
  health-check; it would add a runtime kind and bootstrap machinery with nothing to run.
- **Stay Custom/host and repair the wizard evidence only.** A Custom host Weapon is rejected by
  `mission invoke` (it requires a Ranked embedded binding) and would not be Ranked. Repairing the
  wizard evidence remains a separate demand.
- **Make `writing-plans` the default Archivist.** Not requested; the user chose coexistence.
- **Use `openspec-propose` only.** No change; does not meet the request.

## Open Questions

- Pin key spelling: `writing-plans@1.0.0` (catalog version) versus a shortened form like
  `openspec-propose@1.0`. Does not change the approach.
- Related, not decided here: `strategist check` should not report ready for a Custom binding that
  `mission invoke` rejects, and non-Ranked wizard selections are persisted as `custom` with a bare id.
