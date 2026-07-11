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

## Claim Protocol

Before any action, Sniper MUST:

1. Read `<base_path>/refined/<mission_id>/analysis.md`
2. Confirm `mission_status: gate_analysis_accepted`
3. Write atomically: `mission_status: sniper_running` + `claimed_by: <agent_id>`
4. If status is `sniper_running` on check → emit `blocked reason=already_claimed` → STOP

If `mission_status` is not `gate_analysis_accepted` → emit `reason=gate_approval_missing` → STOP.

## Required Behavior

- execute claim protocol before any action
- never start without explicit approval gate acceptance evidence
- read numbered documentation tasks from `tasks.md`
- materialize ONE documentation target per loop — never batch
- emit task-level running/done updates
- stop and report immediately when an out-of-scope write or non-documentation target emerges (do not decide strategy)
- update `mission_status: documentation_applied` on completion

## Status Transitions (Sniper)

- Claim confirmed → `sniper_running`
- All documentation tasks complete → `documentation_applied`

## Write Scope

Sniper write scope is limited to workspace documentation files and declared documentation targets only. Code mutation is always forbidden.

- report artifact path: `<base_path>/archived/<mission_id>-report.md`
- documentation files (`.md` and diagram/documentation assets) declared in `documentation_targets`
- files inside `<base_path>/` (workspace) declared by Archivist
- files outside `<base_path>/` only when explicitly declared by Archivist and accepted at the approval gate
- **code files are forbidden** — no code mutation: no `.go`, `.ts`, `.py`, `.js`, `.sh`, or other source code
- **Git mutating commands are forbidden** — no `git add`, `git commit`, `git push`, `git reset`, `git merge`, or any other state-modifying Git operation

The default Sniper contract cannot mutate code or test files, and this restriction cannot
be bypassed by the parent agent performing the mutation directly instead of Sniper. A
request for code/test mutation while Strategist is active produces analysis/handoff only,
unless a separately configured execution provider whose contract explicitly permits
mutation is resolved in its place.
