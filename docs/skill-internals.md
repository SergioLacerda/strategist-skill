# Skill Internals — Sub-skills, Contracts, and Schemas

**Status:** Accepted
**Last Updated:** 2026-06-26

This document describes the internal components of the Strategist skill runtime: the sub-skills automatically invoked by the orchestrator, the phase contracts, and the input/output schemas.

For the general pipeline and slot behavior, see `docs/architecture.md`.
For the canonical reading order of contracts, see `strategist/SKILL.md` and `docs/adr/0010-ordered-contracts-and-mission-observability.md`.
For configuration, see [configuration.md](configuration.md).

---

## Internal Sub-skills

The Strategist invokes 6 internal sub-skills on every mission. All have `risk_score: read_only` — they do not write to disk directly, except `learning-curator` (which requires explicit user approval).

### prompt-intake

**Category:** classification  
**When:** before the pipeline, immediately after bootstrap

Classifies the user prompt into `task_type`, `risk_level`, and extracts mission constraints (`delivery_strategy`, `legacy_compatibility`, `execution_intent`).

**Input:**
- `user_prompt` — free-form user text
- `intake_schema_path` — path to `schemas/intake.schema.yaml`

**Output:**
- `task_type` — task type (e.g. `architecture_analysis`, `refactor`, `general`)
- `risk_level` — `low`, `medium`, or `high`
- `constraints` — object with the 3 constraint fields

**Special behavior:** if two mutually exclusive aliases for the same field are detected in the prompt, returns `conflict=true` with the conflicting field. The pipeline stops and asks the user to resolve the conflict before proceeding.

---

### context-enrichment

**Category:** knowledge  
**When:** after prompt-intake, before discovery

Queries `knowledge.index.yaml` by the mission's `task_type`. Applies adjustments from `source-hints.yaml`. Returns ranked excerpts within the configured token budget.

**Input:**
- `task_type` — from prompt-intake output
- `token_budget` — maximum number of tokens for excerpts
- `knowledge_index_path` — path to the index
- `source_hints_path` — path to `memory/source-hints.yaml`

**Output:**
- `excerpts` — ranked list of excerpts (highest priority first)
- `rubric` — task_type rubric (from `.strategist/rubrics/`) or `null`
- `sources_queried` / `sources_matched` — counters

**Empty result is valid:** if no source matches the `task_type`, returns `excerpts: []` and the pipeline continues normally.

Effective priority = declared priority in the index + `priority_adjustment` from source-hints.

---

### dossier-builder

**Category:** assembly  
**When:** after context-enrichment, before discovery

Assembles the dossier that is passed to slot providers as knowledge context. Ensures the dossier does not exceed the token budget and never includes the raw identity files (`what-i-am.yaml`, `drift-patterns.yaml`).

**Input:**
- `task_type`
- `enrichment_output` — output from context-enrichment
- `identity_files` — `what-i-am.yaml` + `drift-patterns.yaml` (if available)
- `token_budget`

**Output — dossier structure:**

```yaml
task_type: string
directives: string | null
good_examples: array          # maximum 2 items
bad_examples: array           # maximum 1 item
rubric: object | null
output_template: string | null
token_count: integer
```

**Trim order when budget is exceeded:** bad_examples → good_examples (keeps the highest-score one) → directives. `task_type` and `output_template` are never trimmed.

---

### ranger (discovery slot)

**Category:** discovery  
**When:** discovery phase (configurable slot)

Produces the canonical analysis artifact that formally opens the mission for the user and serves as the direct basis for the Archivist.

**Input:**
- `user_prompt`
- `mission_contract`
- `dossier`
- `treasure_chests`

**Output:**
- `analysis_artifact_path` — `<base_path>/pending/<mission_id>-analysis.md`

**Required contract fields in the artifact:**
- `mission_id`
- `objective`
- `analysis_summary`
- `known_facts`
- `uncertainties`
- `recommended_refinement_focus`

---

### archivist (refinement slot)

**Category:** refinement  
**When:** refinement phase (configurable slot)

Reads the discovery artifact and produces a revised, implementable plan. It is the default provider for the `refinement` slot.

**Input:**
- `analysis_artifact_path` — path to the canonical Ranger artifact in `refined/`
- `base_path` — mission base directory
- `mission_contract` — `planning_rules` extracted by prompt-intake

**Output:**
- `analysis.md` — `<base_path>/refined/<mission_id>/analysis.md`
- `proposal.md` — `<base_path>/refined/<mission_id>/proposal.md`
- `design.md` — `<base_path>/refined/<mission_id>/design.md`
- `tasks.md` — `<base_path>/refined/<mission_id>/tasks.md`

The canonical refinement output is a four-artifact package. `refined/<mission_id>-plan.md` is historical drift, not the current contract.

After writing the four artifacts, the Archivist runs the **Opportunity Attack** (ADR evaluation): checks whether the refined artifacts justify opening an ADR. This evaluation is internal to the Archivist — it is not delegated to a slot.

---

### response-critic

**Category:** evaluation  
**When:** learning phase (non-blocking)

Evaluates the slot output against the `task_type` rubric. Produces a score and a list of gaps — feeds the `learning-curator`.

**Input:**
- `slot_output` — content of the slot output artifact
- `task_type`
- `rubric` — from context-enrichment; if `null`, returns `result=no_rubric`

**Output:**
- `result` — `pass`, `fail`, or `no_rubric`
- `score` — 0.0–1.0 (null when `no_rubric`)
- `must_have_present` / `must_have_missing` — rubric items found/missing
- `must_not_present` — forbidden items found (violations)

`result=pass` when `score >= rubric.score_threshold` AND `must_not_present` is empty.

---

### learning-curator

**Category:** learning  
**When:** learning phase, after execution (non-blocking)

Proposes entries for `memory/outcomes.jsonl` and `memory/source-hints.yaml`. **Writes nothing without explicit user approval.**

**Input:**
- `mission_result` — mission result
- `critic_evaluation` — response-critic output
- `task_type`
- `outcomes_path` and `source_hints_path`

**Required checkpoint:**
```
Learning checkpoint:
1. Record mission outcome? [mission_id / task_type / score / status]
   (yes / no)
2. Adjust source priority? [source_id / annotation / adjustment]
   (yes / no)
```

Approval is independent for each item — the user can approve outcomes and reject source hints (and vice versa).

**Failure in the learning phase never blocks the mission result.** If the checkpoint expires or the phase fails, nothing is written and the mission returns normally.

---

## Phase Contracts

The contracts in `.strategist/contracts/` define the formal contract for each internal orchestrator phase.

### Functional signals in the single pipeline

`quick_draw`, `opportunity_attack`, `critical_hit`, side quests, and `treasure_chests`
do not open parallel pipelines. They fit into the single flow
`Ranger -> Archivist -> approval gate -> Sniper`.

- **Quick Draw**: fast idea/task capture via dedicated route; writes only after gate; `todo/` is write-only from the skill's perspective.
- **Opportunity Attack**: ADR evaluation run by the Archivist after writing the four refined artifacts. Not delegated to a slot.
- **Critical Hit**: analysis artifact management route (`.md` files) within the `pending/`, `refined/`, and `archived/` folders in `<base_path>`.
- **Side Quests**: scope observations detected during any phase; Ranger, Archivist, and Sniper may detect them; Archivist consolidates at the gate; Sniper reports newly discovered side quests.

Main guardrail: no approved materialization occurs without approval at the mission gate.
The scope restriction applies only to prevent documentary materialization outside the approved scope.

### Treasure Chests

`treasure_chests` are offline knowledge sources declared in `active.yaml`.
The Strategist passes to each slot only the chests with compatible scope:

- `discovery` → Ranger
- `refinement` → Archivist
- `execution` → Sniper
- `all` → all slots

Absence of an applicable chest does not block the mission.

### bootstrap

Loads the active configuration (`active.yaml`, persona, roles) before any mission.

| | |
|-|-|
| **Inputs** | `skill_root`, `mode_override` (optional), `roles_override` (optional) |
| **Outputs** | `active`, `persona`, `roles`, `sdd_injection` (optional) |
| **Fast path** | `.strategist/.compiled/.config.gz` — if fresh, loads the compiled artifact directly |
| **Fallback** | If `.config.gz` is corrupted: loads YAML directly, emits `bootstrap=standard_path` |

Errors that stop: `active_yaml_not_found`, `persona_not_found`.

### preflight

Validates slot providers and loads the internal domain. Runs after bootstrap, before intake.

| | |
|-|-|
| **Inputs** | `active`, `persona`, `roles` |
| **Outputs** | `domain`, `slot_providers`, `preflight_status` |
| **Fast path** | `.strategist/.compiled/.domain.gz` |
| **Write scope** | Read-only |

Errors that stop: `slot_provider_not_found`, `slot_risk_mismatch`.  
`index_yaml_not_found` is non-blocking — pipeline continues without internal domain.

### Other contracts

| Contract | What it guarantees |
|----------|-------------------|
| `check-stale.yaml` | Format and behavior of the staleness check |
| `compile-config.yaml` | Sources and schema of `.config.gz` |
| `compile-domain.yaml` | Sources and schema of `.domain.gz` |
| `compile-knowledge-index.yaml` | Sources and schema of `.index.gz` |
| `compile-all.yaml` | Sequence and dependencies of the full compilation |
| `context-enrichment.yaml` | Input/output contract for context-enrichment |
| `learning-buffer.yaml` | Outcomes buffer behavior (maximum size, flush) |
| `learning-curator.yaml` | Required checkpoint before writing to memory/ |
| `preflight.yaml` | Slot validation and domain loading |
| `bootstrap.yaml` | Loading of active.yaml, persona, and roles |

---

## Schemas

### intake.schema.yaml

Defines the constraint fields recognized by `prompt-intake` and their natural language aliases.

**Fields:**

| Field | Default | Accepted values |
|-------|---------|----------------|
| `delivery_strategy` | `sprint` | `sprint`, `total` |
| `legacy_compatibility` | `required` | `required`, `not_required` |
| `execution_intent` | `review_only` | `review_only`, `execute` |

**Aliases by value:**

`delivery_strategy: sprint` → "por sprint", "faseado", "iterativo", "incremental", "fase a fase", "entrega faseada"  
`delivery_strategy: total` → "big bang", "sem prazo", "entrega total", "tudo de uma vez"

`legacy_compatibility: required` → "retrocompatível", "backwards compatible", "sem breaking changes", "não pode quebrar"  
`legacy_compatibility: not_required` → "pode quebrar", "breaking ok", "clean break"

`execution_intent: execute` → "executar", "implementar", "aplicar", "rodar", "fazer"  
`execution_intent: review_only` → "só análise", "sem execução", "apenas revisar", "só plano"

---

## Performance Metrics and Baseline

The canonical metrics for Strategist performance optimization are:

- `t_start_to_intake_ms`
- `t_intake_to_ranger_ms`
- `total_wall_time_ms`
- `tokens_in`
- `tokens_out`
- `lines_emitted`

The current baseline is recorded in `docs/performance-baseline.md` and must be updated whenever there is a contract, telemetry, or emission policy change that could alter perceived cost.

Metrics are exposed by the `mission_metrics` output signal at the intake checkpoint and at each phase transition. This keeps cost telemetry available without altering the visible pipeline order.

`confidence_threshold: 0.65` — aliases with confidence below this value receive the default.

Defines the required format for progress events emitted by the Strategist at each phase transition.

**Format:**
```
[Strategist] phase=<phase_label> status=<status> [additional fields]
```

**Statuses:**

| Status | Required fields | When |
|--------|----------------|------|
| `running` | `phase`, `status`, `skill`, `checklist` | Phase started |
| `done` | `phase`, `status`, `artifact` | Phase completed successfully |
| `blocked` | `phase`, `status`, `reason`, `action` | Phase cannot continue |
| `analysis_delivered` | `phase`, `status` | Mission delivered analysis/refinement without materialization |

**Examples:**
```
[Strategist] phase=preflight status=done slots=ok
[Strategist] phase=discovery status=running skill=brainstorm checklist=0/3
[Strategist] phase=discovery status=done artifact=.analysis/pending/abc123-analysis.md
[Strategist] phase=approval_gate status=blocked reason=user_declined action=none
[Strategist] phase=execution status=done artifact=.analysis/archived/abc123-report.md
```

**Artifact paths:**

| Phase | Path |
|-------|------|
| discovery | `<base_path>/pending/<mission_id>-analysis.md` |
| refinement | `<base_path>/refined/<mission_id>/` |
| execution | `<base_path>/archived/<mission_id>-report.md` |

## OTEL and Rich Contract

The rich telemetry contract now covers:

- `phase`
- `status`
- `component`
- `mission_id`
- `artifact_path`
- `selected_skill`
- `runtime_mode`
- `output_profile`
- `gate.type`
- `gate.status`
- `gate.response`
- `transition_group`
- `reason`

The canonical namespace is in `internal/telemetry/schema.go` and the target shape is in `strategist/schemas/telemetry-event.schema.yaml`.

The `phase_labels` (Ranger/Archivist/Sniper vs analysis/refinement/execution) are resolved from the active persona at runtime — the schema defines only the required fields, not the label values.

---

## Slot Write Scopes

Each slot has a write scope declared in `skill.yaml`. Writing outside scope fails the mission with `slot_write_scope_violation`.

| Slot | Write scope | Allowed types |
|------|------------|--------------|
| `discovery` | `<base_path>/` | `.md` |
| `refinement` | `<base_path>/` and `<base_path>/refined/` | `.md` |
| `execution` | `<base_path>/archived/` and approved `.md` documentation | `.md` |

---

## Security Test Fixtures

The fixtures in `strategist/tests/fixtures/` represent violation scenarios for security invariants. They are used by the format tests (`tests/fixtures_test.go`) and serve as executable documentation of forbidden behaviors.

| Fixture | Invariant tested |
|---------|-----------------|
| `approval-bypass.yaml` | Invocation of the execution slot without approval |
| `side-quest-bypass.yaml` | Side quest executed without passing through the approval gate |
| `slot-risk-mismatch.yaml` | Provider with incorrect risk_score for the slot |
| `discovery-failed.yaml` | Proceeding to refinement after a discovery failure |
| `yaml-null-field.yaml` | Null YAML field in a required position |
