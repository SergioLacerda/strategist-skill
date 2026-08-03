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
- if `implementation_plan` contains any `task_type: implementation_handoff` item, state this
  explicitly in the gate prompt (see Gate Display With Implementation Handoff below)

## Gate Acceptance Is Not Code Mutation Approval

Approval Gate acceptance means the refined analysis is correct and, if the package contains
`documentation_target` items, that Sniper may materialize them. It never means:

- code implementation is authorized;
- `implementation_handoff` items may be executed by Sniper or by the parent agent directly;
- the pipeline may continue past the gate into source-code mutation because the user said
  `sim`/`accept`/`yes` to the refined package as a whole.

If the accepted package contains `implementation_handoff` items, those items remain
outside Strategist after the gate. The mission resolves as `analysis_delivered` when
there are no accepted `documentation_target` items, or `documentation_applied` after
Sniper materializes accepted documentation targets. In both cases, `implementation_handoff`
items are reported as non-executable handoff work, not as a separate mission status.
Executing the `implementation_handoff` items requires a separate coding task outside
Strategist mode — the Approval Gate does not grant that authorization, regardless of
`execution_gate=allowed` or how emphatically the user accepted the package.

## Handoff Challenge Independence

If the refined package declares `handoff_verification.required: true`, the gate prompt
may show that Sniper must pass a semantic handoff acknowledgment before materialization.
That challenge is not an approval mechanism. Approval Gate acceptance remains the human
decision that moves `mission_status` to `gate_analysis_accepted`; the handoff challenge
only checks whether Sniper preserved objective, boundary, classification, and gate
meaning from the accepted handoff. Passing the challenge never bypasses this gate, and
failing it never counts as user rejection — it blocks execution with a handoff challenge
reason and returns to Archivist repair.

## Gate Display With Implementation Handoff

When the refined package contains `implementation_handoff` items, the gate prompt must
say so before asking for acceptance:

```
📋 MAIN ANALYSIS
   Proposal:    refined/<mission_id>/proposal.md
   Tasks:       refined/<mission_id>/tasks.md — N task(s)

⚠️  IMPLEMENTATION HANDOFF (outside Sniper/Strategist scope)
   <task id> — <objective>  (code/hook/config/test mutation)
   ...
   Accepting this package does not authorize executing these items. They require a
   separate coding task outside Strategist.

Is the analysis correct?  (accept / review / reject)
```

## Side Quests at the Gate

If Archivist identified side quests during refinement:

1. Present list after the main analysis block
2. Assign each a unique ID (SQ-NNN)
3. Show estimated impact and dependencies
4. User may select a subset for documentation
5. Unselected side quests are recorded as `sq_backlog` — not discarded; at mission
   close they get a Riposte capture offer (see § Riposte below)
6. Partial acceptance is valid — Sniper materializes only the accepted items

Gate display format:

```
📋 MAIN ANALYSIS
   Proposal:    refined/<mission_id>/proposal.md
   Tasks:       refined/<mission_id>/tasks.md — N task(s)

🎯 CRITIC (if a rubric evaluation ran)
   score: <0.00–1.00> — <pass|fail>
   gaps:  <must_have_missing / must_not_present items, if any>

📄 DOCUMENTATION TARGETS (outside <base_path>, if any)
   <path> — <description>

📦 SIDE QUESTS (if any)
   [SQ-001] <description> — impact: <low|medium|high>
   [SQ-002] <description> — impact: <low|medium|high>

Is the analysis correct?  (accept / review / reject)
```

## Critic at the Gate (W8/P5)

When the response-critic evaluated the refined package, its result is shown in the
`🎯 CRITIC` line (see `machine/approval-gate.yaml#critic_display`). Rules:

- `fail` → pre-suggest `review` as the default answer in the prompt sentence
  (e.g. "Critic flagged gaps — review?  (accept / **review** / reject)")
- `no_rubric`, or critic did not run → omit the line entirely; never block the gate
- the critic result is advisory display only — it never auto-rejects, never blocks,
  and never substitutes the user's decision

## Riposte (W8/P2)

A parried mission still scores a hit. On `reject` or `revision`, and at mission close
when `sq_backlog` items exist, offer to capture the reason/items as structured backlog
entries via Riposte's own normalize+capture machinery (normative contract:
`machine/riposte.yaml`). Doctrine:

- one combined confirmation at the trigger point — the gate response itself is NOT
  capture confirmation; declining the offer is always valid
- captured entries carry `origin: riposte` and `mission_ref: <mission_id>`; they wait
  in the backlog for a future intake — Riposte never spawns or restarts a mission
- the gate outcome and its FSM transition are unchanged whatever the user answers

## Status Transitions

- Gate presented → frontmatter: `gate_pending`
- Analysis accepted → frontmatter: `gate_analysis_accepted`
- Revision requested → frontmatter: `gate_revision_requested` (valid, not error — Archivist revisits)
- Rejected → frontmatter: `gate_rejected` (valid, not error)

## Gate States

- `analysis_delivered`
- `revision_requested`
- `rejected`
- `analysis_accepted`

`awaiting_review` retired (D10 orphan — no writer, no reader): the "gate is pending a
response" signal is `status=shown` (see `emit_on_show` in
`contracts/machine/approval-gate.yaml`), not a Gate State value. The mission_status
frontmatter equivalent for "pending a response" is `gate_pending` (a different
vocabulary — see `contracts/machine/mission-status.yaml`), not this list.

## Invariant: Gate Is Always Required

The Strategist Approval Gate is mandatory whenever Strategist participates in a request — regardless of:
- invocation mode (direct or delegated)
- route (Main Mission, Critical Hit, Implementation Short Route)
- external approvals granted by the invoking context, parent orchestrator, or governance system
- `execution_gate=allowed` from the local execution context

External approval or `execution_gate=allowed` means only:
> local policy does not block execution

It does NOT mean:
> the user accepted this Strategist refined package

Both checks are required before execution/materialization:
1. local execution context permits execution (`execution_gate=allowed`)
2. Strategist Approval Gate explicitly accepted by the user in the conversation
