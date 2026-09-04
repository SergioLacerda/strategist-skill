<!--
generated: true
source: internal/embed/defaults/schemas/*.yaml (description field)
generator: scripts/generate-schema-index.sh
generator_version: 1
do not edit manually — regenerate with: make docs-generate
-->

# Schema Index

| Schema | Description |
|---|---|
| `active.schema.yaml` | Required fields for `.strategist/active.yaml` preflight validation. |
| `decision.schema.yaml` | Decision ledger entry — critique_skill.txt item 2 and critique_skill_3.txt § Skill/agent gaps item 1. A mission-scoped choice with stable identity, status, and evidence backing. Optional in any given mission's refined package; when present, decisions[] entries are validated against this shape and evaluated by internal/domain's EvaluateMissionQuality (mission_quality.go). See .analysis/done/20260803-critique-skill-affinity-review/design.md § "Consolidated Decision/Evidence Model". |
| `evidence.schema.yaml` | Evidence record — critique_skill.txt item 3's "finding → source → snippet/hash → classification → confidence" chain, refined by the evidence_classes vocabulary proposed in .analysis/todo/v2/pathfinder.txt ("never promote an inference to a fact; cite every historical claim"). Optional in any given mission's refined package; when present, evidence[] entries are validated against this shape and evaluated by internal/domain's EvaluateMissionQuality (mission_quality.go). See .analysis/done/20260803-critique-skill-affinity-review/design.md § "Consolidated Decision/Evidence Model". |
| `fallback-decision.schema.yaml` | Provider-fallback degradation event (ADR-0028) emitted when a configured slot provider is not invocable at mission time and Strategist proceeds using a compatible native role instead. This is a compact, auditable record — NOT a discovery report. It is logged and telemetered via internal/telemetry's AppendFallbackDecisionLine; the implementing skill decides the file/log format. A record is only ever written when a fallback was actually applied — outcome=blocked or no compatible native role never produce one, since nothing degraded. |
| `handoff-archivist-to-sniper.schema.yaml` | Handoff contract from Archivist (refinement) to Sniper (execution). These fields are REQUIRED in the refinement artifact, regardless of format. The implementing skill decides the file format. |
| `handoff-ranger-to-archivist.schema.yaml` | Handoff contract from Ranger (discovery) to Archivist (refinement). These fields are REQUIRED in the analysis artifact, regardless of format. The implementing skill (brainstorming, openspec, custom) decides the file format. Fields may live as YAML frontmatter, embedded sections, or separate files — the orchestrator must be able to locate them. |
| `intake.schema.yaml` | Defines recognized constraint fields and their accepted values with aliases. Used by prompt-intake to extract mission_contract.planning_rules from the user prompt. |
| `mission-result.schema.yaml` | Canonical result statuses for a completed Strategist mission. Documentation-pipeline semantics replace legacy single-mode execution concepts. |
| `outcome-entry.schema.yaml` | Canonical structured shape for historical learning outcomes stored in `.strategist/memory/outcomes.jsonl`. This schema is additive: existing runtime behavior may still emit the minimum fields (`mission_id`, `status`, `timestamp`) while producers converge toward the richer structure below. |
| `progress-contract.yaml` | Defines the mandatory progress event format emitted by Strategist on every phase transition, start, and end. Labels are sourced from the active persona. |
| `response-envelope.schema.yaml` | Canonical envelope for the final Strategist response. Used to keep mission evidence, compliance summary, and mission result in a stable order. |
| `roles.schema.yaml` | Required fields for roles configuration used by Strategist. |
| `scout-route-decision.schema.yaml` | Route decision emitted by Scout (Intake Router) before pipeline branching. This is a compact, auditable classification record — NOT a discovery report. It is logged and telemetered; it is not written as a pending/ analysis artifact. The implementing skill decides the file/log format. |
| `slot-output.schema.yaml` | Output contract validation for discovery and refinement slot providers. |
| `source-card.schema.yaml` | Source card — the retrieval unit delivered to the LLM. Never a raw chunk. Enforces the pattern: evidence → interpretation → impact. |
| `telemetry-event.schema.yaml` | Canonical structured telemetry fields for Strategist mission events. Runtime implementations may emit a subset temporarily, but the contract defines the target shape. |
