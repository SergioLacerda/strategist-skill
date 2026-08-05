---
phase: refinement
slot: refinement
requires_approval: false
contract: write_analysis
---
# Strategist — Contract 04: Refinement

## Owner

Archivist (`refinement`)

## Inputs

- transient analysis handoff artifact: `<base_path>/pending/<mission_id>-analysis.md`
- `mission_contract.planning_rules`
- context dossier
- applicable treasure chests
- `contracts/machine/handoff-contract.yaml#refinement_context_policy` — the source
  deduplication policy; consult before reopening any source listed in the Ranger
  artifact's `sources_consulted[]`

## Outputs

- `<base_path>/refined/<mission_id>/analysis.md`
- `<base_path>/refined/<mission_id>/proposal.md`
- `<base_path>/refined/<mission_id>/design.md`
- `<base_path>/refined/<mission_id>/tasks.md`
- execution handoff fields validated by `.strategist/schemas/handoff-archivist-to-sniper.schema.yaml`
- `evidence_pack_path` when present in the Ranger analysis artifact; passed through, never regenerated
- one appended line to `.strategist/memory/handoff-metrics.jsonl` (skill.yaml#handoff_metrics_log)

## Required Behavior

- treat the Ranger transient analysis artifact as the canonical refinement input
- reuse the Ranger artifact's `relevant_sources_hint` (Search ability output) and
  `selected_runbooks_hint` (select_runbook ability output) by default instead of
  re-running Search or select_runbook; only re-run either with a declared reason
  from `contracts/machine/handoff-contract.yaml#refinement_context_policy.allowed_reasons`
  (see `roles/archivist.yaml#canonical.reuse_search_cache`)
- consult treasure chests before refinement
- before reopening any source listed in the Ranger artifact's `sources_consulted[]`,
  check `contracts/machine/handoff-contract.yaml#refinement_context_policy` — reopen
  only for one of its `allowed_reasons`, and state the matching reason explicitly in
  the refined artifact that needed it (see skill.yaml's
  `archivist_reopens_discovery_sources_without_declared_reason` forbidden_behaviors entry)
- on completion, append one line to `.strategist/memory/handoff-metrics.jsonl`
  (skill.yaml#handoff_metrics_log) — nulls are expected for `brief_compression_ratio`/
  `evidence_coverage_ratio` when the Ranger artifact did not populate `evidence_cards[]`
- produce the four-file refined package
- preserve `evidence_pack_path` from the Ranger analysis artifact when present; the four-file package shape does not change
- promote the Ranger analysis artifact from `pending/` into `<base_path>/refined/<mission_id>/analysis.md`
- classify side quests and surface them at the approval gate
- classify every `tasks.md` / `implementation_plan` item by `task_type`: `documentation_target`,
  `analysis_artifact`, `implementation_handoff`, or `out_of_scope` (see
  `handoff-archivist-to-sniper.schema.yaml`). Only `documentation_target` items are
  Sniper-executable. `implementation_handoff` items (code, hook, config, or test mutation)
  must never be phrased as executable Sniper tasks — they are handed off, not queued
  for materialization.
- evaluate `contracts/machine/handoff-contract.yaml#handoff_verification_policy`
  for Archivist -> Sniper handoffs. When the policy triggers, include optional
  `handoff_verification` metadata in the handoff with `objective`, `boundary`,
  `classification`, and `gate` challenge types. This semantic acknowledgment
  complements the YAML structure contract; it never replaces Approval Gate review.
- a second Handoff Challenge transition, `ranger_to_archivist`, is available in
  `internal/handoff` (`TransitionRangerToArchivist`, challenge types `recall`,
  `boundary`, `classification`, `verdict` — see `03-discovery.md` § Optional
  Handoff Challenge). It is advisory-first: no policy in this workspace
  currently sets `RequiredTypes` for it. Wiring a required-by-default risk
  policy for this transition is a future decision, not made here — see
  `.analysis/refined/20260803-handoff-challenge-extensions/design.md` § Item 1.
- when the mission type is evaluation or audit and the Ranger discovers completed work
  requiring cleanup (archiving finished missions, removing obsolete files): treat that
  cleanup as an opportunity attack, not a main task. The main mission resolves as
  `analysis_delivered`. The cleanup is offered via `opportunity_gate` manifest.
- never emit a single-file refined artifact as the canonical result

### Optional Decision Ledger

Archivist MAY consolidate mission-scoped choices as `decisions:` entries
(`schemas/decision.schema.yaml`) — stable `DEC-NNN` ids, `status`, cited
`evidence` ids, `alternatives_rejected`, `confidence`, `supersedes` — when a
mission's own complexity warrants a durable ledger rather than prose alone.
This is optional. When both `decisions:` and `evidence:` are present,
`machine/mission-quality.yaml`'s predicates describe what a well-formed
package looks like, and a failed predicate is surfaced at the gate
(advisory only — see `05-approval-gate.md`).

## Write Scope

- authorized paths:
  - `<base_path>/refined/<mission_id>/proposal.md`
  - `<base_path>/refined/<mission_id>/analysis.md`
  - `<base_path>/refined/<mission_id>/design.md`
  - `<base_path>/refined/<mission_id>/tasks.md`

## Gate Condition

- if `tasks.md` is empty or absent, mission resolves as `analysis_delivered`

## Language

Write the four-file refined package in `active.language.docs`, independent of the language used
in the surrounding conversation.

## Status Transitions (Archivist)

- On start → update transient analysis frontmatter `mission_status: archivist_pending`
- On complete (all four files written, and transient pending artifact removed) → update promoted analysis frontmatter `mission_status: archivist_done`
