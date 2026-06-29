---
phase: approval_gate
slot: null
requires_approval: true
contract: null
---
# Strategist — Contract 05: Approval Gate

## Inputs

- refined analysis under `<base_path>/refined/<mission_id>/`
- optional documentation targets outside `<base_path>`
- optional side quest summary

## Outputs

- explicit review decision
- gate audit entry when accepted
- `analysis_delivered` when revision/rejection ends the mission without documentation
- `documentation_applied` when Sniper materializes all documentation targets

## Required Behavior

- read `tasks.md` before deciding whether to present the gate
- stop and wait for explicit user review response before Sniper
- re-present analysis content on `review`
- re-emit mission checkpoint when documentation targets are accepted

## Side Quests at the Gate

If Archivist identified side quests during refinement:

1. Present list after the main analysis block
2. Assign each a unique ID (SQ-NNN)
3. Show estimated impact and dependencies
4. User may select a subset for documentation
5. Unselected side quests are recorded as `sq_backlog` — not discarded
6. Partial acceptance is valid — Sniper materializes only the accepted items

Gate display format:

```
📋 MAIN ANALYSIS
   Proposal:    refined/<mission_id>/proposal.md
   Tasks:       refined/<mission_id>/tasks.md — N task(s)

📄 DOCUMENTATION TARGETS (outside <base_path>, if any)
   <path> — <description>

📦 SIDE QUESTS (if any)
   [SQ-001] <description> — impact: <low|medium|high>
   [SQ-002] <description> — impact: <low|medium|high>

Is the analysis correct?  (accept / review / reject)
```

## Status Transitions

- Gate presented → frontmatter: `gate_pending`
- Analysis accepted → frontmatter: `gate_analysis_accepted`
- Revision requested → frontmatter: `gate_revision_requested` (valid, not error — Archivist revisits)
- Rejected → frontmatter: `gate_rejected` (valid, not error)

## Gate States

- `analysis_delivered`
- `revision_requested`
- `rejected`
- `awaiting_review`
- `analysis_accepted`

## Invariant: Gate Is Always Required

The Strategist Approval Gate is mandatory whenever Strategist participates in a request — regardless of:
- invocation mode (direct or delegated)
- route (Main Mission, Critical Hit, Quick Draw, Implementation Short Route)
- external approvals granted by the invoking context, parent orchestrator, or governance system
- `execution_gate=allowed` from the local execution context

External approval or `execution_gate=allowed` means only:
> local policy does not block execution

It does NOT mean:
> the user accepted this Strategist refined package

Both checks are required before execution/materialization:
1. local execution context permits execution (`execution_gate=allowed`)
2. Strategist Approval Gate explicitly accepted by the user in the conversation
