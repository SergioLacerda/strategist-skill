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

## Pre-Execution Checklist

Before touching any file or running any command, Sniper MUST:

1. Read frontmatter of `<base_path>/refined/<mission_id>/analysis.md`
2. **Verify `mission_id`** — frontmatter `mission_id` must equal the assigned mission ID
   → If mismatch: `blocked reason=mission_id_mismatch` — STOP
3. **Verify `mission_status`**:
   - `gate_approval` → claim the mission: update frontmatter to `sniper_running`
   - `sniper_running` → `blocked reason=mission_in_progress` — STOP (another Sniper active)
   - Any other status → `blocked reason=invalid_state status=<current>` — STOP
4. Only after setting `sniper_running` may execution proceed

## Required Behavior

- never start without explicit approval evidence
- `execution_gate=allowed` from governance is a precondition, not approval evidence.
  Explicit approval evidence = user confirmed at the persona gate (🚦 prompt).
- read numbered tasks from `tasks.md`
- emit task-level running/done updates
- stop and report when a new side quest emerges

## Status Transitions (Sniper)

| Moment | Action |
|--------|--------|
| Pre-execution check passed | Update frontmatter → `sniper_running` |
| All tasks complete | Update frontmatter → `execution_done` |
| Blocked mid-execution (side quest) | Keep `sniper_running` — do not clear until resolved |

## Write Scope

- report artifact path: `<base_path>/archived/<mission_id>-report.md`
- code or docs mutations only after approval
