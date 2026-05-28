## 0. Distribution and Install Flow

- [ ] 0.1 Define install script (`install.sh`): silent mode (default) creates `active.yaml` from `templates/pragmatic-standalone.yaml`; `--wizard` flag launches TUI
- [ ] 0.2 Create TUI wizard script: prompts for template selection → persona (pragmatic/epic) → base_path → Scout/Engineer/Hunter providers → knowledge sources; writes `active.yaml`
- [ ] 0.3 Create `<skill_root>/strategist/templates/` with 3 pre-built configs: `pragmatic-standalone.yaml`, `epic-standalone.yaml`, `epic-sdd.yaml`
- [ ] 0.4 Define workspace scaffolding contract triggered by install or `strategist init`: creates `<base_path>/todo/`, `pending/`, `refined/`, `done/`, `.strategist/` in target repo
- [ ] 0.5 Document "zero footprint in target repo" rule: only workspace artifacts (`<base_path>/`) go in target repo; all config (`active.yaml`, `personas/`, `roles/`, `memory/`, `knowledge.index.yaml`) stays in skill root

## 1. Personas and Modes

- [ ] 1.1 Create `<skill_root>/strategist/personas/pragmatic.yaml` with phase_labels (analysis/refinement/execution), tone_directive, prompt_templates, progress_format prefix `[Strategist]`
- [ ] 1.2 Create `<skill_root>/strategist/personas/epic.yaml` with phase_labels (scout/engineer/hunter), tone_directive ("strategic commander"), prompt_templates, progress_format prefix `[Strategist]`
- [ ] 1.3 Document `--mode` flag in skill.yaml: overrides active.yaml persona per-mission
- [ ] 1.4 Update SKILL.md to load persona from active.yaml at preflight and apply phase labels and tone throughout mission

## 2. Strategist Skill Core Files

- [ ] 2.1 Create `<skill_root>/strategist/skill.yaml` with parameters (mode, roles_config), pipeline slot definitions using generic internal names (discovery/refinement/execution), stop_conditions, and forbidden behaviors; add optional sdd_injection block for SDD integration
- [ ] 2.2 Create `<skill_root>/strategist/SKILL.md` with agent instructions: load active.yaml → load persona → preflight → intake → context enrichment → phase flow → approval gate → learning phase
- [ ] 2.3 Create `<skill_root>/strategist/protocol.md` with mandatory routing rules and stop behavior
- [ ] 2.4 Create `<skill_root>/strategist/intake.schema.yaml` with delivery_strategy, legacy_compatibility, execution_intent fields and all aliases
- [ ] 2.5 Create `<skill_root>/strategist/progress-contract.yaml` with event format `[Strategist] phase=<persona_label>`, statuses, and output paths
- [ ] 2.6 Create `active.yaml` schema: mode, base_path, roles_config, knowledge_index_path
- [ ] 2.7 Create `<skill_root>/strategist/roles/sdd-mission.yaml` (discovery=sdd-diagnose, refinement=sdd-engineer, execution=_injected_by_sdd) — SDD integration
- [ ] 2.8 Create `<skill_root>/strategist/roles/spec-driven.yaml` (discovery=brainstorm, refinement=openspec, execution=_injected_by_sdd) — SDD spec-driven

## 3. External Knowledge Index

- [ ] 3.1 Create `<skill_root>/strategist/knowledge.index.yaml` schema: sources[] with id, type, path, tags, priority fields
- [ ] 3.2 Document knowledge query flow in context-enrichment: load index → filter by task_type tags → apply source-hints.yaml priority overrides → load excerpts within token budget
- [ ] 3.3 Create `<skill_root>/strategist/memory/source-hints.yaml` empty initial state with schema: source_id, annotation, priority_adjustment, derived_from_mission
- [ ] 3.4 Update context-enrichment skill.yaml to read knowledge.index.yaml path from active.yaml and apply source-hints overlay

## 4. sdd-engineer Skill

- [ ] 4.1 Create `<skill_root>/sdd-engineer/skill.yaml` with risk_score=read_only, input_policy (requires discovery artifact), output_policy (required sections), forbidden_behaviors
- [ ] 4.2 Create `<skill_root>/sdd-engineer/SKILL.md` with agent instructions: load discovery artifact, produce all required sections, mark insufficient evidence, persist to refined/ path
- [ ] 4.3 Add `sdd-engineer` entry to skill registry (if applicable)

## 5. Learning Loop Skills

- [ ] 5.1 Create `<skill_root>/prompt-intake/skill.yaml` — classifies task_type, risk_level, intake constraints
- [ ] 5.2 Create `<skill_root>/context-enrichment/skill.yaml` — reads knowledge.index.yaml from active.yaml, applies source-hints.yaml overlay, returns ranked excerpts + rubrics within token budget
- [ ] 5.3 Create `<skill_root>/dossier-builder/skill.yaml` — assembles minimal dossier from context-enrichment output + .strategist/ identity files
- [ ] 5.4 Create `<skill_root>/response-critic/skill.yaml` — evaluates output against rubric must_have/must_not
- [ ] 5.5 Create `<skill_root>/learning-curator/skill.yaml` — proposes outcomes.jsonl entry AND source-hints.yaml annotation; presents both for separate approval; writes only on explicit approval; forbidden to write without approval

## 6. Strategist Internal Domain Templates

Note: These are template files shipped with the skill. On `strategist init`, they are copied to `<base_path>/.strategist/` in the target repo workspace. All config (persona, roles, memory, knowledge index) stays in skill root — only these workspace files go to target repo.

- [ ] 6.1 Create template `index.yaml` with load_always, load_by_task_type (architecture_analysis, refactor), load_on_demand sections
- [ ] 6.2 Create template `identity/what-i-am.yaml` with i_am, i_am_not, core_invariants
- [ ] 6.3 Create template `identity/drift-patterns.yaml` with direct_execution, silent_phase_advance, approval_bypass, scope_expansion, hunter_provider_override patterns
- [ ] 6.4 Create template `directives/core.yaml` with always/never rules
- [ ] 6.5 Create template `directives/by-task/architecture-analysis.yaml`
- [ ] 6.6 Create template `rubrics/architecture-analysis.yaml` with must_have, must_not, score_threshold
- [ ] 6.7 Create template `patterns/good/` and `patterns/bad/` with README placeholder

## 7. SDD Plugin Registration (Optional — requires sdd-analysis-plugin-protocol)

- [ ] 7.1 Add Strategist entry to `.sdd/plugins/registry.yaml` with full sdd_injection block: base_path, execution_provider, approval_gate, knowledge_paths (`.sdd/docs`), governance_context
- [ ] 7.2 Verify Strategist entry passes `sdd plugin validate strategist`

## 8. Tests

- [ ] 8.1 Write test: silent install generates active.yaml from pragmatic-standalone template
- [ ] 8.2 Write test: --wizard install prompts for template selection and writes active.yaml
- [ ] 8.3 Write test: install creates workspace structure at configured base_path in target repo
- [ ] 8.4 Write test: zero config files written to target repo root (only <base_path>/ structure)
- [ ] 8.5 Write test: pragmatic persona uses "analysis/refinement/execution" labels in progress events
- [ ] 8.6 Write test: epic persona uses "scout/engineer/hunter" labels in progress events
- [ ] 8.7 Write test: --mode epic overrides active.yaml persona for single mission
- [ ] 8.8 Write test: context-enrichment queries knowledge.index.yaml by task_type tags
- [ ] 8.9 Write test: context-enrichment applies source-hints.yaml priority overrides
- [ ] 8.10 Write test: context-enrichment returns empty dossier (no error) when no sources match task_type
- [ ] 8.11 Write test: preflight passes when all slot providers have correct risk_score
- [ ] 8.12 Write test: preflight blocked when Scout provider has wrong risk_score
- [ ] 8.13 Write test: intake extracts delivery_strategy=total from "sem prazo" prompt
- [ ] 8.14 Write test: approval gate stops pipeline when user declines execution (Hunter not invoked)
- [ ] 8.15 Write test: learning-curator does not write outcomes.jsonl without explicit user approval
- [ ] 8.16 Write test: learning-curator does not write source-hints.yaml without explicit user approval
- [ ] 8.17 Write test: learning-curator allows approving outcomes but rejecting source-hints independently
- [ ] 8.18 Write test: learning loop failure does not block mission result
- [ ] 8.19 Write test: SDD injection knowledge_paths appended (not replacing) standalone knowledge index
