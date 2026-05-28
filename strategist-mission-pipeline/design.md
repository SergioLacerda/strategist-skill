## Context

Strategist is a standalone orchestration tool distributed as a skill and deployable in any repository. It coordinates multi-phase missions through three pluggable slots (Scout → Engineer → Hunter) without being coupled to any specific provider or governance framework.

It can optionally integrate with SDD as a registered plugin — in that mode, SDD injects base_path, execution_provider, knowledge_paths, and governance_context. But this is not a prerequisite: Strategist is fully functional standalone.

Key design decisions from all analysis sessions:
- Distributed via curl (`curl -sL strategist.run | sh`); silent by default, `--wizard` for TUI setup
- All config lives inside the skill (`active.yaml`, `personas/`, `roles/`, `memory/`, `knowledge.index.yaml`); the target repo's footprint is the mission workspace only
- Two operational modes (pragmatic / epic) sharing the same pipeline but with different tone, vocabulary, and phase labels
- External knowledge index (`knowledge.index.yaml`) with multi-source support, queried by task_type tags before each prompt
- Learning cache (`memory/outcomes.jsonl` + `memory/source-hints.yaml`) updated only with explicit human approval after each mission

## Goals / Non-Goals

**Goals:**
- Implement Strategist as a standalone skill with curl-based distribution
- Implement TUI wizard (`--wizard` flag) for interactive setup; silent defaults for standard install
- Implement two operational modes: pragmatic and epic — same pipeline, different persona
- Implement external knowledge index (`knowledge.index.yaml`) with multi-source querying per task_type
- Implement the slot contract validation (preflight checks on provider skill.yaml)
- Implement intake parsing that extracts mission constraints from user prompt
- Implement progress event emission on every phase transition using mode-specific labels
- Implement the mandatory approval gate before Hunter slot
- Implement `<base_path>/.strategist/` internal domain with selective loading (domain stays in target repo workspace, all skill config stays in skill root)
- Implement `sdd-engineer` as the default Engineer slot provider
- Implement the Learning Loop as a five-skill subsystem with two-file approval-gated cache
- Implement optional SDD integration: registration, sdd_injection handling, knowledge_paths consumption

**Non-Goals:**
- Requiring SDD or any governance framework to be present
- Writing any config files to the target repository (only workspace artifacts)
- Implementing sdd-ask executor mode (that is an SDD concern, covered by `sdd-analysis-plugin-protocol`)
- Building a UI or interactive dashboard for mission progress

## Decisions

### Decision 1 — Distribution via curl; silent install by default, TUI wizard opt-in

**Choice**: `curl -sL strategist.run | sh` installs with defaults (no prompts). `--wizard` flag enables the TUI for interactive setup. Generated `active.yaml` lives inside the skill root — nothing written to the target repo.

**Rationale**: Silent install reduces setup friction for experienced users who know what they want. TUI provides a discovery path for new users or when switching contexts. Keeping all config inside the skill means target repos have zero config pollution.

**Alternative**: Require TUI on every install. Rejected — adds friction for automation and CI contexts; `--wizard` flag cleanly separates the two experiences.

---

### Decision 2 — Two operational modes declared in personas/; same pipeline, different voice

**Choice**: `personas/pragmatic.yaml` and `personas/epic.yaml` define phase labels, tone directives, and prompt templates. Mode configured in `active.yaml`; overrideable per-mission with `--mode`. Pipeline logic is identical in both modes.

```
Pragmatic: analysis → refinement → execution
Epic:      scout    → engineer   → hunter
```

**Rationale**: The pipeline's value is in the sequence and the approval gate — not in the vocabulary. Separating personality from pipeline means we can evolve each independently. Adding new modes later (e.g., `stealth`, `audit`) requires only a new persona file.

**Alternative**: Two separate skill.yaml configs for each mode. Rejected — duplicates pipeline logic and creates drift risk between modes.

---

### Decision 3 — Init flow configures base_path, slot providers, and active persona once

**Choice**: On install (or `--wizard`), user selects a template from `templates/` (e.g., `epic-standalone.yaml`, `pragmatic-sdd.yaml`) or customizes interactively. Result saved to `active.yaml`. Override per-mission with `--mode` and `--roles` flags.

**Rationale**: Pre-built templates cover the most common configurations; users can start immediately. `active.yaml` is the single source of truth for the agent. When SDD-integrated, SDD injection overrides Hunter slot and adds knowledge_paths — `active.yaml` remains as fallback.

**Alternative**: Prompt every mission. Rejected — per-mission prompts add friction; `--mode` and `--roles` flags cover the override case cleanly.

---

### Decision 4 — External knowledge index: multi-source, queried by task_type tags

**Choice**: `knowledge.index.yaml` (in skill root) registers multiple knowledge sources, each with `id`, `path`, `type`, `tags`, and `priority`. Before each mission prompt, `context-enrichment` queries the index by `task_type` tags and loads relevant excerpts. `source-hints.yaml` overlays learned priority adjustments from past missions.

```yaml
# knowledge.index.yaml
sources:
  - id: project-architecture
    type: docs
    path: /abs/path/to/docs/architecture
    tags: [architecture, system-design]
    priority: high
  - id: past-examples
    type: examples
    path: .strategist/patterns/good
    tags: [examples, patterns]
    priority: medium
```

**Rationale**: A single indexed file is easy to maintain and version. Tag-based querying keeps the dossier minimal — only sources relevant to the current task_type are loaded. `source-hints.yaml` closes the learning loop on source quality without modifying the index directly.

**Alternative**: Per-task-type index files. Rejected — maintenance burden grows with task types; tags on a single index are equally expressive.

---

### Decision 5 — Two-file learning cache; both require explicit human approval

**Choice**: After mission completion, `learning-curator` proposes updates to two files: `memory/outcomes.jsonl` (mission result record) and `memory/source-hints.yaml` (source quality annotations). Both are presented for review before writing. User approves both, rejects both, or reviews individually.

**Rationale**: Separating outcomes from source-hints allows independent approval — a user may want to record the mission outcome but disagree with a suggested source priority change. Append-only `outcomes.jsonl` ensures auditability; `source-hints.yaml` is the active inference layer that improves future knowledge retrieval.

**Alternative**: Single cache file. Rejected — conflates mission memory with source preference, making each harder to review and audit independently.

---

### Decision 6 — Slot contract validation at preflight, not at invocation

**Choice**: Strategist validates all slot providers during preflight (before any slot executes). If any provider fails validation, the pipeline stops before starting.

**Rationale**: Failing fast at preflight avoids partial execution — discovering a risk mismatch after Scout runs but before Engineer would leave an orphan artifact and an unclear state. Preflight validates `risk_score`, `status`, and `schema_version` for all three slot providers at once.

**Alternative**: Validate each provider immediately before invoking it. Rejected — partial execution state is harder to recover from and harder to explain to the user.

---

### Decision 7 — roles/<config>.yaml as the slot binding mechanism; Scout/Engineer/Hunter as public labels

**Choice**: Slot provider bindings are declared in `roles/<config>.yaml` files. Public labels are Scout (discovery), Engineer (refinement), Hunter (execution). Internal contracts use generic names (discovery/refinement/execution) for compatibility. When SDD-integrated, the Hunter (execution) provider is always overridden by `sdd_injection.execution_provider`.

**Rationale**: Separating binding configs from the skill manifest allows multiple mission configurations without changing Strategist's core logic. Using public names (Scout/Engineer/Hunter) in all user-facing output and progress events makes the pipeline semantics intuitive while keeping internal contracts generically compatible.

**Alternative**: Hardcode providers in skill.yaml. Rejected — makes Strategist SDD-specific and prevents reuse across different tool ecosystems.

---

### Decision 8 — Internal domain loaded via index.yaml (selective loading)

**Choice**: `<base_path>/.strategist/index.yaml` is the only file always loaded. It maps task_type to the specific directive and rubric files needed. Full domain is never loaded into context.

**Rationale**: The internal domain grows over time (examples, lessons, rubrics). Loading it fully on every mission would create a token budget problem. The index enables the agent to load only the 2–4 files relevant to the current task_type, keeping context minimal.

**Alternative**: Load all `.strategist/` files every mission. Rejected — creates unnecessary context bloat; in SDD integration, conflicts with M005 (Token Economy Enforcement).

---

### Decision 9 — Learning loop is parallel, not blocking

**Choice**: Learning Loop (prompt-intake → context-enrichment → dossier-builder → response-critic → learning-curator) runs around the main pipeline. It enriches context before and records outcomes after, but its failure does not block mission execution.

**Rationale**: Learning Loop is an optimization layer. If `context-enrichment` finds no relevant examples, the mission continues without them. If `learning-curator` fails to write an outcome, the mission result is still valid. Making it blocking would add fragility for marginal gain.

**Alternative**: Block mission if context-enrichment fails. Rejected — degrades system reliability for a non-critical enhancement.

---

### Decision 10 — drift-patterns.yaml for self-correction without re-reading governance

**Choice**: `identity/drift-patterns.yaml` lists known agent drift behaviors with a `symptom` and `correction` for each. The agent reads this at preflight and can self-correct without consulting `.sdd/` mandates.

**Rationale**: The agent's most common failure modes are predictable (executing slot work directly, bypassing approval gate, writing artifacts outside base_path). Encoding corrections in a compact, always-loaded file means the agent can catch and fix these without adding governance reads to the hot path.

**Alternative**: Reference governance files at runtime for corrections. Rejected — governance files are large and loading them on every mission is expensive; they are also external to Strategist's own domain.

---

### Decision 11 — sdd-engineer as a focused, read-only skill

**Choice**: `sdd-engineer` is a standalone skill in the skill registry with `risk_score: read_only`. It takes a discovery artifact as input and produces a reviewed plan with tasks, subitems, technical details, design, goals/non-goals, Do/Don't guidance, and an execution checklist.

**Rationale**: No existing SDD skill covers implementation-ready plan generation. `sdd-diagnose` diagnoses problems; `sdd-review-architecture` reviews architecture. Neither produces structured task breakdowns for execution. `sdd-engineer` fills this gap as a first-class read-only skill.

**Alternative**: Extend sdd-diagnose with a planning mode. Rejected — planning and diagnosis are distinct cognitive modes; mixing them would make sdd-diagnose harder to use for pure diagnostic requests.

## Risks / Trade-offs

**[Risk] Provider skill.yaml not found during preflight** → Mitigation: Preflight emits `[SDD] phase=preflight status=blocked reason=slot_provider_not_found` with the missing provider id and resolution path. Mission does not start.

**[Risk] Agent ignores drift-patterns.yaml and drifts anyway** → Mitigation: `identity/what-i-am.yaml` also lists core_invariants as a redundant check. Both files are in `load_always` — either one catching the drift is sufficient.

**[Risk] Learning memory contaminated by bad outcomes** → Mitigation: `learning-curator` MUST NOT write to memory without the user's explicit approval via the learning checkpoint prompt. This is enforced by a `forbidden` rule in the skill contract.

**[Risk] Large `.strategist/` domain slows down context loading** → Mitigation: `index.yaml` selective loading caps the hot-path reads at 2–4 files. Patterns and memory are `load_on_demand` only.

**[Risk] SDD integration mode misused as standalone** → Mitigation: Strategist explicitly documents both modes. `roles/default.yaml` (standalone) and `roles/sdd-mission.yaml` (SDD integration) are separate configs. SDD injection overrides Hunter slot and adds knowledge_paths but does not remove standalone defaults.

## Migration Plan

**Phase 1 — Standalone (no SDD dependency):**
1. Create `<skill_root>/strategist/` with skill.yaml, SKILL.md, protocol.md, intake.schema.yaml, progress-contract.yaml, roles/
2. Create `<skill_root>/sdd-engineer/` with skill.yaml and SKILL.md
3. Create `<skill_root>/{prompt-intake,context-enrichment,dossier-builder,response-critic,learning-curator}/` skill files
4. Update skill registry (if applicable) with all new skill entries
5. On first `strategist init`: generate `<base_path>/todo|pending|refined|done/` + `<base_path>/.strategist/` with index.yaml and identity/

**Phase 2 — SDD integration (optional, after `sdd-analysis-plugin-protocol` is deployed):**
6. Register Strategist in `.sdd/plugins/registry.yaml` with full sdd_injection block
7. Verify plugin passes `sdd plugin validate strategist`

Rollback: Remove Strategist plugin entry from registry (SDD integration disabled). Standalone artifacts remain functional — they have no dependency on the registry entry.

## Open Questions

None. All decisions resolved during brainstorming, critique, and explore sessions (2026-05-27).
