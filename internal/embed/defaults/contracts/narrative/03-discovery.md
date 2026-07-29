---
phase: discovery
slot: discovery
requires_approval: false
contract: write_analysis
---
# Strategist — Contract 03: Discovery

## Owner

Ranger (`discovery`)

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

Resolution of which concrete invocation target handles a given subtype (the
configured external weapon vs. the native `internal_skills/ranger` role) is
defined in `00-routing.md` § Discovery Weapon Resolution by Subtype — Ranger's
own behavior below is identical regardless of which mechanism invoked it.

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
   there (see `roles/ranger.yaml#canonical.search`)
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

- consult treasure chests before analysis
- follow the Retrieval Cascade above; do not skip stages out of order
- cite `evidence_pack_path` in the analysis artifact when the dossier provides one
- write exactly one canonical analysis artifact for the handoff
- include explicit sections for:
  - mission objective
  - known facts
  - uncertainties
  - affected scope
  - side quests
  - recommended refinement focus
- emit start, done, and opportunity events

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
