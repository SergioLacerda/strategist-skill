---
phase: routing
slot: null
requires_approval: false
contract: null
---
# Strategist — Contract 00: Routing

## Purpose

Resolve the route before any mission work starts.

## Routes

- **Quick Draw** — only for explicit quick capture / note append requests
- **Critical Hit** — internal capability for workspace artifact management
  (`pending/`, `refined/`, `archived/`, `done/`). Not a route mutually exclusive with the
  pipeline — may fire at intake or mid-mission. Two modes: plain move (no evaluation, no
  evidence) and closure move (relocate a `pending/`/`refined/` package into `done/`,
  requires an explicit completion/validation claim and a supplied evidence summary).
  Reaching `documentation_applied` at the end of a main_mission is documentation
  completion, not implementation/validation evidence, and does not by itself make a
  package a closure candidate. Never infers implementation status on its own. See
  `critical-hit.yaml` and `11-critical-hit.md`.
- **Implementation Short Route** — for already-refined implementation/materialization requests
- **Main Mission** — every other request

## Route Selection Order

1. Quick Draw keywords detected → Quick Draw
2. Critical Hit conditions satisfied (see `critical-hit.yaml`) → Critical Hit
   (plain move, or closure move when an explicit completion claim + evidence are present)
3. Implementation Short Route conditions satisfied → Implementation Short Route
4. Default → Main Mission

**When in doubt → Main Mission. Conservatism is the safe default.**

## Scout — Intake Router

Steps 2–4 above are decided by **Scout**, an internal pre-pipeline role (the "Intake
Router"). Scout is not a slot and not a configurable provider — it is built-in
Strategist behavior, analogous in scope-boundedness to Sniper but positioned before
the pipeline instead of at the end of it. Scout runs immediately after intake
(`prompt-intake`) and after `quick_draw_detection` finds no match.

Scout classifies the request and emits one compact, auditable `route_decision`
(never a discovery report — see `schemas/scout-route-decision.schema.yaml` and
`contracts/machine/scout-routing.yaml`):

- `mission_id`, `role: Scout`, `component: intake_router`
- `request_category` — coarse classification (e.g. `implementation_evaluation`)
- `selected_route` — `critical_hit` | `implementation_short_route` | `full_pipeline`
- `route_reason`, `route_confidence` (0.0–1.0)
- `evidence_state` — `explicit` | `insufficient` | `requires_discovery`
- `discovery_subtype` — set when `full_pipeline` + `requires_discovery` (see `03-discovery.md`)
- `fallback_route` — always `full_pipeline`
- `gate_required` — always `true`

Scout must NOT: perform deep discovery or read implementation surfaces beyond what
is explicitly supplied; invoke Sniper directly; bypass the Strategist Approval Gate
on any route; replace Ranger when evidence review is required; infer evidence that
was not explicitly supplied. If Scout would need to read broad implementation
surfaces to answer a classification question, it has crossed into Ranger territory
and must select `full_pipeline` instead.

### Post-Route Capability Check

Immediately after Scout emits `route_decision` with `evidence_state: requires_discovery`
(before the discovery weapon is invoked), check the resolved weapon's `skill.yaml`
`discovery_subtype_support` field against the required `discovery_subtype`. This runs
as part of Scout's routing responsibility (see `contracts/machine/scout-routing.yaml`
§ `post_route_capability_check`) — not at classic preflight time, since preflight
runs before intake/routing and before `discovery_subtype` exists.

If the resolved weapon does not declare support for the required `discovery_subtype`,
emit `provider_capability_mismatch` and stop **before** invoking the weapon. Do not
invoke the weapon to discover the mismatch empirically — that wastes an invocation and
risks the weapon partially acting before the mismatch is caught. See `preflight.yaml`
for the full error condition and remediation hint.

## Main Mission Sequence

`bootstrap → preflight → intake → discovery → refinement → approval_gate → execution? → adr? → learning`

Main mission completion does not imply implementation completion. The refined package
remains in `<base_path>/refined/<mission_id>/` by default — that is the normal, expected
terminal state for an analysis/refinement mission, not a gap to be closed. `done/` is
reached only through a separate Critical Hit closure, triggered by an explicit
implementation/validation claim plus a supplied evidence summary (see `11-critical-hit.md`
→ Stale Card Detection for how candidates are surfaced, and → Insufficient Evidence for
what does not qualify).

## Critical Hit Sequence

`bootstrap → preflight → intake → critical_hit_gate → execution → learning`

Same sequence for both plain move and closure move; `critical_hit_gate` renders the mode-appropriate
inline gate (see `11-critical-hit.md`).

## Implementation Short Route

For requests that are already framed as implementation/materialization and arrive with sufficient context:

`bootstrap → preflight → intake → implementation_context_validation → approval_gate → execution_provider_resolution → execution/materialization → learning`

Short route conditions (ALL must hold):
- request explicitly asks for implementation/materialization
- local context or user prompt provides enough scope to avoid full discovery
- documentation/materialization targets are clear
- no unresolved ambiguity
- code mutation is not required
- Git mutation is not required

This route skips full Ranger/Archivist expansion only when context is already refined enough. It does NOT skip the Strategist Approval Gate. If any condition fails, fall through to Main Mission.

### Annotation Limits

Short route may annotate implementation status (e.g. `not_implemented`) only when
`evidence_state: explicit` **and** the annotation is narrow — a single artifact or
card, not a broad implementation-status question spanning multiple systems. Broad
implementation-status questions MUST route to `full_pipeline` with
`discovery_subtype: evaluation` instead (see `03-discovery.md`). If the status has
to be discovered rather than read directly, Short Route must not infer it — Scout
routes to Ranger. The Strategist Approval Gate applies to this annotation exactly
as it does to every other route — it is never skipped, even for a narrow,
evidence-explicit annotation.

Still delegates execution to the resolved provider. Direct execution by the Strategist shell is never permitted.

## Contract Lookup

When operating inside the main mission, consult contracts in this order:

1. `01-bootstrap.md`
2. `02-intake.md`
3. `03-discovery.md`
4. `04-refinement.md`
5. `05-approval-gate.md`
6. `06-execution.md`
7. `07-adr.md`
8. `08-learning.md`
9. `09-response.md`
10. `10-telemetry.md`
11. `11-critical-hit.md`
12. `machine/scout-routing.yaml` — Scout's route-decision trigger conditions and invariants
13. `../schemas/scout-route-decision.schema.yaml` — Scout's route_decision field contract

## Invariants

- No direct repository mutation without canonical pipeline evidence
- No execution without explicit Strategist Approval Gate acceptance
- No slot work performed by Strategist itself
- The invoking local context (any adapter, orchestrator, or harness) may block or permit execution — it does NOT replace the canonical pipeline sequence once Strategist is invoked
- `execution_gate=allowed` from local context never substitutes the Strategist Approval Gate (explicit user approval)
- The Strategist Approval Gate is required on all routes: Quick Draw, Critical Hit, Implementation Short Route, and Main Mission
- A missing or uncallable resolved execution provider is a blocked state — never a reason for direct execution

## Scope Invariant

Strategist produces analysis and documentation only.
Code mutation is never in scope — on any route, including Critical Hit.
Route selection (Critical Hit vs Implementation Short Route vs Main Mission) is handled
internally by Scout, the Intake Router — the delegating agent does not need to specify a route.

Requests to remove, edit, merge, or refactor source files or tests are not Critical Hit.
They are implementation/materialization requests. Strategist may analyze/refine them, but
execution is allowed only if the resolved execution provider contract permits that mutation.
The default Sniper contract does not.
