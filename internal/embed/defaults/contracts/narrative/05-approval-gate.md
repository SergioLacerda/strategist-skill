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

`enforced_by` tags use the unified 3-tier vocabulary defined in
`machine/errors.yaml` (`machine_enforced` / `machine_observed` /
`agent_only`). All five items below are `agent_only` as of 2026-08-30: no Go
code reads `tasks.md`, tracks whether the gate was presented, or validates
gate-prompt content — this phase is entirely agent-narrative.

- read `tasks.md` before deciding whether to present the gate — `enforced_by: agent_only`
- stop and wait for explicit user review response before Sniper — `enforced_by: agent_only`
- re-present analysis content on `review` — `enforced_by: agent_only`
- re-emit mission checkpoint when documentation targets are accepted — `enforced_by: agent_only`
- if `implementation_plan` contains any `task_type: implementation_handoff` item, state this
  explicitly in the gate prompt (see Gate Display With Implementation Handoff below) — `enforced_by: agent_only`

## Gate Acceptance Is Not Code Mutation Approval

Approval Gate acceptance means the refined analysis is correct and, if the package contains
`documentation_target` items, that Sniper may materialize them. It never means:

- code implementation is authorized;
- `implementation_handoff` items may be executed by Sniper or by the parent agent directly;
- the pipeline may continue past the gate into source-code mutation because the user said
  `sim`/`accept`/`yes` to the refined package as a whole.

If the accepted package contains `implementation_handoff` items, those items remain
outside Strategist after the gate. The mission resolves as `analysis_delivered` when
there are no accepted `documentation_target` items — record the acceptance with the
`gate_approved_analysis_only` event (`APPROVAL_GATE → DONE_ANALYSIS`), never with
`gate_approved` (which enters the handoff challenge and can only end in `sniper_done`)
or `gate_denied` (which records a rejection) — or `documentation_applied` after
Sniper materializes accepted documentation targets. A mission that already
entered the handoff challenge with such a package is closed with
`handoff_challenge_not_applicable` (`HANDOFF_CHALLENGE → DONE_ANALYSIS`). Both
analysis-only events are rejected when the refined `tasks.md` declares a
`documentation_target`. In both cases, `implementation_handoff`
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

📋 TAREFA PRINCIPAL (if assertion claims were emitted)
   <id> — <short label> — confiança: <confidence_percent>%
   ...

❓ DÚVIDAS (open questions — claim_kind: question, if any)
   <id> — <statement>
   ...

📄 DOCUMENTATION TARGETS (outside <base_path>, if any)
   <path> — <description>

📦 SIDE QUESTS (if any)
   [SQ-001] <description> — confiança: <confidence_percent>%
   [SQ-002] <description> — confiança: precisa investigar

Is the analysis correct?  (accept / review / reject)
```

A `📋 TAREFA PRINCIPAL` row is one correlated `assertion` claim: id, a short
label, and `confidence_percent`. A `❓ DÚVIDAS` row is one `claim_kind:
question` claim, shown by statement only — never with a percentage, because a
claim too uncertain to support an assertion is, by contract, a question
(`machine/confidence-governance.yaml#rules.low_assertion_review`), not a
manufactured low score. A side quest shows `confidence_percent` when Archivist
assessed one, or the literal text "precisa investigar" when
`investigation_required: true` — see `schemas/handoff-*.schema.yaml#side_quests`
field_descriptions. Never invent or round a confidence value for either a
claim or a side quest to avoid showing "precisa investigar"; an honest
unknown is always preferable to a guessed number.

If the loaded confidence review has `review_required: true` or any
`violations`, append one line after the SIDE QUESTS block, e.g.:
`⚠️  revisão recomendada — 1 afirmação sem evidência suficiente (detalhe:
strategist metrics confidence --mission <id>)`. Otherwise omit the line
entirely — do not restate `review_required: false` or an empty violations
list.

The full cross-agent calibration payload (policy version, low/medium/high
distribution, per-agent sample/coverage/calibration, claim-kind counts,
evidence coverage, calibration status, missing/rejected/duplicate counts) is
not inlined at the gate. It stays available as an internal/debug surface via
the pre-existing `strategist metrics confidence --mission <id>` and
`strategist mission view --mission-id <id> --json` commands — no new command
is introduced for this; both already materialize
`internal/telemetry.LoadConfidenceGateReview`. This is a display
simplification, not a data reduction: the gate decision (`review` default on
`review_required`, low-confidence-must-be-a-question, etc.) still reads the
full review, only the rendered chat message is per-item.

Confidence is a review signal, not an approval. Low-confidence or unsupported
assertions default to `review`, while questions remain visible as questions.
Confidence percentages must not be presented as empirical calibration when
the sample is `no_sample` or has no declared ground-truth event.

Confidence cannot invoke Sniper by itself. The existing human
Approval Gate remains mandatory, and `implementation_handoff` items remain
outside Sniper even when confidence checks pass; only accepted
`documentation_target` items may proceed to materialization.

The runtime advisory projection is materialized by
`internal/telemetry.LoadConfidenceGateReview` from the persisted, validated confidence
history. An empty, malformed, rejected, or incomplete history produces a
review signal and is never interpreted as approval.

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

`enforced_by: agent_only` — same 2026-08-30 review as above: nothing in Go
tracks whether the gate was ever shown or accepted for a given mission (no
mission_status FSM exists outside Markdown frontmatter), so this invariant
holds only because the orchestrating agent follows it.
