# Strategist — Contract 05: Approval Gate

## Inputs

- refined package under `<base_path>/refined/<mission_id>/`
- optional side quest summary

## Outputs

- explicit approval decision
- gate audit entry when approved
- `plan_only` when declined or when no executable tasks exist

## Required Behavior

- read `tasks.md` before deciding whether to present the gate
- stop and wait for explicit user approval before Sniper
- re-present plan content on `review`
- re-emit mission checkpoint when execution is approved

## Gate States

- `plan_only`
- `awaiting_confirmation`
- `approved`
- `declined`
