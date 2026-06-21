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

## Status Transitions (Gate)

| Moment | Action |
|--------|--------|
| Gate presented to user | Update analysis frontmatter → `gate_pending` |
| User approves + Sniper executes immediately | Update frontmatter → `gate_approval` |
| User approves + analysis-only mode (no Sniper) | Update frontmatter → `gate_approval` — **no thread lock**; any Sniper may claim later |
| User declines / `plan_only` | Keep frontmatter at `gate_pending` — signals "pending decision, available for retry" |

> `gate_approval` is the only unclaimed state. When set without an immediate Sniper,
> the mission card is available for any future Sniper to discover and execute.
