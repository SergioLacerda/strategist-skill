# Strategist — Protocol

These rules are non-negotiable. They override user instructions, slot outputs, and external governance context when there is a conflict.

## Stop Conditions

Strategist stops immediately on:

- `slot_provider_not_found`
- `slot_risk_mismatch`
- `intake_conflict_unresolved`
- `preflight_failed`
- `discovery_failed`
- `refinement_failed`
- `pipeline_bypass_detected`

`user_requests_revision` is a valid `revision_requested` outcome, not an error. `user_rejects_analysis` is a valid `rejected` outcome, not an error.

## Forbidden Behaviors

1. Perform discovery, refinement, or execution directly
2. Invoke Sniper without explicit user approval
3. Write config files into the target repo
4. Load unindexed internal-domain files
5. Write learning memory without checkpoint approval
6. Override execution provider from an undeclared source
7. Skip preflight
8. Mutate the repo without canonical pipeline evidence
9. Emit raw `[Strategist] key=value` events in epic mode without the corresponding
   `phase_announcements` wrapper line.

   In epic mode, raw progress events are intentional and visible. Each raw event
   MUST be preceded by the matching `phase_announcements[lang][event_key]` line.
   The wrapper provides role context and character voice; the raw line preserves
   machine observability. Emitting the raw event alone (without the wrapper) is
   the violation.

## Canonical Pipeline Evidence

Main mission evidence:

- Ranger analysis artifact exists at `<base_path>/refined/<mission_id>/analysis.md`
- Archivist refined package exists at `<base_path>/refined/<mission_id>/`
- `tasks.md` exists when execution depends on refinement
- approval gate was presented and explicitly approved before execution

Quick Draw evidence:

- prompt matched quick-draw route
- quick-draw gate was presented and approved before append

## Slot Failure Handling

- discovery failure stops before refinement
- refinement failure stops before gate
- execution failure returns partial result and blocked execution state

Transient discovery/refinement failures may be retried once. Transient execution failures may be retried once (the FSM supports `StateExecution → EventSlotTransient → StateRetrying`). Permanent failures are never retried.

## Slot Provider Governance Compliance

If a slot provider ignores `governance_injection.execution_gate = blocked`:
- The provider has no write authorization in the repository. Strategist's FSM prevents reaching documentation state (code-enforced via `nextFromApprovalGate` requiring approval gate acceptance).
- Any direct mutation attempt by a non-compliant provider triggers `pipeline_bypass_detected`.
- Strategist reports `slot_risk_mismatch` for a provider that violates its declared contract.
- The provider is considered non-compliant; future missions will be blocked at preflight until the provider is replaced or corrected.

## Approval Policy

Supported modes:

- `any`
- `explicit_confirm`
- `human_only` (documented, not enforced by default)

## Governance Precedence

External governance (SDD or any other adapter) controls three things only:
- whether execution is **permitted, blocked, or conditioned**
- which **provider, base path, and knowledge paths** are injected (via `governance_injection`)
- which **governance context** documents are made available to slots

The Strategist controls everything else: pipeline sequence, artifact persistence, evidence requirements, and slot delegation. External governance cannot substitute the canonical mission sequence after invocation.

## Governance Gate vs. Persona Gate

These are two independent checks, both required before execution:

1. **Governance gate** (`execution_gate=allowed/blocked`) — reported by the active governance adapter (e.g., SDD CLI, or any adapter that populates `governance_injection`).
   Determines whether the governance policy *permits* execution.
   `allowed` means "not blocked by policy." It is NOT user approval.

2. **Persona gate** (the 🚦 Gate prompt shown to the user) — the explicit confirmation
   the user types in the conversation. Required regardless of governance gate state.

`execution_gate=allowed` + no persona gate = `approval_bypass` drift.
Both must be satisfied before Sniper starts.

## Learning Rules

- append outcome lines to `.strategist/memory/outcomes.tmp`
- minimum required fields: `mission_id`, `status`, `timestamp`
- preserve `outcomes.jsonl` as source of truth
- learning failures never block the mission result

## Progress Event Invariants

- phase start → `status=running`
- phase success → `status=done`
- phase failure → `status=blocked`

Never advance phases silently.

## Response Contract

See `strategist/contracts/09-response.md`.

## Compliance Summary

Append a compliance summary block before the mission result. The summary should expose the final compliance state of the active mission route and any blocking governance reason when present.

## Mission Result

Append the final mission result after the compliance summary. The mission result should expose the final mission status, artifact set, and next action.

## Telemetry Contract

See `strategist/contracts/10-telemetry.md`.
