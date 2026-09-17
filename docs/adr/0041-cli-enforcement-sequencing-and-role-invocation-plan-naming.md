# ADR-0041 — CLI Enforcement Sequencing and RoleInvocationPlan Naming

**Status:** Proposed
**Date:** 2026-09-15
**Context:** `20260915-cli-enforcement-refactor-refinement`

## Context

A six-document draft (`.analysis/pending/cli-enforcement-refactor/`) proposed migrating
mechanical, deterministic pipeline responsibilities — role↔weapon binding resolution,
preflight checks, phase-transition authority, and per-phase context loading — from
agent-followed prose instructions ("agent_only" enforcement) into Go/CLI-enforced runtime
code ("machine_enforced"), while keeping semantic reasoning (investigation, synthesis,
critique, creativity) with the LLM. This motivation echoes ADR-0028's own originating
incident: a configured provider (`openspec-explore`) that passed static validation but was
not invocable at mission time, resolved only through manual user intervention.

A Strategist discovery pass over the draft found that large parts of its proposed ground are
already decided or already shipped:

- **ADR-0034** already formalizes "Role defines WHAT/authority, Weapon defines
  HOW/specialization" and explicitly removes `provider_class` /`known_provider_class`
  /`specialization_taxonomy` as dead, non-branching fields.
- **ADR-0035** and **ADR-0037** already decide, and ship code for, fail-closed provider
  resolution and mission-lifetime-pinned weapon bindings.
- **`internal/check/check_slots.go`** already implements the two-branch
  exists?/valid?/matches? slot resolver the draft's preflight proposal called for.
- **`internal/rolevalidation/role_weapon.go`**, verified by
  `.analysis/done/20260915-pending-role-weapon-readiness-review/` and
  `.analysis/done/20260914-role-weapon-structure-review/`, already implements and
  test-verifies most of the role/weapon-readiness preflight slice.
- **`internal/domain/state_machine.go`** already implements a partial FSM (gate, execution,
  retry, ADR, Critical-Hit), whose own header comment explicitly defers
  bootstrap/intake/discovery/refinement modeling to a "full-pipeline FSM" follow-up already
  noted in `.analysis/todo/analise-tecnica.md`.

The draft also introduced two names that collide with existing project vocabulary:

- **"Invocation Envelope"** — `internal/plugins/connectors.InvocationEnvelope` already exists
  (`runtime_connector.go:39-46`) as the payload for `RuntimeConnector.Invoke`, a
  host/plugin-runtime boundary type (`SchemaVersion, Instance, Entrypoint, MissionID,
  GateAllowed`) distinct from the Role+Weapon+context+schema composition the draft describes.
- **"Ranked"** — `docs/architecture/strategist-concepts.md` §"Ranked Role" documents a
  `provider_class: rankeado` field as current/live, which directly contradicts ADR-0034's
  removal of that exact field. The draft's "Ranked Binding" (a build-time-certified
  Role+Weapon composition) is a materially different, heavier concept than that stale,
  decorative field.

Separately, `.analysis/pending/skills_plugaveis/02-external-skills-adapters/20260820-world-class-skill-plugin-refinement/`
is an existing, more detailed, still-pending plan covering closely overlapping ground
(digest-pinned lock, `RuntimeConnector` SPI, permission grants, journaled lifecycle) against
the same target packages (`internal/domain`, `internal/plugins`, `internal/install`,
`internal/check`).

## Decision

1. **Sequencing.** Adopt the reconciled remaining work in this order, each increment stated
   as an extension of existing, shipped mechanisms rather than a parallel rewrite:
   1. A Role→Weapon resolver/composer producing a mission-scoped binding descriptor, built on
      ADR-0035/0037's existing fail-closed/pinned-binding rules.
   2. A typed `PreflightResult` object aggregating the existing outputs of
      `internal/check/check_slots.go` and `internal/rolevalidation/role_weapon.go` into one
      versioned envelope.
   3. A phase-transition authority extending `internal/domain/state_machine.go` to cover only
      bootstrap/intake/discovery/refinement — the gap already named in
      `.analysis/todo/analise-tecnica.md` — without altering the FSM's existing
      gate/execution/retry/ADR/Critical-Hit states.
   4. A per-phase context-loading resolver extending the existing declarative
      `contracts/index.yaml` pattern.
2. **Naming.** The new Role+Weapon+context+schema composition type is named
   `RoleInvocationPlan`, not "Invocation Envelope." `internal/plugins/connectors.InvocationEnvelope`
   is unchanged; a doc-comment cross-references both types by name and purpose once
   `RoleInvocationPlan` exists.
3. **Residuals — explicitly not decided by this ADR.** One item surfaced by discovery remains
   an open, tracked residual rather than resolved here:
   - Whether/how to reconcile "Ranked Binding" with the stale `provider_class: rankeado`
     section in `docs/architecture/strategist-concepts.md` and ADR-0034's removal of that
     field.

   This residual is tracked as a separate pending demand
   (`.analysis/pending/20260915-ranked-binding-vs-provider-class-naming-residual.md`) and
   requires its own explicit approval before any related implementation work proceeds. The
   `skills_plugaveis` overlap, originally recorded as a second open residual, is now resolved
   at the sequencing level by the V1/V2 Layering decision below (mission
   `20260915-cli-enforcement-refactor-skills-plugaveis-overlap-residual`); V1's own P1–P9
   delivery plan remains a separate, still-pending, unimplemented plan in its own right.

## V1/V2 Layering (added 2026-09-15, mission `20260915-cli-enforcement-refactor-skills-plugaveis-overlap-residual`)

Discovery for the `skills_plugaveis` overlap residual (above) validated that the two plans are
layered, not competing:

- **V1** (`skills_plugaveis/02-external-skills-adapters/20260820-world-class-skill-plugin-refinement`)
  governs plugin **identity, install, trust, and long-lived slot binding** —
  `PluginPackage`, `AdapterContract`, `InstalledInstance`, `SlotBinding`, `TrustPolicy` — state
  that outlives any single mission. It never models mission-phase sequencing or per-mission
  binding composition; both are explicitly outside its stated non-goals.
- **V2** (this ADR's own scope) governs **mission-time invocation orchestration** —
  `RoleInvocationPlan`, `PreflightResult`, the phase-transition authority, and
  `ContextComposer` — state scoped to one mission's lifetime. It never models plugin
  installation, trust, or permissions.

Two concrete data-consumption obligations follow from this split, recorded now so V2's
eventual implementation does not duplicate V1's authority once V1 ships:

- **`PreflightResult` is an explicitly interim aggregation.** V1's P4 defines a
  10-dimension readiness state vector (descriptor valid, source resolved, trust verified,
  dependencies locked, host API compatible, connector visible, entrypoint probed, permission
  grant complete, connector enforcement coverage, active binding healthy) plus C0–C3
  conformance levels — materially richer than `PreflightResult`'s
  `{status, identity, bindings, warnings, next}` shape, which for now only aggregates
  `check_slots.go`/`role_weapon.go`. Once V1's P4 (`RuntimeConnector` SPI) ships,
  `PreflightResult.bindings`/`.warnings` MUST be re-derived from V1's readiness vector, not
  maintained as a second, parallel readiness computation.
- **`RoleInvocationPlan` reads its pinned binding from V1's `SlotBinding` once V1 ships.**
  Both V1's `SlotBinding` (install-time pin, binding generation, CAS) and V2's
  `RoleInvocationPlan` (mission-time pin) describe the same fact — "this slot currently
  resolves to this weapon at this digest" — at two different horizons. Until V1 ships,
  `RoleInvocationPlan` reads today's flat `.strategist/plugins.lock`; it must not grow its own
  independent digest-pinning mechanism once V1's `SlotBinding`/`plugin.lock` exists, per V1's
  own DEC-101 ("one manifest must not mix publisher metadata, Strategist compatibility,
  workspace installation state, user binding, and trust policy").

This layering decision does not re-sequence or re-scope V1's own P1–P9 delivery plan, and does
not resolve a pre-existing internal inconsistency in V1's own package (`analysis.md`
frontmatter reads `mission_status: documentation_applied` while its own `tasks.md` Sniper
Instructions state the mission has no executable documentation target) — tracked as side
quest SQ-C, not fixed by this amendment.

## Ranked Class Resolution (added 2026-09-15, mission `20260915-ranked-binding-vs-provider-class-naming-residual`)

The "Ranked" naming residual named above (§"Ranked") is resolved. The requester
supplied a new, concrete definition: a **ranked class** is a Role that, at
compile/build time, is already bound to a specific weapon (an embedded skill),
contrasted with the Wizard's existing external-weapon customization flow.

This is reconciled with the draft's "Ranked Binding" concept and the stale
`provider_class: rankeado` field as follows:

- The removed `provider_class: rankeado` field (ADR-0034) is not revived. It
  remains dead vocabulary; `docs/architecture/strategist-concepts.md` §"Ranked
  Role" (renamed "Ranked Class") is corrected to no longer document it as
  current.
- The draft's certification pipeline (manifest/dependency/affinity/
  contract-test/handoff-schema validation, digest calculation) is the **core**
  mechanism. A ranked class is one Role-binding implementation built on that
  core; other future functionality is expected to build on the same core too.
- A large part of the existing catalog/selection structure (including
  `default: true` / `ProviderContract.Default`) is expected to be reused rather
  than replaced. The intended split is deterministic behavior resolved via the
  CLI/build step, with formal contracts and definitions living in the runtime
  (`.strategist/`), keeping the agent-facing surface light.
- The name "Ranked" is kept for the new concept — not replaced with a different
  term.

**Pipeline scope (added 2026-09-16, mission `20260916-ranked-vs-custom-binding-pipelines`):**
Role→Weapon binding is two separate pipelines, not one. **Custom** is today's
flow — a wizard-selected embedded or external weapon, validated/probed, and
persisted to `plugins.lock` — and is exactly what `RoleInvocationPlan`
(task 1.1), `PreflightResult`'s readiness vector (task 2.5), and the trust-
policy/permission-grant readiness wiring (`internal/check/check_trust_and_grants.go`)
govern. **Ranked** ("classe rankeada," resolved above) bypasses that
machinery entirely: its binding is certified at build time, so it needs no
wizard-time validation, no runtime trust verification, and no permission-grant
negotiation. The two pipelines are not layers of the same mechanism — Ranked
does not call into Custom's runtime checks at all. This scoping does not
decide `.analysis/pending/cli_refactor/20260915-cli-enforcement-refactor-refinement/tasks.md`
task 1.1b (SlotBinding's on-disk shape); it explains why that task is a
Custom-pipeline-only question. Task 1.1b is since closed by
`docs/adr/0042-ranked-custom-binding-persistence.md`, which decides that
question: no file separation is needed, because a Ranked binding's runtime
record (if persisted at all) is optional, non-authoritative redundancy, not a
second authority to separate from Custom's.

This amendment records the naming resolution only. It does not design or
authorize the certification pipeline's own implementation, the `plugins.lock`
shape, or the precise mapping of which existing fields are reused verbatim vs.
extended — that remains `.analysis/pending/cli_refactor/v2/02-ranked-role-binding.md`'s
own, separately-approved design scope.

Side quest SQ-001 (the `ranger-weapons` dojo scenario's stale `provider_class`
field assertion, surfaced during this mission's discovery) was fixed as part of
this same mission, at the requester's explicit request.

## Consequences

### Positive
- The remaining CLI-enforcement work is scoped to what is genuinely new, avoiding duplicate
  or contradictory implementation of already-shipped ADR-0034/0035/0037 mechanisms.
- The `RoleInvocationPlan`/`InvocationEnvelope` naming collision is resolved before any code
  is written, avoiding confusion in `internal/plugins/connectors`.
- The `skills_plugaveis` overlap and the `strategist-concepts.md` naming drift are made
  visible and trackable instead of being silently absorbed or silently ignored.

### Negative
- Sequencing decisions here do not by themselves unblock implementation: all four increments
  remain `implementation_handoff` items requiring a separately authorized execution provider.
- The "Ranked" vocabulary conflict is resolved (see "Ranked Class Resolution" above,
  mission `20260915-ranked-binding-vs-provider-class-naming-residual`); the full
  certification-pipeline mechanism it names remains a separate, not-yet-approved
  design in the pending draft.

## Rejected Alternatives

- **Treat the draft as the first proposal in this area and refine it verbatim:** rejected —
  it would restate already-shipped ADR-0034/0035/0037 decisions as new and ignore the
  `skills_plugaveis` overlap.
- **Rename the existing `internal/plugins/connectors.InvocationEnvelope` type instead of the
  new concept:** rejected — it is shipped, tested code with existing call sites; renaming the
  unimplemented new concept is lower-risk.
- **Resolve the `skills_plugaveis` overlap unilaterally inside this ADR:** rejected — the
  discovery pass that produced this ADR was not scoped to fully evaluate that pending plan;
  deciding its fate here would be an incidental by-product, not a considered decision.

## Validation Requirements

- `grep -rn "RoleInvocationPlan" internal/ cmd/` returns zero hits prior to its first
  implementation, confirming no naming collision at introduction time.
- Any `PreflightResult` implementation is covered by a contract test asserting its fields are
  sourced from, not duplicated from, `check_slots.go`/`role_weapon.go`.
- Any phase-transition authority extension leaves `internal/domain/state_machine.go`'s
  existing `stateTransitions` entries unchanged, verified by the existing state-machine test
  suite passing unmodified.

## Scope Boundary

This ADR records the architectural sequencing, naming, and (as of the 2026-09-15 amendment)
V1/V2 layering decisions only. It does not authorize implementation of the Role→Weapon
resolver, `PreflightResult`, the phase-transition authority extension, the context-loading
resolver, or any part of V1's own P1–P9 plan — all remain `implementation_handoff` items
outside Strategist's execution scope, requiring a separately authorized coding task. It does
not authorize implementation of the certification-pipeline core or the ranked-class
mechanism named by the "Ranked Class Resolution" amendment above (that remains
`.analysis/pending/cli_refactor/v2/02-ranked-role-binding.md`'s own,
separately-approved design scope), nor does it resolve V1's internal `mission_status`
inconsistency (SQ-C, tracked but not fixed).
