# Strategist — Contract 04: Refinement

## Owner

Archivist (`refinement`)

## Inputs

- analysis handoff artifact: `<base_path>/refined/<mission_id>-analysis.md`
- `mission_contract.planning_rules`
- context dossier
- applicable treasure chests

## Outputs

- `<base_path>/refined/<mission_id>/proposal.md`
- `<base_path>/refined/<mission_id>/design.md`
- `<base_path>/refined/<mission_id>/tasks.md`
- execution handoff fields validated by `strategist/schemas/handoff-archivist-to-hunter.schema.yaml`

## Required Behavior

- treat the Ranger analysis artifact as the canonical refinement input
- consult treasure chests before refinement
- produce the three-file refined package
- classify side quests and surface them at the approval gate
- never emit a single-file refined artifact as the canonical result

## Write Scope

- authorized paths:
  - `<base_path>/refined/<mission_id>/proposal.md`
  - `<base_path>/refined/<mission_id>/design.md`
  - `<base_path>/refined/<mission_id>/tasks.md`

## Gate Condition

- if `tasks.md` is empty or absent, mission resolves as `plan_only`
