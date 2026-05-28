# Strategist — Implementation Plan

**Based on**: `tasks.md` (57 tasks) + `design.md` + `.analysis/pending/2026-05-27-strategist-v2-design.md`
**Target**: New, isolated repository for the Strategist skill

---

## Dependency Graph

```
Phase 1: Foundation Contracts
  └──► Phase 2: Personas + Role Configs
         └──► Phase 3: Install Flow
  └──► Phase 4: Core Agent Instructions (SKILL.md)
  └──► Phase 5: Knowledge System
         └──► Phase 7: Learning Loop Skills
  └──► Phase 6: Internal Domain Templates  (parallel with 5)
         └──► Phase 7
Phase 7: All Skill Files (assembled)
  └──► Phase 8: Tests
         └──► Phase 9: SDD Integration (optional)
```

---

## Phase 1 — Foundation Contracts
**Goal**: Define all schemas and core contracts before writing agent instructions or scripts.
**Parallelism**: All tasks in this phase can be done simultaneously.
**Done when**: All schema files exist; `skill.yaml` core structure validated.

| Task | File | Notes |
|------|------|-------|
| 2.1  | `skill.yaml` | Core contract: slot definitions (discovery/refinement/execution), parameters, forbidden behaviors, optional sdd_injection block |
| 2.4  | `intake.schema.yaml` | delivery_strategy, legacy_compatibility, execution_intent + all aliases |
| 2.5  | `progress-contract.yaml` | `[Strategist] phase={label} status={status}` format; label sourced from active persona |
| 2.6  | `active.yaml` schema | Fields: mode, base_path, roles_config, knowledge_index_path |
| 3.1  | `knowledge.index.yaml` schema | sources[] with id, type, path, tags[], priority |
| 0.3  | `templates/` (3 files) | `pragmatic-standalone.yaml`, `epic-standalone.yaml`, `epic-sdd.yaml` — fully-filled `active.yaml` instances |

---

## Phase 2 — Personas and Role Configs
**Depends on**: Phase 1 (`active.yaml` schema, `skill.yaml` slot names)
**Parallelism**: All 4 tasks parallel.
**Done when**: Both personas defined; both SDD role configs created; `--mode` flag documented in skill.yaml.

| Task | File | Notes |
|------|------|-------|
| 1.1  | `personas/pragmatic.yaml` | phase_labels: analysis/refinement/execution; analytical tone; prompt_templates |
| 1.2  | `personas/epic.yaml` | phase_labels: scout/engineer/hunter; strategic tone; prompt_templates |
| 1.3  | `skill.yaml` update | Add `--mode` flag documentation; describe persona override behavior |
| 2.7  | `roles/sdd-mission.yaml` | discovery=sdd-diagnose, refinement=sdd-engineer, execution=_injected_by_sdd |
| 2.8  | `roles/spec-driven.yaml` | discovery=brainstorm, refinement=openspec, execution=_injected_by_sdd |

---

## Phase 3 — Install Flow
**Depends on**: Phase 1 (templates), Phase 2 (personas exist so defaults are valid)
**Parallelism**: 0.1 and 0.2 have soft dependency (wizard calls install internals); 0.4 and 0.5 are parallel.
**Done when**: `curl -sL strategist.run | sh` installs with defaults; `--wizard` shows TUI; workspace scaffolds in target repo.

| Task | File | Notes |
|------|------|-------|
| 0.3  | *(already Phase 1)* | Templates already done |
| 0.4  | `install.sh` — scaffold section | Creates `<base_path>/todo|pending|refined|done/.strategist/` in target repo |
| 0.5  | `SKILL.md` — footprint rule | Document "zero config in target repo"; enforced as a forbidden behavior |
| 0.1  | `install.sh` — silent mode | Copies `templates/pragmatic-standalone.yaml` → `active.yaml`; calls scaffold |
| 0.2  | `install.sh` — `--wizard` branch | TUI: template select → persona → base_path → Scout/Engineer/Hunter → knowledge sources → writes `active.yaml` |

---

## Phase 4 — Core Agent Instructions
**Depends on**: Phase 1 (`skill.yaml`, schemas), Phase 2 (personas, roles)
**Parallelism**: `SKILL.md` and `protocol.md` can be written in parallel.
**Done when**: Agent can read `SKILL.md` and execute the full mission flow end-to-end (conceptually).

| Task | File | Notes |
|------|------|-------|
| 2.2  | `SKILL.md` | Full agent instructions: load active.yaml → load persona → preflight → intake → context enrichment → Scout → Engineer → approval gate → Hunter → learning phase |
| 1.4  | `SKILL.md` update | Persona loading section: how agent reads phase_labels, applies tone_directive, formats progress events |
| 2.3  | `protocol.md` | Mandatory routing rules: when to stop, what is forbidden, how to handle slot failures |

**SKILL.md structure** (for reference during authoring):
```
1. Bootstrap (load active.yaml, persona, identity)
2. Preflight (validate slots, load .strategist/index.yaml)
3. Intake (extract constraints via intake.schema.yaml)
4. Context Enrichment (query knowledge.index.yaml → build dossier)
5. Mission Phases (slot invocation loop: Scout → Engineer)
6. Approval Gate (mandatory pause; present refined plan)
7. Hunter (invoke execution slot)
8. Learning Phase (response-critic → learning-curator → approval)
```

---

## Phase 5 — Knowledge System
**Depends on**: Phase 1 (`knowledge.index.yaml` schema, `active.yaml` schema)
**Parallelism**: 3.2 and 3.3 are parallel; 3.4 depends on 3.2.
**Done when**: `context-enrichment` skill contract is complete; `source-hints.yaml` empty state exists.

| Task | File | Notes |
|------|------|-------|
| 3.3  | `memory/source-hints.yaml` | Empty initial state with schema comment: source_id, annotation, priority_adjustment, derived_from_mission |
| 3.2  | `context-enrichment` — query flow doc | Document in skill.yaml: load index → filter tags → apply source-hints → load excerpts → respect token budget |
| 3.4  | `context-enrichment/skill.yaml` | Full skill contract: input=task_type+token_budget, output=ranked_excerpts+rubric; reads knowledge_index_path from active.yaml |

---

## Phase 6 — Internal Domain Templates
**Depends on**: Phase 1 (slot names for identity files)
**Parallelism**: All 7 tasks fully parallel — these are independent YAML/markdown files.
**Done when**: All template files exist; `strategist init` can copy them to target repo workspace.

| Task | File | Notes |
|------|------|-------|
| 6.1  | `templates/domain/index.yaml` | load_always, load_by_task_type (architecture_analysis, refactor), load_on_demand |
| 6.2  | `templates/domain/identity/what-i-am.yaml` | i_am, i_am_not, core_invariants |
| 6.3  | `templates/domain/identity/drift-patterns.yaml` | 5 patterns: direct_execution, silent_phase_advance, approval_bypass, scope_expansion, hunter_provider_override |
| 6.4  | `templates/domain/directives/core.yaml` | always/never rules |
| 6.5  | `templates/domain/directives/by-task/architecture-analysis.yaml` | Task-specific directives |
| 6.6  | `templates/domain/rubrics/architecture-analysis.yaml` | must_have, must_not, score_threshold |
| 6.7  | `templates/domain/patterns/good/README.md` + `patterns/bad/README.md` | Placeholder with structure docs |

---

## Phase 7 — All Skill Files
**Depends on**: Phase 4 (SKILL.md → agent instruction style), Phase 5 (knowledge system contracts), Phase 6 (domain template structure)
**Parallelism**: `prompt-intake`, `dossier-builder`, `response-critic`, `sdd-engineer` are parallel. `learning-curator` and `context-enrichment` (final version) depend on earlier skill contracts.
**Done when**: All 8 skill files exist with complete contracts.

| Task | File | Notes |
|------|------|-------|
| 5.1  | `prompt-intake/skill.yaml` | Classifies task_type, risk_level, intake constraints; output feeds context-enrichment |
| 5.3  | `dossier-builder/skill.yaml` | Input: context-enrichment output + .strategist/ files; output: minimal dossier within token budget |
| 5.4  | `response-critic/skill.yaml` | Input: mission output + rubric; output: evaluation score + gaps |
| 4.1  | `sdd-engineer/skill.yaml` | risk_score=read_only; input_policy=requires discovery artifact; output_policy=all required sections |
| 4.2  | `sdd-engineer/SKILL.md` | Load discovery artifact → produce: tasks, subitems, technical details, Do/Don't, execution checklist |
| 5.2  | `context-enrichment/skill.yaml` | Final version with knowledge index integration (builds on Phase 5 draft) |
| 5.5  | `learning-curator/skill.yaml` | Proposes outcomes.jsonl + source-hints.yaml; presents both for independent approval; forbidden to write without approval |
| 4.3  | skill registry | Add sdd-engineer entry (format depends on target repo's registry format) |

---

## Phase 8 — Tests
**Depends on**: All previous phases complete.
**Parallelism**: Tests grouped by concern — install tests, persona tests, knowledge tests, pipeline tests, learning tests can be written in parallel.
**Done when**: All 19 test cases written and passing.

### Install tests (8.1–8.4)
- Silent install → `active.yaml` from pragmatic-standalone
- Wizard → `active.yaml` from selection
- Workspace structure in target repo
- Zero config files in target repo root

### Persona tests (8.5–8.7)
- Pragmatic labels in progress events
- Epic labels in progress events
- `--mode` override works

### Knowledge tests (8.8–8.10)
- Query by task_type tags
- source-hints overlay applied
- Empty result on no tag match (no error)

### Pipeline tests (8.11–8.14)
- Preflight pass / fail (risk_score, provider not found)
- Intake extraction
- Approval gate enforcement

### Learning cache tests (8.15–8.19)
- No write without approval (outcomes)
- No write without approval (source-hints)
- Independent approval per file
- Learning failure doesn't block result
- SDD knowledge_paths append behavior

---

## Phase 9 — SDD Integration (Optional)
**Depends on**: Phase 8 complete.
**Condition**: Only when `sdd-analysis-plugin-protocol` change is deployed in the SDD harness repo.

| Task | File | Notes |
|------|------|-------|
| 7.1  | `.sdd/plugins/registry.yaml` | Strategist entry with full sdd_injection: base_path, execution_provider, approval_gate, knowledge_paths, governance_context |
| 7.2  | validation | `sdd plugin validate strategist` passes |

---

## Repository Structure (target state after Phase 8)

```
strategist/                          ← new isolated repo
├── install.sh
├── skill.yaml
├── SKILL.md
├── protocol.md
├── active.yaml                      ← generated at install (gitignore'd)
├── templates/
│   ├── pragmatic-standalone.yaml
│   ├── epic-standalone.yaml
│   ├── epic-sdd.yaml
│   └── domain/                      ← workspace templates
│       ├── index.yaml
│       ├── identity/
│       ├── directives/
│       ├── rubrics/
│       └── patterns/
├── personas/
│   ├── pragmatic.yaml
│   └── epic.yaml
├── roles/
│   ├── sdd-mission.yaml
│   └── spec-driven.yaml
├── schemas/
│   ├── intake.schema.yaml
│   └── progress-contract.yaml
├── knowledge.index.yaml             ← user-managed (gitignore or versioned)
├── memory/
│   ├── outcomes.jsonl               ← gitignore'd
│   └── source-hints.yaml           ← gitignore'd
└── skills/
    ├── prompt-intake/skill.yaml
    ├── context-enrichment/skill.yaml
    ├── dossier-builder/skill.yaml
    ├── response-critic/skill.yaml
    ├── learning-curator/skill.yaml
    └── sdd-engineer/
        ├── skill.yaml
        └── SKILL.md
```

---

## Execution Order Summary

```
Phase 1  Foundation Contracts          (all parallel)         ~8 files
Phase 2  Personas + Role Configs       (all parallel)         ~5 files
Phase 3  Install Flow                  (sequential within)    ~3 scripts/docs
Phase 4  Core Agent Instructions       (2 parallel)           ~3 files
Phase 5  Knowledge System              (mostly parallel)      ~3 files
Phase 6  Internal Domain Templates     (all parallel)         ~9 files
Phase 7  All Skill Files               (mostly parallel)      ~8 files
Phase 8  Tests                         (grouped parallel)     ~19 tests
Phase 9  SDD Integration               (optional)             ~2 files
```

**Phases 4, 5, and 6 can overlap** — Phase 4 (SKILL.md) and Phase 6 (templates) have no dependency on each other. Phase 5 (knowledge system) also runs independently. The bottleneck is Phase 7, which waits for all three.

---

## Done Criteria per Phase

| Phase | Done When |
|-------|-----------|
| 1 | All schema files exist; `skill.yaml` has complete slot contract |
| 2 | Both personas renderable; `--mode` documented; role configs complete |
| 3 | `sh install.sh` creates workspace in a test repo; `--wizard` shows prompts |
| 4 | Agent reading SKILL.md can execute full flow without ambiguity |
| 5 | `context-enrichment` contract complete; query flow documented and testable |
| 6 | `strategist init` can scaffold `.strategist/` from templates |
| 7 | All skills have valid contracts; `sdd-engineer` produces all required sections |
| 8 | All 19 tests pass |
| 9 | `sdd plugin validate strategist` passes (SDD harness only) |
