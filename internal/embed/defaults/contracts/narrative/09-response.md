---
phase: response
slot: null
requires_approval: false
contract: null
---
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

## Chat Language

`active.language.chat` binds two independent things, both required:

1. Which language variant is read for named `content_by_lang` and
   `phase_announcements` templates (mechanism: `strategist check
   --print-content-by-lang <lang> --persona <mode>`, per `01-bootstrap.md`).
2. The language of all **Strategist-mediated conversational prose** — mission
   narration, phase updates, analysis/diagnostic write-ups, gate framing, and
   any other free text the parent agent writes while a Strategist mission is
   active. This is not limited to the named templates in (1); it covers
   everything the agent says in the conversation on behalf of a Strategist
   mission.

Scope of (2) is intentionally narrow: it binds output produced while
Strategist is actively running a mission, not all assistant output for the
rest of the session regardless of relevance. This matches Strategist's Scope
Invariant (`00-routing.md`) — it does not claim authority over conversation
unrelated to any Strategist mission.

`active.language.docs` is independent of `active.language.chat` and is
unaffected by this rule — written artifacts (analysis, proposal, design,
tasks, ADRs) continue to resolve their language from `active.language.docs`
only, per `03-discovery.md`, `04-refinement.md`, and `07-adr.md`.

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
- Critical Hit closure completion report: `<base_path>/done/<id>/completion-report.md` (see `11-critical-hit.md`)

## Console Policy Enforcement

Before emitting any user-facing text, the active persona's `console_policy` must be loaded.

Rules:
- `show_raw_events: true` (epic mode) → emit raw `[Strategist] key=value` progress lines,
  each preceded by the corresponding `phase_announcements[lang][event_key]` wrapper line.
  `content_by_lang` templates remain in use for events without a raw log equivalent
  (approval prompts, mission complete, etc.)
- `show_mission_envelope: true` → wrap mission start with `mission_envelope.open` and mission end with `mission_envelope.close`
- `emit_jsonl_telemetry: true` → bypass `content_by_lang`; emit all events as JSONL

Violation detection: if a raw `[Strategist] \w+=\S+` event appears in epic output WITHOUT
a preceding `phase_announcements` wrapper line, this is `forbidden_behavior #9`. Self-correct
by re-emitting with the announcement prefix.

## Input/Output Contract — `mission_envelope.close`

**Inputs required:**

| Field | Type | Description |
|-------|------|-------------|
| `status_label` | string | e.g., `"MISSION COMPLETE"` (localized via i18n bundle) |
| `phase_timeline` | string | Lines built from `phase_timeline_entry` template, one per executed phase |
| `artifact_block` | string | Lines built from `artifact_entry` template, one per produced artifact |
| `conclusion_text` | string | Final sentence with `{mission_id}` and `{next_action}` |

**`phase_timeline_entry`:** `"  {icon} {phase_label} → {result_label}"`

**`artifact_entry`:** `"  📁 {key}: {path}"`

**Output:** the box rendered with all sections filled — no raw fields outside the box.

## Mission Close Sequence (profile=epic)

Emit in this order:

0. If `sq_backlog` is non-empty, present the Riposte capture offer
   (`machine/riposte.yaml`, trigger `mission_close_sq_backlog`) — declining never
   blocks the close
1. `content_by_lang.*.response_complete` — 1-line compliance summary
2. `content_by_lang.*.mission_complete` — renders `mission_envelope.close`

## Output Examples

**✅ COMPLIANT:**
```
⚖️ **Compliance [20260620-cicd-landing]:** pipeline_compliant=yes | phases=5

╔══════════════════════════════════════════════════════════╗
║  STRATEGIST SKILL · MISSION COMPLETE                     ║
╠══════════════════════════════════════════════════════════╣
  ✅ Ranger      → reconnaissance complete
  ✅ Archivist   → analysis refined
  ✅ Gate        → review accepted
  ✅ Sniper      → documentation materialization complete
╠══════════════════════════════════════════════════════════╣
  📁 discovery:  <base_path>/refined/ID/analysis.md
  📁 refined:    <base_path>/refined/ID/
  📁 report:     <base_path>/archived/ID-report.md
╚══════════════════════════════════════════════════════════╝
🎯 Mission ID complete. Next action: push to main.
```

**❌ VIOLATION (forbidden_behavior #9):**
```
[Strategist] phase=sniper status=done          ← raw event without phase_announcements wrapper

[Strategist] response_complete                 ← ad-hoc format does not exist
  pipeline_compliant: yes
  phases_run: preflight, intake, ...

mission_id: ID                                 ← loose YAML fields outside the envelope
status: completed
```

**✅ COMPLIANT (epic raw event with wrapper):**
```
🗡️ **Sniper:** Target confirmed. Silence — materializing documentation.
[Strategist] phase=materialization status=starting
```
