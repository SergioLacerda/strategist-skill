# ADR-0067 — Ranked embedded Archivist Weapon: `writing-plans`

**Status:** Proposed
**Date:** 2026-10-10
**Mission:** `20261010-writing-plans-ranked-promotion-r2` (refined package `.analysis/refined/20261010-writing-plans-ranked-promotion-r2/`)
**Related:** [ADR-0030](0030-world-class-strategist-plugin-architecture.md), [ADR-0035](0035-embedded-weapon-fallback-policy.md), [ADR-0043](0043-ranked-pipeline-pilot-implementation-decisions.md), [ADR-0045](0045-ranked-testsuitedigest-shared-pin.md), [ADR-0061](0061-weapon-sidecar-generation-and-versioned-catalog.md)

## Context

The Archivist can run as a Ranked Weapon only through `openspec-propose`, whose private
`openspec_root` runtime ships an executable and is published with
`strategist mission normalize-openspec`. `writing-plans` is a prompt-only skill whose catalog sidecar
declares `runtime.kind: host` (`strategist-host-skill/v1`).

Three facts keep it from being Ranked today:

- A Ranked binding is rejected when its runtime is `host` or `executable`, an `embedded` runtime
  cannot declare a HostAPI, and `prompt_bridge` execution requires `embedded`
  (`internal/domain/compiled_registry_validation_ranked.go`, `internal/domain/ranked_runtime.go`).
- Certification is an explicit pin map (`rankedCertificationPairs` in
  `internal/install/embedded_weapon_certification.go`); only `brainstorming`, `openspec-propose` and
  `sniper` are pinned.
- `strategist mission complete` registers completion only for `ranger/discovery` and
  `sniper/execution` (`internal/application/invocation_validation.go`), and
  `completeMissionInvocation` sends every non-Sniper completion to the discovery normalizer.

Selecting `writing-plans` through the wizard consequently persisted a `custom` binding with an
unversioned instance id and `source`, `binding` and `certification` digests inherited from the
previous Ranked `openspec-propose` binding. `strategist check` reported it ready (its Custom branch
runs only when `providers/<instance>/package.yaml` exists) while `strategist mission invoke` rejected it
with `custom_binding_invalid`.

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
not prove behavior. The contract tests required by the plan (certification, completion dispatch,
publication, containment) are the evidence this pairing relies on, and they must be written first.

## Consequences

- A prompt-only Weapon can fill the Archivist slot through the same invoke/complete boundary as the
  Ranger, at the cost of one more pin and one more completion path to keep correct.
- Two Archivist publishers (OpenSpec and embedded) share a tail, so idempotence and conflict rules stay
  in one place; the existing `normalize-openspec` tests must pass unchanged.
- The catalog, generated runtime and docs gates are regenerated through `prepare-embedded` and
  `compile`; no installation changes until `writing-plans` is selected.
- Implementation is outside Strategist and is tracked as the `implementation_handoff` tasks of the
  refined package (2.1 to 6.2).

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
- Whether ADR-0045's caveat is enough for a second Archivist pairing or needs its own evidence
  statement before this ADR moves to Accepted.
- Related, not decided here: `strategist check` should not report ready for a Custom binding that
  `mission invoke` rejects, and non-Ranked wizard selections are persisted as `custom` with a bare id.
