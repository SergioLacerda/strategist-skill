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

- Ranger analysis: `<base_path>/refined/<mission_id>-analysis.md`
- Archivist package: `<base_path>/refined/<mission_id>/`
- Sniper report: `<base_path>/archived/<mission_id>-report.md`
- optional ADR: `<base_path>/archived/<mission_id>-adr.md`

## Console Policy Enforcement

Before emitting any user-facing text, the active persona's `console_policy` must be loaded.

Rules:
- `show_raw_events: false` → use `content_by_lang` or `mission_envelope` exclusively; never emit `[Strategist] key=value` lines to the user console
- `show_mission_envelope: true` → wrap mission start with `mission_envelope.open` and mission end with `mission_envelope.close`
- `emit_jsonl_telemetry: true` → bypass `content_by_lang`; emit all events as JSONL

Violation detection: if a raw `[Strategist] \w+=\S+` pattern appears in user-facing output while `profile=epic`, this is `forbidden_behavior #9`. Self-correct by re-emitting the event via the appropriate `content_by_lang` template.
