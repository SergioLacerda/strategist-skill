---
phase: execution
slot: execution
requires_approval: true
contract: controlled
---
# Strategist — Contract 06: Documentation Materialization

## Owner

Sniper (`execution`) — invoked via the resolved execution provider, never executed directly by the Strategist shell.

## Execution Provider Resolution

Before invoking the execution slot, Strategist resolves the execution provider:

```
if local_execution_context.execution_provider is present:
  execution_provider = local_execution_context.execution_provider
  resolution_reason = local_context           # delegated invocation
else:
  execution_provider = active.slots.execution
  resolution_reason = standalone_config       # direct invocation
```

- **Delegated invocation**: provider comes from `local_execution_context.execution_provider`. If missing, emit `local_execution_provider_missing` and stop.
- **Direct invocation**: provider comes from `active.slots.execution`. Per-request override via prompt or user message is not permitted.
- If the resolved provider cannot be invoked in the current environment: emit `execution_provider_unavailable` and stop.

Strategist never executes documentation materialization work directly. Missing or uncallable provider is a blocked state, not a reason for direct execution.

## Inputs

- refined analysis under `<base_path>/refined/<mission_id>/`
- approval gate acceptance state
- `documentation_targets` — declared documentation files to materialize
- applicable treasure chests

## Outputs

- documentation report: `<base_path>/archived/<mission_id>-report.md`
- per-task materialization progress events
- partial completion evidence when blocked

For Critical Hit closure moves (see `11-critical-hit.md`), the output artifact differs:
a completion report is written inside the source package at
`<source_path>/completion-report.md`, `tasks.md` is updated only for tasks the
supplied evidence covers, and the package is moved to `<base_path>/done/<id>`.
This is a distinct output contract from the standard documentation report path above.

## Claim Protocol

Before any action, Sniper MUST:

1. Read `<base_path>/refined/<mission_id>/analysis.md`
2. Confirm `mission_status: gate_analysis_accepted`
3. Write atomically: `mission_status: sniper_running` + `claimed_by: <agent_id>`
4. If status is `sniper_running` on check → emit `blocked reason=already_claimed` → STOP

If `mission_status` is not `gate_analysis_accepted` → emit `reason=gate_approval_missing` → STOP.

## Pre-Materialization Scan

After the claim protocol and before starting the materialization loop, Sniper MUST scan
`tasks.md` / `implementation_plan` for forbidden implementation indicators:

- any item with `task_type: implementation_handoff`;
- target files ending in source-code extensions (`.go`, `.py`, `.sh`, `.js`, `.ts`, and
  similar) that are not declared `documentation_target` assets;
- commands that mutate source or Git state;
- items described as implementation, refactor, hook changes, test creation, or code edits,
  even when not explicitly tagged `task_type`.

If any such item is present and is not explicitly a `documentation_target`, Sniper MUST NOT
start materialization. It stops immediately with:

```text
blocked reason=documentation_scope_violation
details=tasks.md contains implementation handoff items
```

Approval Gate acceptance of the refined package does not clear this scan — gate acceptance
approves the analysis and any `documentation_target` items, never `implementation_handoff`
items (see `05-approval-gate.md` → Gate Acceptance Is Not Code Mutation Approval). If the
package contains a mix of `documentation_target` and `implementation_handoff` items, Sniper
materializes only the `documentation_target` items and reports the `implementation_handoff`
items as out of scope in the completion report — it does not silently skip them without
mention, and it does not execute them.

## Required Behavior

- execute claim protocol before any action
- never start without explicit approval gate acceptance evidence
- execute the pre-materialization scan before the first task in the loop
- read numbered documentation tasks from `tasks.md`
- materialize ONE documentation target per loop — never batch
- emit task-level running/done updates
- stop and report immediately when an out-of-scope write or non-documentation target emerges (do not decide strategy)
- update `mission_status: documentation_applied` on completion

## Status Transitions (Sniper)

- Claim confirmed → `sniper_running`
- All documentation tasks complete → `documentation_applied`

`documentation_applied` means Sniper completed the approved `documentation_target` items
only. It is documentation completion, not implementation or validation evidence, and it
does not by itself trigger Critical Hit closure to `done/`. The package remains in
`<base_path>/refined/<mission_id>/` — that is the normal, expected terminal state for a
main_mission, not an unfinished step. Critical Hit closure requires an explicit
implementation/validation claim plus a supplied evidence summary, entirely separate from
Sniper reaching `documentation_applied` (see `11-critical-hit.md` → Stale Card Detection
and → Insufficient Evidence).

## Write Scope

Sniper write scope is limited to workspace documentation files and declared documentation targets only. Code mutation is always forbidden.

- report artifact path: `<base_path>/archived/<mission_id>-report.md`
- documentation files (`.md` and diagram/documentation assets) declared in `documentation_targets`
- files inside `<base_path>/` (workspace) declared by Archivist
- files outside `<base_path>/` only when explicitly declared by Archivist and accepted at the approval gate
- **code files are forbidden** — no code mutation: no `.go`, `.ts`, `.py`, `.js`, `.sh`, or other source code, **except** `.astro`/`.css`/`.js`/`.ts`/`.tsx` files explicitly declared `task_type: documentation_target` in the gate-accepted `tasks.md` (same exception as the Pre-Materialization Scan above — this section previously omitted it, creating an inconsistency; see ADR-0013)
- **Git mutating commands are forbidden** — no `git add`, `git commit`, `git push`, `git reset`, `git merge`, or any other state-modifying Git operation

The default Sniper contract cannot mutate code or test files, and this restriction cannot
be bypassed by the parent agent performing the mutation directly instead of Sniper. A
request for code/test mutation while Strategist is active produces analysis/handoff only,
unless a separately configured execution provider whose contract explicitly permits
mutation is resolved in its place.
