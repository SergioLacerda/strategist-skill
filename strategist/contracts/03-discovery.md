# Strategist — Contract 03: Discovery

## Owner

Ranger (`discovery`)

## Inputs

- original user prompt
- `mission_contract.planning_rules`
- context-enrichment dossier
- applicable treasure chests

## Outputs

- analysis handoff artifact: `<base_path>/refined/<mission_id>-analysis.md`
- handoff fields validated by `strategist/schemas/handoff-ranger-to-archivist.schema.yaml`
- opportunity manifest summary when present

## Required Behavior

- consult treasure chests before analysis
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

- authorized path: `<base_path>/refined/<mission_id>-analysis.md`
- type: `.md`

## Notes

- `pending/` may still exist for legacy artifacts, but the canonical Ranger output for the main mission is the analysis handoff in `refined/`
- this artifact is the authoritative input for Archivist
