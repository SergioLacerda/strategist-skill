---
phase: discovery
slot: discovery
requires_approval: false
contract: write_analysis
---
# Strategist — Contract 03: Discovery

## Owner

Ranger (`discovery`)

Ranger is the native discovery Role and the configured discovery package is its
Weapon. The analysis handoff is an Artifact; `discovery_subtype` names the
kind of discovery being performed and does not change Role ownership.
`LEVELING` remains an immutable operational resolver consumed by INITIATIVE and
is outside this discovery boundary.

## Discovery Subtypes

Ranger receives `discovery_subtype` from Scout's `route_decision` (see
`00-routing.md` § Scout — Intake Router) whenever Scout selects `full_pipeline`
with `evidence_state: requires_discovery`. Scout selects the route; this vocabulary
describes Ranger's behavior after that selection.

| Subtype | Purpose | Expected output |
| --- | --- | --- |
| `creative` | shape new ideas, features, designs, or behavior changes | design options, assumptions, refinement focus |
| `evaluation` | decide whether a demand was implemented | verdict, evidence, gaps, residual recommendation |
| `diagnostic` | investigate a failure, mismatch, or blocked runtime | root-cause candidates, evidence, next check |
| `closure_evidence` | gather evidence for possible close/move to `done` | closure verdict, residuals, move recommendation |

The configured discovery weapon is required input to the fixed Ranger role.
Ranger must invoke it and treat its result as untrusted. Ranger's normalization,
checkpoint, lock, state, and handoff behavior below is
identical regardless of which weapon is selected.

The weapon result is untrusted. Ranger must normalize it into the pending
artifact, validate the required handoff schema and control metadata, and stop
with `role_invocation_failed` when invocation, compatibility, checkpoint, state,
lock, or log evidence is unavailable. Ranger never silently substitutes its
native behavior for the selected Weapon.

## Inputs

- original user prompt
- `mission_contract.planning_rules`
- `route_decision.discovery_subtype` (from Scout)
- context-enrichment dossier
- applicable treasure chests

## Outputs

- transient analysis handoff artifact: `<base_path>/pending/<mission_id>-analysis.md`
- handoff fields validated by `.strategist/schemas/handoff-ranger-to-archivist.schema.yaml`
- `discovery_subtype` (carried forward from `route_decision`)
- `evaluation_verdict` (`implemented` | `partially_implemented` | `not_implemented`),
  required when `discovery_subtype: evaluation`
- opportunity manifest summary when present
- `evidence_pack_path` when the context-enrichment dossier's `source_cards` are non-empty (see `machine/context-enrichment.yaml#evidence_pack`); null otherwise, non-blocking
- `relevant_sources_hint` produced by the Search ability during the Retrieval Cascade's
  treasure-chest stage; reused by Archivist by default (see `04-refinement.md`)
- `selected_runbooks_hint` produced by the select_runbook ability during the same
  Retrieval Cascade stage. Ranger always runs the command (stage 6 below), so the
  field is a list whenever discovery ran the cascade: an empty list means the command
  ran and nothing matched (non-blocking); null means it was not run, which is a gap
  to report in `uncertainties`. Reused by Archivist by default (see `04-refinement.md`),
  same reuse policy as `relevant_sources_hint`.

## Optional Handoff Challenge (Ranger → Archivist)

Archivist MAY apply a `ranger_to_archivist` Handoff Challenge
(`internal/handoff`, `TransitionRangerToArchivist`) against this handoff to
verify it correctly restated the artifact's content before refinement
proceeds. Its challenge-type vocabulary is fitted to this handoff's actual
content, not a reuse of the Archivist → Sniper MVP's `objective`/`gate`
types:

| Type | Validates |
| --- | --- |
| `recall` | Archivist can restate the critical `known_facts` entries by id |
| `boundary` | Archivist distinguishes `affected_scope` from `side_quests` — side quests are never treated as in-scope |
| `classification` | Archivist distinguishes a `known_facts` entry from an `uncertainties` entry |
| `verdict` | *(only when `discovery_subtype: evaluation`)* Archivist correctly restates `evaluation_verdict` |

This is advisory-first: no policy in this workspace currently sets
`RequiredTypes` for `ranger_to_archivist`, mirroring the MVP's own
"don't block low-risk or documentation-only transitions by default"
posture. Wiring a required-by-default risk policy for this transition is a
future decision, not made here — see
`.analysis/refined/20260803-handoff-challenge-extensions/design.md` § Item 1.

## Evaluation Discovery Procedure

When `discovery_subtype: evaluation`, Ranger must:

- read the target demand package and relevant implementation surfaces;
- compare requested scope with actual code/docs/tests/runtime behavior;
- run or report relevant validation where available;
- classify status as `implemented`, `partially_implemented`, or `not_implemented`
  and record it as `evaluation_verdict`;
- identify residual work;
- recommend whether Archivist should generate a refined residual package;
- avoid source mutation and avoid treating validation execution as implementation
  itself.

Evaluation discovery does not require design-option exploration, does not require
a `writing-plans` handoff, and does not require a design-doc commit as a
completion condition. Those are `creative`-subtype obligations only (see
`04-refinement.md` and Ranger's creative-subtype role directives).

## Weapon Profile for a Delegated Ranger

When Ranger runs as a delegated sub-role, it reaches the Weapon through the host skill loader: the
host's own copy of the skill, resolved by the host (for example from a user skills directory), not
an in-process embedded connector. No production code wires an embedded connector or an invoker for a
sub-role, so this channel is a defined degrade of the embedded one, not a claim of live embedded
invocation. It changes nothing else: the Weapon is still invoked, its output is still untrusted and
normalized by Ranger, and `native_substitution: forbidden` is unchanged. Nothing below substitutes
the Weapon; the profile only lists which Weapon steps do not apply to a delegated discovery run.
This contract does not certify the host copy: the roster and certification digests describe the
embedded package, not the file the host loader served. A delegated host therefore emits an
`strategist-invocation-receipt/v1` only after loading and invoking the selected Weapon. Ranger
rejects a missing, malformed, stale, replayed, or mission/role/Weapon-mismatched receipt with
`role_invocation_failed`. The receipt binds the mission, role, Weapon, relative resolved location,
resolved digest, issue time, and nonce. The nonce is retained in mission-scoped replay storage;
telemetry never includes it, the resolved location, prompts, outputs, secrets, or home paths.

| Keep | Drop |
| --- | --- |
| explore project context | the HARD-GATE user-approval step |
| decompose into units | one-question-at-a-time dialogue |
| alternatives with trade-offs and a recommendation | the second design document (the only artifact is `<base_path>/pending/<mission_id>-analysis.md`) |
| YAGNI | the `git commit` of a design file |
| | the spec-review gate and the `writing-plans` transition |

`weapon_invocation` is required for a delegated run and optional otherwise. Record it in the handoff
(`schemas/handoff-ranger-to-archivist.schema.yaml`): `invoked`, `resolved_from` (where the Weapon was
actually resolved, e.g. the embedded runtime or the host skill directory), `steps_dropped`, and
`resolved_digest`, the `sha256:<64 hex>` of the raw bytes of the file the loader served. This is the
home of `invocation_evidence: required`; it does not change which Weapon is bound. When the active
runtime catalog (`.strategist/plugins/catalog.yaml`) supplies an `upstream_content_digest`, Ranger
compares it to the receipt digest and blocks a mismatch. An absent pin is explicitly
`pin_unavailable`, not a verification claim.

A receipt authenticates that the host invoked a Weapon. It does **not** prove that the parent agent
was prevented from independently reading, reasoning, or using its own tools. Hosts must report
`capability_isolation: unverified` unless separate conformance evidence establishes scoped delegated
capabilities; this repository currently makes no structural-prevention claim for Codex or Claude.

## Retrieval Cascade

Ranger's source retrieval follows this order, normative rather than heuristic. Each
stage runs only if the previous stage did not reach `stop_when: sufficient_evidence`:

1. explicit paths named in the prompt or mission contract
2. registry / manifest lookups (`index.yaml`, `active.yaml`, treasure-chests manifest)
3. keyword search over the workspace
4. symbol search (definitions, references)
5. architecture / structure index, when one exists for the workspace
6. treasure chests (`consult_treasure_chests`) — the **Search** ability runs as a
   sub-routine of this stage, before `consult_treasure_chests` opens any chest: it
   filters candidate jewels/potions and produces `relevant_sources_hint`, so a whole
   chest is not paid for when a jewel/potion already summarizes what would be found
   there (see `roles/ranger.yaml#canonical.search`). The **select_runbook** ability
   runs alongside Search, at the same point: it scores `docs/runbooks/*.runbook.yaml`
   sidecars against mission signals via `internal/runbook.Select()` and produces
   `selected_runbooks_hint` — a bounded, reasoned selection (at most one primary, at
   most two supporting runbooks, each with a non-empty match reason) distinct from
   Search's own unstructured jewel/potion relevance matching (see
   `roles/ranger.yaml#canonical.select_runbook`). **Ranger MUST run `strategist
   runbook select --format json --signal <signal> ...` at this point**, with one
   `--signal` per mission signal (the mission's `task_type`, the Scout
   `discovery_subtype`, and the salient keywords of the request), and record the
   command, the signals and its result in the analysis artifact. No sidecar or no
   match is a valid, non-blocking result (an empty list); skipping the command is
   not, because a runbook that applies would silently never reach the mission
7. semantic search, when a semantic provider is configured — optional, last resort

`stop_when: sufficient_evidence` is met when either condition holds:

- every open Discovery Question has at least one Evidence Card with `confidence >= 0.6`, or
- all seven cascade stages have been attempted (evidence gaps are then reported as
  `uncertainties`, not left implicit)

This cascade governs order only; per-role token ceilings remain governed by
`skill.yaml#role_budgets`.

## Intra-Phase Parallelism

Independent Discovery sub-tasks (e.g. reading unrelated source files, running
independent read-only checks) may run concurrently within this phase. This is
distinct from cross-phase concurrency: Scout and Archivist must never run
simultaneously (decision conflict) — see `00-routing.md`. `skill.yaml#budget_policy`'s
`timeout_seconds` applies to the phase total, not to the sum of sequential sub-tasks.

## Required Behavior

- before finishing, persist this boundary's confidence: `strategist metrics record --mission <mission_id> --agent ranger --claim-file -` (claim piped on stdin, never a file under `<base_path>`; see `machine/confidence-governance.yaml#producers.claim_placement`),
  or, when no confidence summary was produced, `strategist metrics record --mission <mission_id> --agent ranger --missing --correlation-key <key> --reason <why>`;
  never finish silently (see `machine/confidence-governance.yaml#producers`)
- consult treasure chests before analysis
- follow the Retrieval Cascade above; do not skip stages out of order
- cite `evidence_pack_path` in the analysis artifact when the dossier provides one
- write exactly one canonical analysis artifact for the handoff
- always declare `sources_consulted[]` in the frontmatter (an empty list when discovery opened no source); list every opened source in it (`source_path`,
  `content_fingerprint`, `coverage_status`). For an `evaluation` or `diagnostic` subtype, also
  cite the line ranges the Archivist must quote, so the refinement can quote them without
  reopening the file (see `machine/handoff-contract.yaml#refinement_context_policy`). Without
  the list that policy is inert and every Archivist read is unaccounted
- include explicit sections for:
  - mission objective
  - known facts
  - uncertainties
  - affected scope
  - side quests
  - recommended refinement focus
- emit start, done, and opportunity events

### Evidence That Rests on Runtime State

Refined packages outlive the runtime that produced them: `.strategist/` is generated, gitignored
and replaced by an upgrade or a reinstall. A finding that rests on runtime state (anything under
`.strategist/`: memory logs, missions, generated files) is therefore quoted in the analysis
artifact, not only pointed to: the value or lines relied on (`excerpt`, at most 1200 bytes, with
`excerpt_anchor`) and the moment they were read (`captured_at`). Use the `evidence:` fields of
`schemas/evidence.schema.yaml` when evidence entries are recorded, or quote the value inline in the
`known_facts` line otherwise. The path stays as a pointer for whoever still has the source, never as
the only evidence. Files tracked by git need only their path and anchor (and `commit` when it
matters).

### Optional Evidence Recording

Ranger MAY record individual findings as `evidence:` entries
(`schemas/evidence.schema.yaml`) in addition to the required `known_facts`
prose, when a finding's source, classification (`explicit` /
`corroborated_inference` / `weak_inference` / `unknown`), and confidence are
worth tracking individually — e.g., missions later evaluated for
mission_quality (`machine/mission-quality.yaml`), or findings likely to be
cited by name in a later mission. This is optional, not required: forcing it
on every mission would add ceremony without proportional value, the same
restraint already applied to Pathfinder/Anamnese/Quiz activation.

## Write Scope

- authorized path: `<base_path>/pending/<mission_id>-analysis.md`
- type: `.md`

## Language

Write the analysis artifact in `active.language.docs`, independent of the language used in the
surrounding conversation.

## Mission Status Protocol

Every analysis artifact MUST begin with YAML frontmatter:

```yaml
---
mission_id: <mission_id>
mission_status: ranger_pending
date: <YYYY-MM-DD>
---
```

### Pre-creation checklist

Before writing `<base_path>/pending/<mission_id>-analysis.md`, Ranger MUST:

| Condition | Action |
|-----------|--------|
| File does not exist | Create with `mission_status: ranger_pending` → proceed |
| Exists + status `ranger_pending` or `archivist_pending` or `sniper_running` | `blocked reason=mission_in_progress` → STOP |
| Exists + status `ranger_done` | Skip Ranger, resume from Archivist |
| Exists + status `archivist_done` or `gate_pending` | Skip to gate re-presentation |
| Exists + status `gate_analysis_accepted` | Skip to Sniper claim |
| Exists + status `gate_revision_requested` | Resume from Archivist revision |
| Exists + status `gate_rejected` | Do not reprocess; report status |
| Exists + status `documentation_applied` | Emit warning, do not reprocess |

### Status transitions (Ranger)

- On create → `ranger_pending`
- On analysis complete → update frontmatter to `ranger_done`

## Notes

- `pending/` is the canonical transient location for Ranger output during discovery
- Archivist reads this transient artifact, then promotes it to `<base_path>/refined/<mission_id>/analysis.md`
- this transient artifact is the authoritative input for Archivist
