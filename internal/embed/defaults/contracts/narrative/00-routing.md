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

1. Critical Hit conditions satisfied (see `critical-hit.yaml`) → Critical Hit
   (plain move, or closure move when an explicit completion claim + evidence are present)
2. Implementation Short Route conditions satisfied → Implementation Short Route
3. Default → Main Mission

**When in doubt → Main Mission. Conservatism is the safe default.**

## Scout — Intake Router

Steps 2–4 above are decided by **Scout**, an internal pre-pipeline role (the "Intake
Router"). Scout is not a slot and not a configurable provider — it is built-in
Strategist behavior, analogous in scope-boundedness to Sniper but positioned before
the pipeline instead of at the end of it. Scout runs immediately after intake
(`prompt-intake`).

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

### Discovery Weapon Resolution by Subtype

Discovery invocation target does not depend on `discovery_subtype` or on
`active.slots.discovery` — all discovery subtypes (`creative`, `evaluation`,
`diagnostic`, `closure_evidence`) always resolve to `internal_skills/ranger`
(`kind=native_role`). The parent agent embodies Ranger directly — the same
native-role mechanism already used for execution/`sniper` — reading
`roles/ranger.yaml` + `internal_skills/ranger/SKILL.md` and performing
discovery under that contract. An external discovery plugin configured at
`active.slots.discovery` is never consulted for discovery invocation; the
field remains present for provider-metadata/future use but does not gate any
current subtype.

This exists because an external discovery plugin's own `SKILL.md` is authored
independently of Strategist and cannot be relied on to honor
`roles/ranger.yaml` or subtype-specific obligations, even when its manifest
declares `native` or `adapter` support — declared support in a manifest is a
capability claim by whoever wrote it, never a live behavior guarantee. This
was previously handled by a Post-Route Capability Check applied only to the
`creative` subtype (the other three subtypes were already native-only); that
check was removed once a live invocation of a manifest-compliant `creative`
weapon (`brainstorming`, declaring `discovery_subtype_support: creative:
native`) surfaced structural incompatibilities with Ranger's autonomous
single-shot contract that the manifest check could not have caught (see
`.analysis/refined/20260728-ranger-drift-eval/`). Only
`internal_skills/ranger`, authored by Strategist itself, can be trusted to
compose with `roles/ranger.yaml` per its own documented "Invocation Contract".

### Provider Resolution Policy (ADR-0028)

This section does not apply to discovery — discovery's resolution is settled by
§ Discovery Weapon Resolution by Subtype above (always native, no exception, for a stronger,
independently-established reason: a live invocation of a manifest-compliant
discovery plugin surfaced structural incompatibilities that a per-request policy cannot
detect in advance). It applies to **refinement**, and to any other slot where
`strategist check` reports a `fallback=<role>(native_role)` annotation for the
configured provider (see `roles/default.yaml` and
`internal/check/check_slots.go#resolveNativeFallback`).

`static strategist check` validation of an external skill plugin slot (valid
`skill.yaml`, matching `risk_score`) does not prove the slot plugin is invocable by
the current agent runtime — only a live mission invocation reveals that. When a
configured external skill plugin fails at invocation time, and `strategist check`'s
own SLOTS output shows a compatible native-role fallback exists for that slot,
the agent MUST resolve the block according to `active.yaml`'s
`provider_resolution_policy` (absent or empty defaults to `ask`):

- **`block`** — preserve the strict, pre-ADR-0028 behavior: emit
  `role_invocation_failed`, stop, and wait for the user to fix the provider
  configuration or reconfigure the slot. No fallback is offered automatically.
- **`ask`** (default) — present the block to the user with the concrete choice:
  (a) use the compatible native role for this mission, (b) reconfigure
  `active.slots.<phase>` to a different, available provider, or (c) accept the
  mission's current terminal state (e.g. analysis-only) without refinement. Do
  not pick for the user. Record the choice made (e.g. as an ADR or mission
  decision) rather than silently repeating the question on the very next
  mission without referencing the prior one.
- **`native`** — use the compatible native role automatically, without asking,
  but the agent MUST emit degradation evidence identifying the configured
  provider, the effective (fallback) provider, and the reason, e.g.:
  `[Strategist] phase=<phase> status=degraded reason=native_fallback configured_provider=<x> effective_provider=<y>`.
  Auto-substitution under this policy is still never a substitute for the
  Strategist Approval Gate, still preserves the resolved role's own write scope
  and role contract (`must`/`must_not`), and still never applies to discovery
  or to a slot where no compatible native role exists.

None of the three policies authorizes skipping the Approval Gate, changing
write scope, or treating an external skill plugin failure as license to invent a
provider that isn't `roles/<id>.yaml`-backed. `block` and `ask` never mutate
`active.yaml`; only `native` changes *behavior* for the current mission, never
the stored configuration — an operator who wants the native role as the
permanent, standing choice still edits `active.slots.<phase>` (via
`strategist install`/`compile`, not a manual hand-edit, to avoid an
unacknowledged `hash_mismatch` — see
`docs/runbooks/role-invocation-failed.md` § Refinement-Specific Escalation).

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

`enforced_by` tags use the unified 3-tier vocabulary defined in
`machine/errors.yaml` (`machine_enforced` / `machine_observed` /
`agent_only`). Reviewed against actual Go call sites (2026-08-30): every
invariant below is `agent_only` — there is no live-mission FSM in Go that
gates routing or execution; `internal/domain/pipeline_bypass.go`'s
`EvaluatePipelineBypass` implements the matching decision logic for the first
invariant but has zero non-test callers repo-wide, so it is not on a
reachable path today.

- No direct repository mutation without canonical pipeline evidence — `enforced_by: agent_only`
- No execution without explicit Strategist Approval Gate acceptance — `enforced_by: agent_only`
- No slot work performed by Strategist itself — `enforced_by: agent_only`
- The invoking local context (any adapter, orchestrator, or harness) may block or permit execution — it does NOT replace the canonical pipeline sequence once Strategist is invoked — `enforced_by: agent_only`
- `execution_gate=allowed` from local context never substitutes the Strategist Approval Gate (explicit user approval) — `enforced_by: agent_only`
- The Strategist Approval Gate is required on all routes: Critical Hit, Implementation Short Route, and Main Mission — `enforced_by: agent_only`
- A missing or uncallable resolved execution provider is a blocked state — never a reason for direct execution — `enforced_by: agent_only`

## Scope Invariant

Strategist produces analysis and documentation only.
Code mutation is never in scope — on any route, including Critical Hit.
Route selection (Critical Hit vs Implementation Short Route vs Main Mission) is handled
internally by Scout, the Intake Router — the delegating agent does not need to specify a route.

Requests to remove, edit, merge, or refactor source files or tests are not Critical Hit.
They are implementation/materialization requests. Strategist may analyze/refine them, but
execution is allowed only if the resolved execution provider contract permits that mutation.
The default Sniper contract does not.
