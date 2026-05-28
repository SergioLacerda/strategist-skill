## Why

Complex AI-assisted missions (analyze → plan → execute) have no governed pipeline today. Without it, agents improvise multi-step work inline, skip refinement, execute without approval, and produce untracked artifacts. The Strategist provides a self-contained orchestration tool that enforces Scout → Engineer → approval gate → Hunter as a governed sequence — deployable standalone or integrated with SDD governance.

## What Changes

- **New**: Strategist skill — curl-distributed standalone orchestrator with pluggable Scout, Engineer, and Hunter slots; silent install by default; TUI wizard via `--wizard`; zero config files in target repo
- **New**: Two operational modes — `pragmatic` (analytical tone, phases: analysis/refinement/execution) and `epic` (strategic tone, phases: scout/engineer/hunter); same pipeline, different persona; configurable at install, overrideable per-mission with `--mode`
- **New**: External knowledge index (`knowledge.index.yaml`) — multi-source registry with id/path/tags/priority; Strategist queries by task_type tags before each prompt; `source-hints.yaml` overlays learned priority preferences
- **New**: Two-file learning cache — `memory/outcomes.jsonl` (mission results) and `memory/source-hints.yaml` (source quality annotations); both require explicit human approval before writing; approvals are independent
- **New**: `sdd-engineer` skill — the default Engineer slot provider (genuine gap in current skill registry)
- **New**: `<base_path>/.strategist/` workspace domain — AI-first identity, directives, rubrics, and patterns; path follows injected or configured base; lives in target repo (not in skill root)
- **New**: `personas/` directory — `pragmatic.yaml` and `epic.yaml` persona configs; `templates/` directory with pre-built install configs
- **New**: `roles/` configuration files — declarative slot bindings; `active.yaml` stores current config
- **New**: Learning Loop skills — `prompt-intake`, `context-enrichment`, `dossier-builder`, `response-critic`, `learning-curator`
- **New**: `intake.schema.yaml` — prompt constraint extraction (delivery_strategy, legacy_compatibility, execution_intent)
- **New**: `progress-contract.yaml` — structured progress event format using persona-specific phase labels
- **Optional integration**: When SDD's `sdd-analysis-plugin-protocol` is in place, Strategist registers as an SDD plugin and receives `base_path`, `execution_provider`, `knowledge_paths`, and `governance_context` via injection

## Distribution and Deployment Modes

### Installation
```bash
# Silent (defaults, no prompts):
curl -sL strategist.run | sh

# Interactive TUI wizard:
curl -sL strategist.run | sh --wizard
```

The wizard offers pre-built templates: `pragmatic-standalone`, `epic-standalone`, `epic-sdd`. All configuration lives inside the skill root — **the target repository receives only the mission workspace** (`<base_path>/todo|pending|refined|done/.strategist/`). Zero config files in the repo root.

### Standalone Mode (no SDD dependency)
Strategist runs independently in any repository using `active.yaml` (generated at install). Slot providers (Scout/Engineer/Hunter) are configured from the selected template or TUI. External knowledge sources registered in `knowledge.index.yaml` (also in skill root).

### SDD Integration Mode (optional)
When registered as an SDD plugin via `sdd-analysis-plugin-protocol`, SDD injects:
- `base_path` → overrides `active.yaml` base
- `execution_provider` → overrides Hunter slot
- `knowledge_paths` → appends SDD compiled docs (e.g. `.sdd/docs`) to Strategist's knowledge pool
- `governance_context` → metadata for metrics and audit

## Capabilities

### New Capabilities

- `strategist-orchestrator`: Tool that routes missions through Scout, Engineer, approval gate, and Hunter slots; validates providers; emits progress events; manages artifact state
- `slot-providers`: Pluggable slot system — each of the three slots (Scout/Engineer/Hunter) accepts any provider that satisfies the slot's risk contract; bindings declared in `roles/<config>.yaml`; public labels Scout/Engineer/Hunter map to internal generic slot names (discovery/refinement/execution) for contract compatibility
- `mission-intake`: Extracts structured constraints (delivery_strategy, legacy_compatibility, execution_intent) from user prompt before pipeline starts; applies them to mission contract passed to all providers
- `strategist-internal-domain`: AI-first self-learning knowledge base at `<base_path>/.strategist/` — loaded selectively via `index.yaml`; contains identity, drift patterns, directives, rubrics, patterns, and memory
- `learning-loop`: Five-skill subsystem (prompt-intake → context-enrichment → dossier-builder → response-critic → learning-curator) that enriches context before missions and records human-approved outcomes after
- `sdd-engineer-skill`: New read-only Engineer slot skill that transforms discovery artifacts into implementation-ready plans with tasks, subitems, technical details, Do/Don't guidance, and execution checklist

### Optional SDD Integration Capabilities

- `sdd-plugin-registration`: Strategist can register in `.sdd/plugins/registry.yaml` as `analysis_orchestrator` when SDD's `sdd-analysis-plugin-protocol` is deployed
- `sdd-knowledge-injection`: When SDD-integrated, receives compiled governance docs as additional knowledge paths — Strategist consults them before LLM calls

## Impact

- `<base_path>/.strategist/`: new directory — Strategist's private workspace (standalone: under `.analysis/`; SDD-integrated: under `.sdd/analysis/`)
- `<skill_root>/strategist/`: new skill directory with `roles/`, `intake.schema.yaml`, `progress-contract.yaml`
- `<skill_root>/sdd-engineer/`: new skill directory
- `<skill_root>/prompt-intake/`, `context-enrichment/`, `dossier-builder/`, `response-critic/`, `learning-curator/`: new Learning Loop skill directories
- No breaking changes to existing skills, governance, or CLI
- **No hard dependency on `sdd-analysis-plugin-protocol`** — SDD integration is optional and additive
