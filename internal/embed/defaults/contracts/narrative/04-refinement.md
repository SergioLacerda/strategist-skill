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

## Outputs

- `<base_path>/refined/<mission_id>/analysis.md`
- `<base_path>/refined/<mission_id>/proposal.md`
- `<base_path>/refined/<mission_id>/design.md`
- `<base_path>/refined/<mission_id>/tasks.md`
- execution handoff fields validated by `.strategist/schemas/handoff-archivist-to-sniper.schema.yaml`

## Required Behavior

- treat the Ranger transient analysis artifact as the canonical refinement input
- consult treasure chests before refinement
- produce the four-file refined package
- promote the Ranger analysis artifact from `pending/` into `<base_path>/refined/<mission_id>/analysis.md`
- classify side quests and surface them at the approval gate
- when the mission type is evaluation or audit and the Ranger discovers completed work
  requiring cleanup (archiving finished missions, removing obsolete files): treat that
  cleanup as an opportunity attack, not a main task. The main mission resolves as
  `analysis_delivered`. The cleanup is offered via `opportunity_gate` manifest.
- never emit a single-file refined artifact as the canonical result

## Write Scope

- authorized paths:
  - `<base_path>/refined/<mission_id>/proposal.md`
  - `<base_path>/refined/<mission_id>/analysis.md`
  - `<base_path>/refined/<mission_id>/design.md`
  - `<base_path>/refined/<mission_id>/tasks.md`

## Gate Condition

- if `tasks.md` is empty or absent, mission resolves as `analysis_delivered`

## Status Transitions (Archivist)

- On start → update transient analysis frontmatter `mission_status: archivist_pending`
- On complete (all four files written, and transient pending artifact removed) → update promoted analysis frontmatter `mission_status: archivist_done`
