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

- before finishing, persist this boundary's confidence. The OpenSpec path does this
  inside `strategist mission normalize-openspec`: it records Archivist's explicit
  canonical-artifact claim before publication and aborts without publishing when the
  record cannot be written. Other refinement paths use `strategist metrics record --mission
  <mission_id> --agent archivist --claim-file -` (see
  `machine/confidence-governance.yaml#producers.claim_placement`), or `--missing
  --correlation-key <key> --reason <why>` when no summary exists. Never finish silently —
  Archivist also records the critic (`--agent response_critic`) and `mission_quality`
  boundaries with the same command (see `machine/confidence-governance.yaml#producers`)
- before invoking the selected refinement weapon's own CLI/tooling, apply
  `roles/archivist.yaml#canonical.resolve_weapon_scratch_root` — read
  `plugins/catalog.yaml#providers[id=<provider>].runtime`. For Ranked
  `openspec_root`, run with the declared `.strategist/openspec` root as its
  working directory; `.strategist/weapon-runtime/<provider_id>/` supplies the
  bundled launcher only and is never the project root. Never use the host
  repository root (see `agent-protocol.md` §3 Refinement Routing)
- treat the selected refinement weapon's output as untrusted input;
- normalize that output into the canonical refined package before emitting the
  Archivist-to-Sniper handoff;
- stop with an explicit error when normalization, schema, state, lock, or
  control-log validation fails;

- treat the Ranger transient analysis artifact as the canonical refinement input
- reuse the Ranger artifact's `relevant_sources_hint` (Search Feat output) and
  `selected_runbooks_hint` (select_runbook Feat output) by default instead of
  re-running Search or select_runbook; only re-run either with a declared reason
  from `contracts/machine/handoff-contract.yaml#refinement_context_policy.allowed_reasons`
  (see `roles/archivist.yaml#canonical.reuse_search_cache`)
- consult treasure chests before refinement
- before reopening any source listed in the Ranger artifact's `sources_consulted[]`,
  check `contracts/machine/handoff-contract.yaml#refinement_context_policy` — reopen
  only for one of its `allowed_reasons`, and state the matching reason explicitly in
  the refined artifact that needed it (see skill.yaml's
  `archivist_reopens_discovery_sources_without_declared_reason` forbidden_behaviors entry).
  A verification read of a `coverage_status: full` source is a reopen too: it is allowed only
  as `stale_evidence_check` (the file changed after the Ranger artifact and the read stays
  inside the line ranges Ranger cited). List every reopen in the refined artifact as a table
  (`source_path`, `reason`, line range) — an empty table means zero reopens
- pass `--reopens N` explicitly to `strategist metrics handoff-record`, where `N` is the number of
  rows in that table (`0` when the table is empty). An unset `--reopens` records `null`, meaning
  "not measured"; only an explicit `0` means "measured, none"
- on a gate revision (`gate_revision_requested`), re-run the role's `on_start` leveling label for the
  revised run and record the revised package with
  `strategist metrics handoff-record --mission <mission_id> --revision <n>` (`n` starts at 1) so
  each revision keeps its own metrics line and level row; the base line is left as recorded
- when the invoking shell reports the token usage of the discovery run (the sub-role's own usage
  summary, run at the gate and at DONE with `strategist mission report-usage`), pass it as
  `--discovery-tokens`; omit the flag when nothing was reported. These counts are self-reported by
  the invoking agent, so treat them as a weak signal, never as evidence of cost
- on completion, append one line to `.strategist/memory/handoff-metrics.jsonl`
  (skill.yaml#handoff_metrics_log) with `strategist metrics handoff-record --mission <mission_id>`
  (pass only the values that were measured; it never derives the two ratios, and a mission that
  already has a line is left unchanged) — nulls are expected for `brief_compression_ratio`/
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
- amend a package that is already published only through
  `strategist mission normalize-openspec --mission-id <mission_id> --change-id <new_change>
  --amend --amends <previous_change_id> --authorization-ref "<quote or gate event>"` — never by hand.
  The mode replaces `proposal.md`, `design.md` and `tasks.md` with the new change, leaves
  `analysis.md`, `mission_status` and the original `provider_change_id` untouched, records an
  `amendments:` list in the replaced files' frontmatter, snapshots the previous files under
  `<package>/.amendments/NNN/`, and refuses a claimed or applied package, a rejected mission, a
  pending analysis, and an analysis-only accepted package that would gain a documentation target.
  The authorization reference is a human's words or gate event recorded verbatim; the command
  cannot verify it. The default mode is unchanged and still fails closed on a differing package.
- keep Ranger's quoted evidence when it rests on runtime state (`excerpt`, `captured_at`; see
  `03-discovery.md` § Evidence That Rests on Runtime State), and quote any runtime state the
  refined files cite themselves (a value with its capture time) rather than pointing only at a
  `.strategist/` path; the refined package must stay readable after the runtime is replaced
- classify side quests and surface them at the approval gate
- classify every `tasks.md` / `implementation_plan` item by `task_type`: `documentation_target`,
  `analysis_artifact`, `implementation_handoff`, or `out_of_scope` (see
  `handoff-archivist-to-sniper.schema.yaml`). Only `documentation_target` items are
  Sniper-executable. `implementation_handoff` items (code, hook, config, or test mutation)
  must never be phrased as executable Sniper tasks — they are handed off, not queued
  for materialization.
- declare the typed `handoff_policy_facts` block in the frontmatter of `analysis.md`
  (`handoff-archivist-to-sniper.schema.yaml#handoff_policy_facts`): `mandatory_constraints`,
  `unresolved_questions` and `forbidden_scope` as lists, `destructive_operation_possible`,
  `security_sensitive_task` and `informational_only` as booleans. Every field is required and
  nothing is inferred from prose; `informational_only: true` is rejected when any require fact
  holds. Publish it with `strategist mission normalize-openspec --handoff-facts <file>` (a YAML
  mapping of those fields); without the flag the command warns, and a package without the block
  has no evaluable handoff policy and cannot enter execution. An amendment keeps `analysis.md`
  byte-identical, so the facts are declared at publication, not by `--amend`.
- the Archivist -> Sniper policy (`contracts/machine/handoff-contract.yaml#handoff_verification_policy`)
  is derived from the package, never chosen by the caller: after the Approval Gate is accepted
  run `strategist handoff evaluate --mission-id <id>` (adding `--challenges` and `--ack` when the
  package requires the challenge, with `objective`, `boundary`, `classification` and `gate`
  challenge types). It records a durable passed, failed or policy-authorized skipped outcome.
  The command also records the handoff confidence (`--confidence-summary`, or an explicit
  missing-record) and the failure loop: a failed outcome returns the mission to refinement,
  so a repaired package needs a new Approval Gate acceptance, and the last allowed failure
  blocks the mission. A passed or skipped outcome does not enter execution by itself.
  This semantic acknowledgment complements the YAML structure contract; it never replaces
  Approval Gate review.
- the lifecycle-owned Ranger -> Archivist Handoff Challenge is evaluated from
  the normalized artifact's typed `ranger_handoff_policy_facts` block before
  this provider is invoked (see `03-discovery.md` § Conditional Handoff
  Challenge and `contracts/machine/handoff-contract.yaml#archivist_entry_policy`).
  Archivist must not treat the standalone `handoff verify` diagnostic as
  authorization, and an absent or invalid facts block is a hard denial.
- when the mission type is evaluation or audit and the Ranger discovers completed work
  requiring cleanup (archiving finished missions, removing obsolete files): treat that
  cleanup as an opportunity attack, not a main task. The full pipeline resolves as
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
