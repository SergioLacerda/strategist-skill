# Strategist v2 — Design Spec
**Date**: 2026-05-27
**Revision**: v2 — standalone distribution, dual modes, knowledge index, learning cache

---

## Summary

Strategist is a standalone orchestration skill distributed via curl. It coordinates multi-phase missions through three pluggable slots (Scout → Engineer → Hunter) with two operational modes (pragmatic/epic), an external multi-source knowledge index consulted before each prompt, and an approval-gated two-file learning cache.

This document captures the v2 design additions on top of the v1 foundation documented in `.analysis/plans/strategist-mission-pipeline/`.

---

## Architecture

```
SKILL ROOT (~/.claude/skills/strategist/)
├── install.sh               ← curl target; --wizard flag for TUI
├── skill.yaml
├── SKILL.md                 ← agent instructions
├── active.yaml              ← generated at install (current config)
├── templates/               ← pre-built install configs
│   ├── pragmatic-standalone.yaml
│   ├── epic-standalone.yaml
│   └── epic-sdd.yaml
├── personas/
│   ├── pragmatic.yaml       ← tone + phase labels
│   └── epic.yaml
├── roles/
│   ├── sdd-mission.yaml
│   └── spec-driven.yaml
├── knowledge.index.yaml     ← multi-source knowledge registry
├── memory/
│   ├── outcomes.jsonl       ← approved mission records
│   └── source-hints.yaml   ← learned source preferences
└── schemas/
    ├── intake.schema.yaml
    └── progress-contract.yaml

TARGET REPO (~/<project>/)   ← zero config files here
└── <base_path>/             ← default: .analysis/
    ├── todo/
    ├── pending/
    ├── refined/
    ├── done/
    └── .strategist/         ← workspace domain (NOT config)
        ├── index.yaml
        ├── identity/
        ├── directives/
        ├── rubrics/
        └── patterns/
```

---

## Distribution

```bash
# Silent (defaults: pragmatic-standalone template, base_path=.analysis):
curl -sL strategist.run | sh

# Interactive TUI:
curl -sL strategist.run | sh --wizard
```

The TUI wizard offers:
1. Template selection (pragmatic-standalone / epic-standalone / epic-sdd / custom)
2. Base directory for missions (default: `.analysis`)
3. Scout provider (default: `sdd-diagnose`)
4. Engineer provider (default: `sdd-engineer`)
5. Hunter provider (required, no default)
6. Knowledge sources (zero or more paths; add later via `strategist knowledge add`)

Result: `active.yaml` written to skill root. No files written to target repo at install time.

---

## Two Operational Modes

Same pipeline. Different persona. Configured in `active.yaml`, overrideable with `--mode`.

| Dimension       | Pragmatic                       | Epic                            |
|----------------|----------------------------------|---------------------------------|
| Phase labels   | analysis / refinement / execution| scout / engineer / hunter       |
| Tone           | Direct, analytical               | Strategic, narrative            |
| Opening        | "What is the problem?"           | "What is the mission?"          |
| Progress       | `[Strategist] phase=analysis`    | `[Strategist] phase=scout`      |
| Completion     | "Plan ready for review."         | "Briefing ready. Awaiting approval." |

### Persona file contract

```yaml
# personas/epic.yaml
phase_labels:
  discovery: scout
  refinement: engineer
  execution: hunter
tone_directive: >
  You are a strategic commander directing specialist operatives.
  Speak with precision and urgency. Name phases by operative role.
prompt_templates:
  mission_start: "What is the mission?"
  approval_gate: "Engineer has submitted the briefing. Proceed to Hunter?"
progress_format: "[Strategist] phase={label} status={status}"
```

---

## External Knowledge Index

`knowledge.index.yaml` in skill root. Queried by `context-enrichment` before each mission prompt.

```yaml
# knowledge.index.yaml
schema_version: "1.0"
sources:
  - id: project-architecture
    type: docs
    path: /abs/path/to/docs/architecture
    tags: [architecture, system-design, adr]
    priority: high
  - id: coding-standards
    type: docs
    path: /abs/path/to/docs/standards
    tags: [implementation, coding, style]
    priority: medium
  - id: past-examples
    type: examples
    path: .strategist/patterns/good
    tags: [examples, patterns, reference]
    priority: medium
```

### Query flow

```
mission received
      │
      ▼
prompt-intake classifies task_type
      │
      ▼
context-enrichment:
  1. load knowledge.index.yaml
  2. filter sources by tags ∩ task_type
  3. apply source-hints.yaml priority overrides
  4. load excerpts in priority order (respect token budget)
  5. return ranked excerpts + rubric for task_type
      │
      ▼
dossier-builder:
  + excerpts from external sources
  + .strategist/ identity + directive for task_type
  + memory/lessons.yaml entries for task_type
  → minimal dossier handed to mission before prompt
```

---

## Learning Cache

Two files. Both require explicit human approval before writing.

```
POST-MISSION (after approval gate and Hunter execution):

response-critic evaluates the mission
      │
      ▼
learning-curator proposes:

  File 1: memory/outcomes.jsonl (append-only)
  ───────────────────────────────────────────
  {
    "mission_id": "mission-2026-05-27-001",
    "task_type": "architecture_analysis",
    "what_worked": "...",
    "what_to_avoid": "...",
    "sources_used": ["project-architecture"],
    "mode": "epic",
    "approved_at": "..."
  }

  File 2: memory/source-hints.yaml (annotation overlay)
  ───────────────────────────────────────────────────────
  - source_id: project-architecture
    annotation: "high signal for architecture_analysis"
    priority_adjustment: high → critical
    derived_from: mission-2026-05-27-001

learning-curator presents both for SEPARATE review:
  "🧠 Record this mission outcome? [Y/n]"
  "📚 Apply source hint adjustment? [Y/n]"
      │
    ┌─┴──────┐
    │        │
  Each    approved independently
  file    persisted or discarded separately
```

---

## Mission Flow (complete)

```
[Preflight]
  ├── load active.yaml → persona + roles + base_path
  ├── load personas/<mode>.yaml → phase labels + tone
  ├── load .strategist/index.yaml → identity + load_always files
  └── validate Scout/Engineer/Hunter providers

[Intake]
  └── extract delivery_strategy, task_type, execution_intent

[Context Enrichment] (non-blocking, parallel)
  ├── query knowledge.index.yaml by task_type tags
  ├── apply source-hints.yaml priority overlay
  └── build minimal dossier → agent receives before prompt

[Scout / Analysis phase]
  └── artifact written to <base_path>/pending/

[Engineer / Refinement phase]
  └── artifact written to <base_path>/refined/

[APPROVAL GATE]  ← mandatory stop, user approves
  └── present refined plan path

[Hunter / Execution phase]
  └── artifact written to <base_path>/done/

[Learning Phase] (post-execution)
  ├── response-critic evaluates
  ├── learning-curator proposes outcomes + source-hints
  └── awaits independent human approval for each file
```

---

## Key Invariants

- **Zero target repo config**: All configuration (`active.yaml`, `personas/`, `roles/`, `memory/`, `knowledge.index.yaml`) lives in skill root. Only workspace artifacts (`<base_path>/`) go in target repo.
- **Mode is voice, not pipeline**: Both modes run identical pipeline logic. Persona changes only labels, tone, and prompt templates.
- **Knowledge index is additive**: SDD-injected `knowledge_paths` append to, never replace, the skill's own `knowledge.index.yaml`.
- **Learning requires human**: `learning-curator` is forbidden from writing either cache file without explicit user confirmation per file.
- **Approval gate is absolute**: Hunter slot is forbidden from executing without user approval after Engineer completes.

---

## Open Questions

None. All decisions resolved in brainstorming session 2026-05-27.
