---
phase: approval_gate
slot: null
requires_approval: true
contract: null
---
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

## Side Quests at the Gate

If Archivist identified side quests in `opportunity_manifest`:

1. Present list after the main mission block
2. Assign each a unique ID (SQ-NNN)
3. Show estimated impact and dependencies
4. User may select a subset for execution
5. Unselected side quests are recorded as `sq_backlog` — not discarded
6. Partial approval is valid — Sniper executes only the approved items

Gate display format:

```
📋 MAIN MISSION
   Proposal:  refined/<mission_id>/proposal.md
   Tasks:     refined/<mission_id>/tasks.md — N task(s)

📦 SIDE QUESTS (if any)
   [SQ-001] <description> — impact: <low|medium|high>
   [SQ-002] <description> — impact: <low|medium|high>

Approve? (yes / no / select IDs)
```

## Status Transitions

- Gate presented → frontmatter: `gate_pending`
- Gate approved → frontmatter: `gate_approval`
- Gate declined → frontmatter remains `gate_pending` (`plan_only` state — valid, not error)

## Gate States

- `plan_only`
- `awaiting_confirmation`
- `approved`
- `declined`
