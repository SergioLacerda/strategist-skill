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

## Required Behavior

- never start without explicit approval evidence
- read numbered tasks from `tasks.md`
- emit task-level running/done updates
- stop and report when a new side quest emerges

## Write Scope

- report artifact path: `<base_path>/archived/<mission_id>-report.md`
- code or docs mutations only after approval
