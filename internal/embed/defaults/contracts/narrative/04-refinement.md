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

- before finishing, persist this boundary's confidence: `strategist metrics record --mission <mission_id> --agent archivist --claim-file -` (claim piped on stdin, never a file under `<base_path>`; see `machine/confidence-governance.yaml#producers.claim_placement`),
  or, when no confidence summary was produced, `strategist metrics record --mission <mission_id> --agent archivist --missing --correlation-key <key> --reason <why>`;
  never finish silently — Archivist also records the critic (`--agent response_critic`) and `mission_quality` boundaries with the same command (see `machine/confidence-governance.yaml#producers`)
- before invoking the selected refinement weapon's own CLI/tooling, apply
  `roles/archivist.yaml#canonical.resolve_weapon_scratch_root` — read
  `skills/<provider>/skill.yaml#scratch_root`, and when it is `runtime`, run the
  weapon with `.strategist/weapon-runtime/<provider_id>/` as its working
  directory, never the host repository root (see `agent-protocol.md` §3
  Refinement Routing)
- treat the selected refinement weapon's output as untrusted input;
- normalize that output into the canonical refined package before emitting the
  Archivist-to-Sniper handoff;
- stop with an explicit error when normalization, schema, state, lock, or
  control-log validation fails;

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
  `evidence_coverage_ratio` when the Ranger artifact did not populate `evidence_cards[]`;
  include the Archivist's `model`, `effort` and `level_source` (null when unknown)
- produce the four-file refined package
- preserve `evidence_pack_path` from the Ranger analysis artifact when present; the four-file package shape does not change
- promote the Ranger analysis artifact from `pending/` into `<base_path>/refined/<mission_id>/analysis.md`.
  When the bound refinement weapon is `openspec-propose`, this promotion MUST be done by
  running `strategist mission normalize-openspec --mission-id <mission_id> --change-id
  <change_id>` against the completed OpenSpec change — never by hand-copying
  `proposal.md`/`design.md`/`tasks.md` and manually editing frontmatter. That command
  (`internal/refinement.NormalizeOpenSpec`) atomically publishes the four canonical files,
  injects `provider`/`provider_change_id`/`provider_runtime` and `mission_status:
  archivist_done` into the analysis frontmatter, and archives the completed change into
  `changes/archive/`. Bypassing it and promoting by hand is a documented drift source (see
  `.analysis/done/drift/` for the incident this codifies) — it silently loses the provider
  metadata and leaves the change unarchived.
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

### OpenSpec No-Spec-Delta Changes

Most `cmd/` adapter-migration and pure-refactor missions produce an OpenSpec
change with no capability/spec-level requirement changes. `openspec validate`
rejects a zero-delta change unless its `.openspec.yaml` declares
`skip_specs: true` — and setting that flag alone is not enough; the file also
needs valid `schema`/`created` metadata or the marker is silently not
honored. Use this minimal shape verbatim for that case:

```yaml
schema: spec-driven
created: <YYYY-MM-DD>
skip_specs: true
```

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
  (`refinement_done_no_tasks`)
- if `tasks.md` has tasks but none is a `documentation_target` (every item is
  `implementation_handoff`, `analysis_artifact` or `out_of_scope`), submit
  `refinement_done` and present the gate; on acceptance the mission resolves as
  `analysis_delivered` through `gate_approved_analysis_only` (see
  `05-approval-gate.md`). This is the same rule the gate contract states.

## Language

Write the four-file refined package in `active.language.docs`, independent of the language used
in the surrounding conversation.

## Status Transitions (Archivist)

- On start → update transient analysis frontmatter `mission_status: archivist_pending`
- On complete (all four files written, and transient pending artifact removed) → update promoted analysis frontmatter `mission_status: archivist_done`
