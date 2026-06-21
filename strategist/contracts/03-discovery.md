# Strategist — Contract 03: Discovery

## Owner

Ranger (`discovery`)

## Inputs

- original user prompt
- `mission_contract.planning_rules`
- context-enrichment dossier
- applicable treasure chests

## Outputs

- transient analysis handoff artifact: `<base_path>/pending/<mission_id>-analysis.md`
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

- authorized path: `<base_path>/pending/<mission_id>-analysis.md`
- type: `.md`

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
| Exists + status `gate_approval` | Skip to Sniper claim |
| Exists + status `execution_done` | Emit warning, do not reprocess |

### Status transitions (Ranger)

- On create → `ranger_pending`
- On analysis complete → update frontmatter to `ranger_done`

## Notes

- `pending/` is the canonical transient location for Ranger output during discovery
- Archivist reads this transient artifact, then promotes it to `<base_path>/refined/<mission_id>/analysis.md`
- this transient artifact is the authoritative input for Archivist
