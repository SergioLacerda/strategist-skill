# Strategist Mission Pipeline — Design Spec
<!-- revised: 2026-05-27 — plugin model adopted from strategist_critique.md -->

## Summary

A configurable orchestration plugin that coordinates a governed pipeline of three pluggable slots (discovery → refinement → execution) with a mandatory approval gate, structured progress feedback, and an internal self-learning domain.

The Strategist is a **SDD analysis provider plugin** — autonomous, externally registered, and indifferent to where artifacts live. SDD injects the base path and execution provider at registration time. The Strategist names the injected execution provider its **Hunter slot**.

---

## Architecture

```
User request
    ↓
sdd-ask [router mode]
    ↓ detects: analysis/planning mission
    ↓ delegates to: registered analysis_provider (Strategist)
    ↓
Strategist
  ├── preflight       → reads .strategist/index.yaml, validates slots
  ├── intake          → extracts constraints from prompt
  ├── slot: discovery   (risk: read_only)     → <base>/pending/
  ├── slot: refinement  (risk: read_only)     → <base>/refined/
  ├── approval gate   (mandatory, always)
  └── slot: execution   (risk: controlled_write) → <base>/done/
        └── provider = execution_provider injected by SDD
                ↓
        sdd-ask [executor mode] — executes reviewed tasks directly
```

### sdd-ask Dual Mode

`sdd-ask` operates in two distinct modes depending on input type:

| Mode | Triggered by | Behavior |
|---|---|---|
| Router | `input.type = user_prompt` | classify intent → route to provider |
| Executor | `input.type = reviewed_task` | execute directly, no re-routing |

When Strategist invokes the Hunter slot, it passes `reviewed_task` — sdd-ask enters executor mode and does not re-route.

### Pluggable Providers

Every slot accepts any provider that satisfies the slot contract. Bindings are declared in `roles/<config>.yaml` and can be overridden by SDD injection.

| Slot | Risk required | Example providers |
|---|---|---|
| discovery | `read_only` | `brainstorm`, `sdd-diagnose`, any read-only skill |
| refinement | `read_only` | `openspec`, `sdd-engineer`, any read-only skill |
| execution | `controlled_write` | `sdd-ask` (SDD default), `caveman`, `stabilize` |

Example role configurations:

| Config | Discovery | Refinement | Execution |
|---|---|---|---|
| `sdd-mission` | `sdd-diagnose` | `sdd-engineer` | injected by SDD |
| `spec-driven` | `brainstorm` | `openspec` | injected by SDD |
| `quick-analysis` | `brainstorm` | `brainstorm` | injected by SDD |

The execution slot provider is always determined by SDD injection, never hardcoded in `roles.yaml`. This ensures SDD maintains execution authority.

---

## SDD Integration

The Strategist is registered in `.sdd/plugins/registry.yaml`, not in the skill registry. SDD injects configuration at invocation time.

```yaml
# .sdd/plugins/registry.yaml (entry for Strategist)
- id: strategist
  type: analysis_orchestrator
  config:
    base_path: .sdd/analysis        # SDD owns this path
    execution_provider: sdd-ask     # SDD determines who executes
```

SDD contract injected into Strategist at runtime:

```yaml
sdd_injection:
  base_path: .sdd/analysis
  execution_provider: sdd-ask
  approval_gate: required           # SDD enforces this
  artifact_schema: mission-contract.schema.json
  result_schema: mission-result.schema.json
```

The Strategist maps `execution_provider` → Hunter slot internally. SDD has no knowledge of the Hunter metaphor — it only knows it injected an execution provider.

---

## Artifact State Machine

Base path is provided by SDD (default: `.sdd/analysis`). Four state directories:

```
<base>/
├── todo/      ← queued inputs
├── pending/   ← discovery slot artifacts
├── refined/   ← refinement slot artifacts (ready for execution)
└── done/      ← completed missions
```

Artifact flow:

```
todo/<mission-id>.md
    ↓ discovery executes
pending/<mission-id>-discovery.md
    ↓ refinement executes
refined/<mission-id>-plan.md
    ↓ approval gate + execution
done/<mission-id>-plan.md
```

Artifact naming: `mission-{YYYY-MM-DD}-{sequence}-{phase}.md`

---

## Strategist Plugin Contract

The Strategist must declare this contract to be accepted by SDD:

```yaml
id: strategist
type: analysis_orchestrator
version: 1.0.0

capabilities:
  - discovery
  - refinement
  - execution_orchestration

risk:
  max: controlled_write

input_contract: mission-contract.schema.json
output_contract: mission-result.schema.json

uses:
  sdd_base_path: true
  sdd_execution_provider: true
  sdd_approval_gate: true
  sdd_progress_events: true

forbidden:
  - bypass_sdd_approval_gate
  - write_outside_sdd_base_path
  - use_execution_provider_other_than_injected
  - create_parallel_governance_structure
```

---

## Roles Configuration

Each `roles/<config>.yaml` declares discovery and refinement providers. The execution provider is always injected by SDD — it is declared in roles only as a documentation hint, never as a binding.

```yaml
# .sdd/skills/strategist/roles/sdd-mission.yaml
version: "1.0.0"
description: "SDD governed mission: sdd-diagnose → sdd-engineer → [sdd-injected]"

slots:
  discovery:
    provider: sdd-diagnose
    risk: read_only

  refinement:
    provider: sdd-engineer
    risk: read_only

  execution:
    provider: _injected_by_sdd     # resolved at runtime from SDD config
    risk: controlled_write
    requires_approval: true
```

### Slot Provider Validation (preflight)

Strategist validates `skill.yaml` of each declared provider before starting:

| Check | Discovery | Refinement | Execution |
|---|---|---|---|
| `risk_score == read_only` | required | required | — |
| `risk_score == controlled_write` | — | — | required |
| `status == active` | required | required | required |
| `schema_version` present | required | required | required |

Provider resolution order:
1. `.sdd/skills/<provider>/skill.yaml`
2. `.claude/skills/<provider>/skill.yaml`
3. `registry.json` entry → `skill_yaml` path

If preflight fails:
```
[SDD] phase=preflight status=blocked
      reason=slot_risk_mismatch slot=refinement
      provider=openspec expected=read_only found=controlled_write
```

---

## Intake Parsing

Extracted from user prompt before invoking any slot. Applied to `mission_contract.planning_rules`.

| Field | Values | Default | Detected aliases |
|---|---|---|---|
| `delivery_strategy` | `sprint` / `total` | `sprint` | "sem prazo", "big bang", "incremental" |
| `legacy_compatibility` | `required` / `not_required` | `required` | "sem retrocompatibilidade", "pode quebrar" |
| `execution_intent` | `plan_only` / `execute_after_approval` | `execute_after_approval` | "só análise", "só plano" |

Rules:
- Confidence ≥ 0.65 (LLM-assessed match against known aliases) → use without asking
- Field absent → use default
- Conflicting values detected → stop and ask user

`task_type` (e.g., `architecture_analysis`, `refactor`) is a separate classification derived by `prompt-intake` from the user prompt. It drives knowledge loading in `.strategist/index.yaml` (`load_by_task_type`) and is independent from the intake constraint fields above.

Extracted constraints become `mission_contract.planning_rules` and are passed to all slot providers.

---

## Progress Events

Every phase transition emits a structured event. No internal reasoning exposed.

```
[SDD] phase=preflight        status=done   slots=ok provider=strategist
[SDD] phase=intake           status=done   delivery=sprint legacy=required
[SDD] phase=discovery        status=running skill=sdd-diagnose checklist=2/6
[SDD] phase=discovery        status=done   artifact=.sdd/analysis/pending/mission-001-discovery.md
[SDD] phase=refinement       status=running skill=sdd-engineer  checklist=3/8
[SDD] phase=refinement       status=done   artifact=.sdd/analysis/refined/mission-001-plan.md
[SDD] phase=approval_gate    status=waiting
[SDD] phase=execution        status=running skill=sdd-ask       checklist=1/7
[SDD] phase=execution        status=done   artifact=.sdd/analysis/done/mission-001-plan.md
```

On blocker:
```
[SDD] phase=refinement status=blocked checklist=1/8
      reason=missing_discovery_artifact
      action=re-run discovery or provide artifact at .sdd/analysis/pending/
```

---

## Internal Domain — `.strategist/`

The Strategist's self-contained knowledge base. Lives adjacent to the project (or inside `<base>/`). Independent from `.sdd/` governance. Optimized for selective, minimal loading.

```
<base>/.strategist/
├── index.yaml              ← read first, always
│
├── identity/
│   ├── what-i-am.yaml      ← invariants
│   └── drift-patterns.yaml ← self-correction patterns
│
├── directives/
│   ├── core.yaml
│   └── by-task/
│       └── <task-type>.yaml
│
├── rubrics/
│   └── <task-type>.yaml
│
├── patterns/
│   ├── good/
│   └── bad/
│
└── memory/
    ├── lessons.yaml        ← human-curated
    └── outcomes.jsonl      ← runtime-generated
```

### `index.yaml`

```yaml
version: "1.0.0"
agent_domain: strategist

load_always:
  - identity/what-i-am.yaml
  - directives/core.yaml

load_by_task_type:
  architecture_analysis:
    - directives/by-task/architecture-analysis.yaml
    - rubrics/architecture-analysis.yaml
  refactor:
    - directives/by-task/refactor.yaml
    - rubrics/refactor.yaml

load_on_demand:
  patterns: patterns/
  memory: memory/lessons.yaml
  outcomes: memory/outcomes.jsonl
```

### `identity/what-i-am.yaml`

```yaml
i_am:
  - a slot orchestrator
  - a pipeline configurator
  - a convergence enforcer
  - a progress reporter

i_am_not:
  - a discovery executor
  - a refinement executor
  - a code modifier
  - a governance authority
  - an SDD core component

core_invariants:
  - never execute slot work directly
  - never advance phase without artifact
  - never invoke execution without user approval
  - always emit progress events
  - always use execution_provider injected by SDD
  - always write artifacts under SDD base_path
```

### `identity/drift-patterns.yaml`

```yaml
drifts:
  - id: direct_execution
    symptom: "Performing discovery or refinement instead of delegating to slot provider"
    correction: "Stop. Identify active slot. Invoke provider. Resume."

  - id: silent_phase_advance
    symptom: "Moving to next phase without emitting progress event or persisting artifact"
    correction: "Emit event. Persist artifact. Then advance."

  - id: approval_bypass
    symptom: "Invoking execution slot without asking user"
    correction: "Stop execution. Return to approval_gate. Ask user."

  - id: scope_expansion
    symptom: "Planning tasks outside the mission's declared scope"
    correction: "Flag scope expansion. Return to user. Do not expand."

  - id: execution_provider_override
    symptom: "Using a different execution provider than the one injected by SDD"
    correction: "Restore execution_provider from SDD injection. Do not override."
```

### `directives/core.yaml`

```yaml
always:
  - read index.yaml first
  - validate slot contracts before pipeline starts
  - emit [SDD] event on every phase transition
  - persist artifact before advancing phase
  - use execution_provider from SDD injection
  - write artifacts under SDD-provided base_path

never:
  - load full knowledge base into context
  - invent slot providers not declared in roles.yaml
  - skip approval gate
  - modify files during discovery or refinement phases
  - override SDD-injected execution_provider
```

---

## Learning Loop

A parallel subsystem. Enriches context before each mission, records outcomes after.

### Flow

```
Prompt
  ↓ prompt-intake      → classify intent, task_type, risk
  ↓ context-enrichment → query knowledge_paths for rubrics, patterns, lessons
  ↓ dossier-builder    → build minimal dossier for LLM slot provider
  ↓ LLM executes via slot
  ↓ response-critic    → evaluate against rubric
  ↓ user verdict       → accept / reject / correct
  ↓ learning-curator   → record to .strategist/memory/ (only if approved)
```

### Knowledge Paths

Configurable via `--knowledge` parameter. Multiple paths merged, deduplicated by `id`.

```
--knowledge .sdd/analysis/.strategist   ← project learning (runtime)
--knowledge .team/best-practices        ← team-curated base (version-controlled)
```

Expected structure at any knowledge path:

```
<knowledge_path>/
├── examples/good/
├── examples/bad/
├── rubrics/
└── directives/
```

### Minimal Dossier Format

```md
[SDD DOSSIER]
task_type: architecture_analysis
delivery: sprint | legacy: required

directives (non-negotiable):
- Always cite evidence before recommendation
- Never propose broad rewrite without scoped evidence

good_examples: 1 matched
- auth-boundary-analysis.md (pattern: layer boundary violation)

bad_examples: 1 matched
- vague-refactor.md (avoid: advice without evidence)

output_template: Summary / Findings / Evidence / Risks / Recommendation / Confidence

rubric: architecture_analysis (threshold: 0.75)
```

### Learning Governance Rule

The agent must not promote any output to learning memory without explicit human approval.

```
[SDD LEARNING]
Mission result:
[ ] accepted        [ ] accepted with adjustments
[ ] rejected        [ ] do not record

Record as:
[ ] good example    [ ] bad example    [ ] update rubric
```

---

## Stop Conditions

```yaml
stop_conditions:
  - missing_sdd_injection          # base_path or execution_provider not provided
  - missing_roles_config
  - slot_provider_not_found
  - slot_risk_mismatch
  - missing_artifact
  - low_confidence
  - ambiguous_scope
  - governance_conflict
  - execution_approval_denied
```

---

## Registration Summary

The Strategist is registered in `.sdd/plugins/registry.yaml` (not the skill registry).

Skills created as companions (in skill registry):

| Skill | Category | Risk | Note |
|---|---|---|---|
| `sdd-engineer` | refinement | read_only | new — genuine gap, default refinement provider |
| `prompt-intake` | learning | low | new — Learning Loop |
| `context-enrichment` | learning | low | new — Learning Loop |
| `dossier-builder` | learning | low | new — Learning Loop |
| `response-critic` | learning | low | new — Learning Loop |
| `learning-curator` | learning | low | new — Learning Loop |

All 8 existing skills remain unchanged. `sdd-ask` gains a documented executor mode (no contract change required — input type differentiation only).

---

## Non-Goals

- Strategist does not replace `sdd-ask` for simple single-skill requests
- Strategist does not implement discovery, refinement, or execution logic directly
- Learning Loop does not learn autonomously — all memory updates require human approval
- `.strategist/` domain does not duplicate or override `.sdd/` governance
- Strategist does not determine the execution provider — SDD does

---

## Open Questions

None. All design decisions resolved during brainstorming sessions (2026-05-27).
