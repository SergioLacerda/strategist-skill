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
| `status_label` | string | e.g., `"MISSÃO CONCLUÍDA"` / `"MISSION COMPLETE"` |
| `phase_timeline` | string | Lines built from `phase_timeline_entry` template, one per executed phase |
| `artifact_block` | string | Lines built from `artifact_entry` template, one per produced artifact |
| `conclusion_text` | string | Final sentence with `{mission_id}` and `{next_action}` |

**`phase_timeline_entry`:** `"  {icon} {phase_label} → {result_label}"`

**`artifact_entry`:** `"  📁 {key}: {path}"`

**Output:** the box rendered with all sections filled — no raw fields outside the box.

## Mission Close Sequence (profile=epic)

Emit in this order:

1. `content_by_lang.*.response_complete` — 1-line compliance summary
2. `content_by_lang.*.mission_complete` — renders `mission_envelope.close`

## Output Examples

**✅ CONFORME:**
```
⚖️ **Compliance [20260620-cicd-landing]:** pipeline_compliant=yes | fases=5

╔══════════════════════════════════════════════════════════╗
║  STRATEGIST SKILL · MISSÃO CONCLUÍDA                   ║
╠══════════════════════════════════════════════════════════╣
  ✅ Ranger      → reconhecimento concluído
  ✅ Arquivista  → análise refinada
  ✅ Gate        → aprovação concedida
  ✅ Sniper      → implementação concluída
╠══════════════════════════════════════════════════════════╣
  📁 discovery:  <base_path>/refined/ID/analysis.md
  📁 refined:    <base_path>/refined/ID/
  📁 report:     <base_path>/archived/ID-report.md
╚══════════════════════════════════════════════════════════╝
🎯 Missão ID concluída. Próxima ação: push para main.
```

**❌ VIOLAÇÃO (forbidden_behavior #9):**
```
[Strategist] phase=sniper status=done          ← raw event sem phase_announcements wrapper

[Strategist] response_complete                 ← padrão ad-hoc não existe
  pipeline_compliant: yes
  phases_run: preflight, intake, ...

mission_id: ID                                 ← campos YAML soltos fora do envelope
status: completed
```

**✅ CONFORME (epic raw event com wrapper):**
```
🗡️ **Sniper:** Alvo confirmado. Silêncio — executando.
[Strategist] phase=execution status=starting
```
