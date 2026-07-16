# Strategist Skill — Detailed Technical Documentation

> Current behavior note (2026-06-26): Strategist now operates as a
> documentation, diagram, and analysis orchestrator. Sniper remains the executor
> in the lore, but execution means approved documentation/handoff materialization,
> not source-code implementation. This detailed onboarding file is being
> consolidated; when it conflicts with `docs/mental-model.md`,
> `docs/configuration.md`, or `.strategist/agent-protocol.md`, those sources win.

## Cognitive Runtime, Self-Learning and AI Agent Convergence

<p align="center">
<a href="readme-detailed.md">🇧🇷 Português</a> | 🇺🇸 English
</p>

---

## How To Read This Document

This detailed README is structured for two audiences:

- **Quick pass (5-10 min)**: `Overview` → `Mission Pipeline` → `Stop Conditions` → `Forbidden Behaviors`.
- **Implementation/Operations**: follow the full order from `Installation` → `File Structure` → `Technical Flow` → `Slot Configuration` → `SDD Integration`.

### Quick Index

- [Overview](#overview)
- [Problem](#problem)
- [Installation](#installation)
- [File Structure](#file-structure)
- [Mission Pipeline](#mission-pipeline)
- [Internal Technical Flow](#internal-technical-flow)
- [Operation Modes (Personas)](#operation-modes-personas)
- [Knowledge System](#knowledge-system)
- [Slot Configuration (roles)](#slot-configuration-roles)
- [SDD Integration (Optional)](#sdd-integration-optional)
- [Stop Conditions](#stop-conditions)
- [Forbidden Behaviors](#forbidden-behaviors)
- [Drift Self-Correction](#drift-self-correction)
- [Architectural Decisions](#architectural-decisions)
- [Progress Flow](#progress-flow)

### Executive Summary

- Multi-phase orchestrator with pluggable slots: Ranger, Archivist, and Sniper.
- Single pipeline with **mandatory approval gate** before execution.
- Selective knowledge system via `knowledge.index.yaml` + `source-hints`.
- Non-blocking learning loop (learning failures do not block mission result).
- Strong safety constraints: stop conditions, forbidden behaviors, and drift self-correction.

## Overview

**Strategist** is an autonomous skill for orchestrating documentation, diagram,
and analysis missions for AI agents.

It coordinates multi-phase work through three pluggable roles (slots):

```
Ranger (discovery) -> Role responsible for exploring the problem scope from the initial prompt
Archivist (refinement) -> Role responsible for refining the problem scope and creating an execution plan
Sniper (execution) -> Role responsible for materializing the approved documentation/handoff plan
```
Each role has a defined function, but the interesting part is that you can specify which skill fulfills that role.
Strategist orchestrates the flow, validates contracts, emits progress events, and enforces the approval gate.

It is **standalone by default** and can optionally integrate as a plugin into governance models (harness engineering) such as the **SDD Harness**.

---

## Problem

AI agents tend to fail when operating without governance:

- lose context between iterations
- confuse analysis with execution
- execute without diagnosis or an approved plan
- ignore architecture and existing decisions
- enter retry loops without traceability
- generate large, low-density prompts
- modify code before human approval exists

Strategist solves this with:

```
structural governance
+ routing via pluggable slots
+ selective context via knowledge index
+ mandatory approval gate
+ non-blocking learning loop
+ drift self-correction
```

---

## File Structure

```
strategist/
├── skill.yaml                       ← skill contract (slots, pipeline, forbidden_behaviors)
├── SKILL.md                         ← complete agent instructions
├── protocol.md                      ← mandatory routing rules
├── active.yaml                      ← generated on install (gitignore'd)
├── knowledge.index.yaml             ← knowledge index of sources
│
├── personas/
│   ├── pragmatic.yaml               ← analytical tone; labels: analysis/refinement/execution
│   └── epic.yaml                    ← strategic tone; labels: ranger/archivist/sniper
│
├── roles/
│   ├── default.yaml                 ← default slot bindings
│   ├── mission.yaml                 ← mission-specific bindings
│   └── spec-driven.yaml             ← bindings for spec-driven flow
│
├── schemas/
│   ├── intake.schema.yaml           ← mission_contract fields
│   └── progress-contract.yaml       ← progress event format
│
├── templates/
│   ├── pragmatic-standalone.yaml    ← active.yaml template: pragmatic, no SDD
│   ├── epic-standalone.yaml         ← active.yaml template: epic, no SDD
│   ├── epic-sdd.yaml                ← active.yaml template: epic, with SDD injection
│   ├── known-providers.yaml         ← catalog of known providers for the wizard
│   └── domain/                      ← workspace templates (.strategist/)
│       ├── index.yaml
│       ├── identity/
│       ├── directives/
│       ├── rubrics/
│       └── patterns/
│
├── memory/
│   ├── outcomes.jsonl               ← mission history (gitignore'd)
│   └── source-hints.yaml            ← learned priority adjustments (gitignore'd)
│
└── internal_skills/
    ├── prompt-intake/               ← classifies task_type, risk_level, constraints
    ├── context-enrichment/          ← queries knowledge index, applies source-hints
    ├── dossier-builder/             ← assembles minimal dossier within the token budget
    ├── response-critic/             ← evaluates slot output against rubric
    ├── learning-curator/            ← proposes entries for outcomes + source-hints
    └── archivist/                    ← refinement skill (default Archivist slot)
        ├── skill.yaml
        └── SKILL.md
```

### Workspace in the target repository (generated by install)

```
<base_path>/
├── todo/                            ← missions awaiting execution
├── pending/                         ← discovery artifacts in progress
├── refined/                         ← reviewed plans ready for approval
├── archived/                        ← completed execution reports
└── .strategist/                     ← internal domain (copied from templates/domain/)
    ├── index.yaml                   ← controls selective file loading
    ├── identity/
    │   ├── what-i-am.yaml
    │   └── drift-patterns.yaml
    ├── directives/
    ├── rubrics/
    └── patterns/
```

---

## Mission Pipeline

Complete pipeline: Ranger → Archivist → approval gate → Sniper

### Business Flow: Iteration Between Roles

```
                    ┌─────────────────────────────────────────┐
                    │              STRATEGIST                 │
                    │      Orchestrator — does not execute    │
                    └──────────────────┬──────────────────────┘
                                       │
                         ┌─────────────▼──────────────┐
                         │           RANGER           │
                         │         (discovery)        │
                         │  "What needs to be done?   │
                         │   What is the current      │
                         │   state?"                  │
                         │  → pending/<id>-analysis   │
                         └─────────────┬──────────────┘
                                       │
                         ┌─────────────▼──────────────┐
                         │         ARCHIVIST          │
                         │        (refinement)        │
                         │  "How to execute? What     │
                         │   decisions to make?"      │
                         │  → refined/<id>/           │
                         │    analysis.md             │
                         │    proposal.md             │
                         │    design.md               │
                         │    tasks.md                │
                         │  → Opportunity Attack      │
                         │    (ADR evaluation after   │
                         │     4 artifacts written)   │
                         └─────────────┬──────────────┘
                                       │
                    ┌──────────────────▼──────────────────┐
                    │           Approval Gate             │
                    │   MANDATORY STOP (if tasks.md is    │
                    │   not empty).                       │
                    │   Also covers pending side quests   │
                    │   consolidated by the Archivist.    │
                    └──────────────────┬──────────────────┘
                              approved │
                         ┌─────────────▼──────────────┐
                         │           SNIPER           │
                         │         (execution)        │
                         │  1. Side quests (if any)   │
                         │     mv/promotes artifacts  │
                         │  2. Main plan              │
                         │  → archived/<id>-report.md │
                         └─────────────┬──────────────┘
                                       │
                         ┌─────────────▼──────────────┐
                         │       Learning Phase       │
                         │      (non-blocking)        │
                         │  Records outcomes and      │
                         │  source-hints with         │
                         │  human approval            │
                         └────────────────────────────┘
```

#### Side Quests — Detail

Side quests are cross-phase scope observations. Any slot (Ranger, Archivist, Sniper) may detect them during their work. Archivist consolidates pre-execution side quest findings before presenting the approval gate. Sniper reports newly discovered side quests after execution.

Side quests surface at the approval gate alongside the main plan. Each workspace operation requires approval at the **main gate** — no execution occurs outside this step.

Examples of side quest patterns:
```
              todo/spec.md ──────────► already implemented in git?
                                              │ yes
                                              ▼
                                       propose → move → archived/

           pending/discovery.md ──────► has a plan in refined/?
                                              │ yes
                                              ▼
                                       propose → promote → archived

            refined/plan/ ────────────► has a report in archived/?
                                              │ yes
                                              ▼
                                       propose → promote → archived
```

#### Quick Draw — Detail

`quick_draw` is a rapid-note intent signal within the same mission:

- **Input:** explicit quick draw / rapid TODO capture prompt.
- **Ranger:** only normalizes the sentence into `idea: ...` without expanding scope.
- **Archivist:** assigns theme (`architecture`, `security`, `analysis`, `general`) and destination `.analysis/todo/<theme>.md`; computes `ideas_added`.
- **Gate:** approval at the main mission gate.
- **Sniper:** appends to the themed file; `todo/` is write-only from the skill perspective; returns `ideas_added`.

Without main-gate approval, nothing is written.

#### Treasure Chests — Scope

`active.yaml` can declare `treasure_chests` as optional offline knowledge sources.
Strategist filters by scope and passes them to each slot:

- `discovery` → Ranger
- `refinement` → Archivist
- `execution` → Sniper
- `all` → all slots

If no scope match exists, the slot continues without blocking.

**Index / Mine (current public commands):** `strategist treasure-chest index`
internalizes offline scanning, candidate detection, and jewel polishing into
deduplicated `status: proposed` jewels in one run. `strategist treasure-chest
mine` is the separate human curation step that promotes `proposed` jewels to
`accepted`/`verified`, or marks them `deprecated`. There is no public `scan`,
`polish`, or `pack` command — those are internal phases folded into `index`. Jewel
runtime consultation (preferring `accepted`/`verified` jewels over expanding full
source documents) and Scout/Treasure telemetry are **planned**, not yet emitted —
see `docs/observability-contract.md`.

---

### Internal Technical Flow

```
INVOCATION
─────────────────────────────────────────────────────────────────────
  user prompt
       │
       ▼
  ┌──────────────────────────────────────────────────────────────┐
  │ 1. Bootstrap                                                 │
  │    • Loads active.yaml (single source of config)             │
  │    • Resolves persona → tone_directive + phase_labels        │
  │    • SDD injection (if plugin active): overrides Sniper slot,│
  │      base_path, knowledge_paths, governance_context          │
  └──────────────────────────┬───────────────────────────────────┘
                             │
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 2. Preflight                              stops on 1st fail  │
  │    2a. Loads .strategist/index.yaml → load_always files      │
  │    2b. Loads identity/what-i-am.yaml + drift-patterns.yaml   │
  │    2c. Resolves slot providers (roles/<config>.yaml)         │
  │        skill_root → .claude/skills → registry                │
  │    2d. Validates risk contracts:                             │
  │        Ranger    → write_analysis                            │
  │        Archivist → write_analysis                            │
  │        Sniper    → controlled                                │
  │    emit: phase=preflight status=done                         │
  └──────────────────────────┬───────────────────────────────────┘
                             │
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 3. Intake                                                    │
  │    invokes prompt-intake skill                               │
  │    → mission_contract: task_type, risk_level, constraints    │
  └──────────────────────────┬───────────────────────────────────┘
                             │
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 4. Context Enrichment                                        │
  │    invokes context-enrichment (knowledge.index.yaml + hints) │
  │    invokes dossier-builder → minimal dossier by token budget │
  └──────────────────────────┬───────────────────────────────────┘
                             │
MISSION PHASES
─────────────────────────────────────────────────────────────────────
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 5a. RANGER (discovery slot)         contract: write_analysis │
  │     → pending/<mission_id>-analysis.md                       │
  │     emit: phase=<ranger_label> status=done                   │
  └──────────────────────────┬───────────────────────────────────┘
                             │
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 5b. ARCHIVIST (refinement slot)   contract: write_analysis   │
  │     input: discovery artifact + any side quests detected     │
  │     → refined/<mission_id>/                                  │
  │         analysis.md  proposal.md  design.md  tasks.md        │
  │     → Opportunity Attack (ADR evaluation — internal to       │
  │         Archivist, after all 4 artifacts are written)        │
  │     emit: phase=<archivist_label> status=done                │
  └──────────────────────────┬───────────────────────────────────┘
                             │
GATE AND EXECUTION
─────────────────────────────────────────────────────────────────────
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 6. Approval Gate                                             │
  │    reads tasks.md:                                           │
  │    • empty → analysis_delivered (no materialization)         │
  │    • internal → normal gate                                  │
  │    • external → gate + external scope warning                │
  │    Also covers pending side quests consolidated by Archivist  │
  │    STOP — waits for explicit response                        │
  └────────────┬──────────────────────────┬──────────────────────┘
  no/decline   │                          │ yes/approve
  analysis     │                          │
  delivered    │                          │
               │         ┌───────────────▼───────────────────┐
               │         │ 7. SNIPER (execution slot)        │
               │         │    contract: controlled            │
               │         │    1. Side quests (if any):        │
               │         │       mv/promotes stale artifacts  │
               │         │    2. Main plan (tasks.md)         │
               │         │    → archived/<mission_id>-report.md │
               │         │    emit: phase=<sniper_label> done  │
               │         └───────────────┬───────────────────┘
               │                         │
               └─────────────────────────┤
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 8. Learning Phase (non-blocking)                             │
  │    invokes response-critic → invokes learning-curator        │
  │    checkpoint to user before any write                       │
  │    failure: does not block the result                        │
  └──────────────────────────┬───────────────────────────────────┘
                             │
  ┌──────────────────────────▼───────────────────────────────────┐
  │ 9. Mission Result                                            │
  │    status: execution_done | analysis_delivered | blocked     │
  │    artifacts: discovery, side_quest_report?,                 │
  │               refined_plan, execution_report?                │
  └──────────────────────────────────────────────────────────────┘
```

---

### Main Flow

```
User prompt
  ↓
Bootstrap (active.yaml + persona + SDD injection)
  ↓
Preflight (validates slots, loads internal domain)
  ↓
Intake (extracts mission_contract)
  ↓
Context Enrichment (queries knowledge index → assembles dossier)
  ↓
Ranger / discovery (discovery slot)
  → may detect side quests
  ↓
Archivist / refinement (refinement slot)
  → analysis.md, proposal.md, design.md, tasks.md
  → Opportunity Attack (ADR evaluation — internal to Archivist, after 4 artifacts)
  ↓
Approval Gate ← MANDATORY STOP (if tasks.md is not empty)
  also covers pending side quests consolidated by Archivist
  ↓ (only with explicit approval)
Sniper / execution (execution slot)
  1. Approved side quests (move/promote stale artifacts)
  2. Main plan (tasks.md)
  ↓
Learning Phase (non-blocking)
  ↓
Mission Result
```

---

### 1. Bootstrap

When invoked, the agent:

1. Loads `active.yaml` (single source of configuration).
2. Resolves the persona (`personas/<mode>.yaml`) and applies `tone_directive` and `phase_labels`.
3. If `--mode` was passed, overrides the mode for this mission only.
4. If `--roles` was passed, overrides `roles_config` for this mission only.
5. If `governance_injection` is present:
   - Reports the governance gate as policy context
   - May provide `base_path` and knowledge paths
   - Does not replace the persona approval gate
   - Does not override the pipeline sequence or slot contracts

---

### 2. Preflight

Executed **before any slot or intake**. Stops on the first failure.

**2a. Internal domain**

Loads `<base_path>/.strategist/index.yaml`. If not found, continues without a domain.
If found, loads only the files listed under `load_always`. No file outside the index is loaded.

**2b. Identity files**

- `identity/what-i-am.yaml` → loads `core_invariants` (active throughout the entire mission)
- `identity/drift-patterns.yaml` → loads all patterns (used for self-correction)

**2c. Slot provider resolution**

For each slot (discovery, refinement, execution), resolves the provider's `skill.yaml` from:
`<skill_root>/skills/<provider>/skill.yaml` — single canonical path, no fallback chain.

If no path resolves: emits a blocked event and stops.

**2d. Risk contract validation**

- Ranger: `risk_score` MUST be `write_analysis`
- Archivist: `risk_score` MUST be `write_analysis`
- Sniper: `risk_score` MUST be `controlled`
- Mismatch → blocked event with `reason=slot_risk_mismatch`

**Completion event:** `[Strategist] phase=preflight status=done slots=ok`

---

### 3. Intake

Invokes the `prompt-intake` skill with the full user prompt.

Result (`mission_contract`):

```yaml
task_type: architecture_analysis | refactor | general | ...
risk_level: low | medium | high
constraints:
  delivery_strategy: incremental | total
  legacy_compatibility: required | not_required
  execution_intent: review_only | execute
```

Constraint conflicts → stops and asks the user for clarification.
Missing fields → defaults applied via `intake.schema.yaml`.

The `mission_contract` is passed to all slot providers.

---

### 4. Context Enrichment

Invokes `context-enrichment` with `task_type` and the mission token budget.

The enrichment:
1. Queries `knowledge.index.yaml` filtering by tags that match the `task_type`
2. Applies priority adjustments from `memory/source-hints.yaml`
3. Loads excerpts within the token budget

Loads files from `load_by_task_type[task_type]` in `index.yaml` (if internal domain is present).

Invokes `dossier-builder` to assemble the minimal dossier for slot providers. If no source matches, the dossier contains only `task_type` and `output_template`.

---

### 5. Mission Phases

#### 5a. Ranger (discovery slot)

```
[Strategist] phase=<ranger_label> status=running skill=<provider> checklist=0/3
```

Invokes the discovery slot provider with:
- User prompt
- `mission_contract.planning_rules`
- Context enrichment dossier

Artifact produced: `<base_path>/pending/<mission_id>-analysis.md`

```
[Strategist] phase=<ranger_label> status=done artifact=<path>
```

Failure → blocked event with `reason=ranger_failed`. Does not advance to Archivist.

#### 5b. Archivist (refinement slot)

```
[Strategist] phase=<archivist_label> status=running skill=<provider> checklist=1/3
```

Invokes the refinement slot provider with:
- Path to the discovery artifact
- Any side quests detected by Ranger
- `mission_contract.planning_rules`
- Dossier

Artifact produced: `<base_path>/refined/<mission_id>/` (subdirectory with four files)
- `analysis.md` — structured analysis of the discovery artifact
- `proposal.md` — what and why
- `design.md` — how (architecture, affected components, decisions)
- `tasks.md` — numbered documentation/materialization steps (Sniper input)

After writing all four artifacts, Archivist runs **Opportunity Attack** (ADR evaluation): assesses whether the refined artifacts justify opening an ADR. This is internal to the Archivist — not delegated to a slot.

Rules:
- Archivist never produces a standalone `.md` in `refined/` — always the subdirectory with the four files
- If `tasks.md` is empty or absent after Archivist completes, Sniper is not invoked

```
[Strategist] phase=<archivist_label> status=done artifact=<path>
```

Failure → blocked event. Does not present the approval gate.

---

### 6. Approval Gate (MANDATORY)

After the Archivist completes, Strategist reads `tasks.md` before presenting the gate:

**If `tasks.md` is empty or absent:**
  emits `[Strategist] phase=approval_gate status=analysis_delivered`, returns result `status: analysis_delivered`.
  The gate **is not presented** — the mission is complete.

**If `tasks.md` contains tasks only within `<base_path>/`:**
  presents the gate once with the visible plan.

**If `tasks.md` contains tasks that mutate source code, git state, system config, or non-documentation files:**
  blocks the mission and returns to Archivist for a documentation-only refinement.

The gate also covers pending side quests consolidated by the Archivist — upon approval, the Sniper first executes side quests and then the main plan.

Presents to the user (active persona template):

```
Archivist briefing complete. Mission plan at: <artifact_path>

Authorize Sniper deployment? (yes / no / review)
```

Responses:
- **yes / approve / authorize** → advances to Sniper
- **no / decline / stop** → emits `[Strategist] phase=approval_gate status=analysis_delivered`, returns result `status: analysis_delivered` with paths to discovery and refined plan artifacts
- **review** → displays plan content, re-prompts

Invoking Sniper without explicit approval is a **forbidden behavior**.

---

### 7. Sniper (execution slot)

```
[Strategist] phase=<sniper_label> status=running skill=<provider> checklist=2/3
```

Invokes the execution slot provider with:
- Approved side quest items (if any) — items surfaced during discovery or refinement and approved at the gate; Sniper executes these alongside primary targets
- Path to the approved refined plan
- `mission_contract.planning_rules`

**Execution order:**
1. Side quests (if manifest is non-empty): `mv`, artifact promotion, `Status:` updates. No writes outside `<base_path>/`.
2. Main plan (`tasks.md`).

Side quest failure is **non-blocking** — records the failure, continues with the main plan.

Artifact produced: `<base_path>/archived/<mission_id>-report.md`

```
[Strategist] phase=<sniper_label> status=done artifact=<path>
```

---

### 8. Learning Phase (non-blocking)

After the mission completes (status `execution_done` or `analysis_delivered`):

1. Invokes `response-critic` with slot outputs and the `task_type` rubric
2. Invokes `learning-curator` with the critic evaluation, mission result, and `task_type`

The `learning-curator` **MUST present a checkpoint to the user** before writing any file.
Proposes updates to:
- `memory/outcomes.jsonl` — mission record (append-only)
- `memory/source-hints.yaml` — priority adjustments for knowledge sources

Both require explicit approval (can be approved/rejected individually).

**Failure in the learning phase does not block or modify the mission result.**

---

### 9. Mission Result

```yaml
mission_id: <id>
status: execution_done | analysis_delivered | blocked
artifacts:
  discovery: <path>           # present if Ranger executed
  side_quest_report: inline   # present if side quests executed (inline block, not a file)
  refined_plan: <path>        # present if Archivist executed
  execution_report: <path>    # present if Sniper executed
blockers: []                  # blocker codes if status=blocked
```

---

## Operation Modes (Personas)

Strategist has two modes with the **same pipeline** and a **different voice**.

| Aspect | Pragmatic | Epic |
|--------|-----------|------|
| **Tone** | Analytical, direct | Strategic, decisive |
| **Discovery label** | `analysis` | `ranger` |
| **Refinement label** | `refinement` | `archivist` |
| **Execution label** | `execution` | `sniper` |
| **Approval prompt** | "Refinement complete. Proceed?" | "Authorize Sniper deployment?" |
| **Default template** | `pragmatic-standalone.yaml` | `epic-standalone.yaml` / `epic-sdd.yaml` |

Selection:
- Via `active.yaml`: `mode: pragmatic` or `mode: epic`
- Per-mission override: `--mode pragmatic` or `--mode epic`

---

## Knowledge System

### knowledge.index.yaml

Multi-source index for context enrichment. Each source has:

```yaml
sources:
  - id: project-architecture
    type: docs
    path: /abs/path/to/docs/architecture
    tags: [architecture, system-design, architecture_analysis]
    priority: high

  - id: past-good-examples
    type: examples
    path: .analysis/.strategist/patterns/good
    tags: [examples, patterns, refactor, architecture_analysis]
    priority: medium

  - id: team-directives
    type: directives
    path: /abs/path/to/team-directives.md
    tags: [all]
    priority: high
```

`context-enrichment` filters by tags that match the mission's `task_type` and loads only relevant sources within the token budget.

### source-hints.yaml

Learned priority adjustment layer. Overlaid on the index before ranking. Updated by `learning-curator` with human approval.

### Internal Domain (.strategist/)

The `index.yaml` controls selective loading — the agent **never scans the full directory**:

```yaml
load_always:
  - identity/what-i-am.yaml
  - identity/drift-patterns.yaml
  - directives/core.yaml

load_by_task_type:
  architecture_analysis:
    - directives/by-task/architecture-analysis.yaml
    - rubrics/architecture-analysis.yaml
  refactor:
    - directives/by-task/architecture-analysis.yaml
    - rubrics/architecture-analysis.yaml

load_on_demand:
  - patterns/good/
  - patterns/bad/
  - memory/lessons.yaml
```

---

## Slot Configuration (roles/)

Each `roles/<config>.yaml` file declares the providers for the three slots:

### roles/default.yaml (standalone)

```yaml
discovery: brainstorming
refinement: openspec-explore
execution: sniper
```

### roles/mission.yaml (mission-specific)

```yaml
discovery: brainstorming
refinement: archivist
execution: sniper
```

Per-mission override: `--roles mission`

---

## SDD Integration (Optional)

Strategist can receive governance context from SDD or another governance adapter.

When active, a governance adapter may inject policy context:

```yaml
governance_injection:
  provider: sdd
  execution_gate: allowed
  base_path: .sdd/analysis          # overrides base_path
  knowledge_paths:
    - .sdd/docs                     # added to knowledge index
```

**Rules:**
- Governance does not override the Sniper slot
- Governance does not replace the explicit Approval Gate
- `knowledge_paths` are **added** to sources, not replaced
- governance context is read-only and does not override `protocol.md`

Template for use with SDD: `templates/epic-sdd.yaml`

---

## Stop Conditions

| Code | Condition | Resolution |
|------|-----------|------------|
| `slot_provider_not_found` | Provider's skill.yaml not found | Check id in roles config and skill root path |
| `slot_risk_mismatch` | Ranger ≠ `write_analysis`, Archivist ≠ `write_analysis`, or Sniper ≠ `controlled` | Replace provider |
| `intake_conflict_unresolved` | Two mutually exclusive constraint aliases in the prompt | User must clarify |
| `preflight_failed` | Any preflight check failed | See emitted reason code |
| `user_denies_execution` | User declined at the approval gate | Returns `analysis_delivered` (not an error) |
| `ranger_failed` | Ranger did not produce an artifact | Does not advance to Archivist |
| `refinement_failed` | Archivist did not produce an artifact | Does not present the approval gate |
| `side_quest_sniper_failed` | Sniper failed on side quests | Non-blocking — continues with the main plan |

---

## Forbidden Behaviors

The following behaviors are **never allowed**:

1. **Execute discovery, refinement, or execution directly** — always invoke the configured slot provider. If no provider exists, stop with `slot_provider_not_found`.

2. **Invoke the execution slot without explicit user approval** — the approval gate is mandatory. Any path that reaches the execution slot without an affirmative response to the approval prompt is a prohibited bypass.

3. **Write config to the target repository** — `active.yaml`, `personas/`, `roles/`, `memory/`, `knowledge.index.yaml`, and any other skill root config must never be written to the target repository.

4. **Load files not referenced in `index.yaml`** — when the internal domain is present, only files listed in `load_always`, `load_by_task_type`, or `load_on_demand` may be loaded.

5. **Write to `memory/` without approval** — the `learning-curator` must present proposed entries for review before any write.

6. **Resolve the execution slot from an undeclared source** — the execution slot provider must come from `roles/<config>.yaml` or `sdd_injection.execution_provider`.

7. **Skip preflight** — preflight runs before intake, on every invocation, including re-invocations with the same config.

8. **Delegate Opportunity Attack to a slot provider** — ADR evaluation is internal to the Archivist after writing the four refined artifacts. It is not a separate Strategist phase and is not delegated to Ranger or Sniper.

9. **Ask the Sniper to create documents, specs, or plans** — creation of analysis artifacts is the Archivist's responsibility (contract: `write_analysis`). Sniper executes; it never writes analyses.

10. **Execute side quests before approval at the main gate** — side quests are executed by the Sniper after explicit approval at the main gate, not before the Archivist.

---

## Drift Self-Correction

When `drift-patterns.yaml` is loaded, the agent checks patterns before each phase:

| Pattern | Symptom | Correction |
|---------|---------|------------|
| `direct_execution` | About to execute slot work directly | Stop. Identify active slot. Invoke provider. Resume. |
| `silent_phase_advance` | About to start next phase without emitting a `done` event | Emit `done` event first. |
| `approval_bypass` | About to invoke Sniper without asking the user | Stop. Present approval gate prompt. |
| `scope_expansion` | Addressing something outside the user's mission | Stop. Return to mission scope. |
| `sniper_provider_override` | Resolved Sniper from a source other than roles config or sdd_injection | Stop. Re-resolve from declared source. |
| `side_quest_approval_bypass` | About to move files from opportunity_attack without passing through the main gate | Stop. Side quests only execute after explicit approval at the main gate. |
| `route_plan_creation_to_sniper` | About to ask Sniper to create a document, spec, or plan | Stop. Artifact creation is Archivist's work. Return to phase 5c. |
| `opportunity_attack_as_slot` | About to delegate Opportunity Attack (ADR evaluation) to Ranger or Sniper | Stop. Opportunity Attack is internal to Archivist — run it after the four refined artifacts are written. |

---

## Architectural Decisions

### Standalone-first

Strategist does not require SDD or any governance framework. SDD integration is optional and additive — it does not modify pipeline logic.

### Identical pipeline for both modes

Pragmatic and Epic share the same pipeline. The separation is vocabulary and tone only. Adding new modes in the future requires only a new `personas/<mode>.yaml` file.

### Preflight validates all slots before starting

Fast failure in preflight avoids partial execution. Discovering a risk mismatch after the Ranger has already run would create orphaned artifact state and a difficult recovery situation.

### Non-blocking Learning Loop

The Learning Loop is an optimization layer. Failure in any loop skill (prompt-intake, context-enrichment, dossier-builder, response-critic, learning-curator) does not block the mission result.

### Selective loading via index.yaml

The internal domain grows over time with examples, lessons, and rubrics. Loading it entirely on every mission would create a token budget problem. The `index.yaml` limits the hot-path to 2–4 files relevant to the `task_type`.

### Two-file learning cache with independent approval

`outcomes.jsonl` and `source-hints.yaml` are approved separately — the user may want to record the mission result without agreeing to a suggested source priority adjustment.

---

## Progress Flow

Every phase transition emits exactly one event:

```
[Strategist] phase=preflight status=done slots=ok
[Strategist] phase=<ranger_label> status=running skill=<provider> checklist=0/3
[Strategist] phase=<ranger_label> status=done artifact=<path>
[Strategist] phase=opportunity_attack status=running
[Strategist] phase=opportunity_attack status=done side_quests=N
[Strategist] phase=<archivist_label> status=running skill=<provider> checklist=1/3
[Strategist] phase=<archivist_label> status=done artifact=<path>
[Strategist] phase=approval_gate status=waiting                 # only if tasks.md is non-empty
[Strategist] phase=<sniper_label> status=running skill=<provider> checklist=2/3
[Strategist] phase=side_quest_execution status=running          # only if side_quests > 0 and approved
[Strategist] phase=side_quest_execution status=done             # only if side_quests > 0 and approved
[Strategist] phase=<sniper_label> status=done artifact=<path>
```

Emitting a `running` event and advancing to the next phase without emitting `done` is a violation of the `silent_phase_advance` pattern.

---

For complete agent instructions, see `.strategist/SKILL.md`.
For mandatory routing rules, see `.strategist/protocol.md`.
For the complete skill contract, see `.strategist/skill.yaml`.
