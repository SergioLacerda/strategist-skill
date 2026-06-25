---
phase: execution
slot: execution
requires_approval: true
contract: controlled
---
# Strategist — Contract 06: Execution

## Owner

Sniper (`execution`)

## Inputs

- refined package under `<base_path>/refined/<mission_id>/`
- approved gate state
- applicable treasure chests

## Outputs

- execution report: `<base_path>/archived/<mission_id>-report.md`
- per-task execution progress events
- partial completion evidence when blocked

## Claim Protocol

Before any action, Sniper MUST:

1. Read `<base_path>/refined/<mission_id>/analysis.md`
2. Confirm `mission_status: gate_approval`
3. Write atomically: `mission_status: sniper_running` + `claimed_by: <agent_id>`
4. If status is `sniper_running` on check → emit `blocked reason=already_claimed` → STOP

If `mission_status` is not `gate_approval` → emit `reason=gate_approval_missing` → STOP.

## Required Behavior

- execute claim protocol before any action
- never start without explicit approval evidence
- read numbered tasks from `tasks.md`
- execute ONE task per loop — never batch
- emit task-level running/done updates
- stop and report immediately when a new side quest emerges (do not decide strategy)
- update `mission_status: execution_done` on completion

## Status Transitions (Sniper)

- Claim confirmed → `sniper_running`
- All tasks complete → `execution_done`

## Write Scope

- report artifact path: `<base_path>/archived/<mission_id>-report.md`
- workspace files under `<base_path>/` and documentation files (`.md`) anywhere — code mutation is always forbidden regardless of approval gate state
