# Strategist — Contract 09: Response

## Goal

Make it explicit to the user that all progress is happening inside a governed mission.

## Final Envelope

Every Strategist response must close in this order:

1. progress / pipeline evidence
2. compliance summary
3. mission result

## Narrative Rules

- user-facing messages should identify the active mission whenever practical
- phase transitions should read as in-mission events, not generic chat replies
- the skill shell stays concise; details come from the ordered contracts

## Mission Result Minimum Fields

- `mission_id`
- `status`
- `artifacts`
- `next_action`

## Artifact Contract

- Ranger analysis: `<base_path>/refined/<mission_id>/analysis.md`
- Archivist package: `<base_path>/refined/<mission_id>/`
- Sniper report: `<base_path>/archived/<mission_id>-report.md`
- optional ADR: `<base_path>/archived/<mission_id>-adr.md`
