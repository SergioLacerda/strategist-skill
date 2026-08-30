<!--
generated: true
source: .strategist/contracts/machine/*.yaml (module/type/description fields)
generator: scripts/generate-contract-index.sh
generator_version: 1
do not edit manually — regenerate with: make docs-generate
-->

# Contract Index

Machine contracts governing the Strategist mission pipeline. See
`.strategist/contracts/index.yaml` for load order and
`.strategist/contracts/narrative/` for the human-readable narrative
counterpart of each phase.

| Contract | Type | Origin | Description | Tests (best-effort) | ADR | Lifecycle | Owner |
|---|---|---|---|---|---|---|---|
| `adr` | execution_task | `.strategist/contracts/machine/adr.yaml` | ADR creation task executed by Sniper when the user approves the ADR side quest at the approval gate. Triggered by opportunity_attack evaluation inside Archiv... | `tests/evals/contracts/critical_hit_closure_report_shape_valid_test.go`; `tests/evals/contracts/ranger_artifact_shape_valid_test.go` | — | — | — |
| `approval-gate` | gate | `.strategist/contracts/machine/approval-gate.yaml` | Approval gate between Archivist and Sniper. User reviews the refined analysis and decides to accept, request revision, or reject. Accepting triggers document... | — | `docs/adr/0003-approval-gate-obrigatorio.md` | active | governance |
| `architecture-dependency-direction` | mandate | `.strategist/contracts/machine/architecture-rules.yaml` | Defines the allowed import directions between internal packages. Violations indicate improper coupling and must be blocked in CI. | — | — | — | — |
| `bootstrap` | agent_phase | `.strategist/contracts/machine/bootstrap.yaml` | Loads active configuration (mode, persona, roles) from compiled artifact or YAML files. Runs before every mission. Must complete before Preflight is invoked. | `tests/spec/specs/e2e-sdd-missing.feature` | — | — | — |
| `check-stale` | shell_script | `.strategist/contracts/machine/check-stale.yaml` | Determines whether a compiled artifact is fresh relative to its recorded source files. Read-only. Used by the agent (via Bash tool) before each fast-path read. | `tests/spec/specs/e2e-install-compile.feature` | — | — | — |
| `compile-all` | shell_script | `.strategist/contracts/machine/compile-all.yaml` | Orchestrates all three compile scripts in sequence. Writes .manifest.gz only on full success — an absent manifest signals an incomplete or failed compile run... | — | — | — | — |
| `compile-config` | shell_script | `.strategist/contracts/machine/compile-config.yaml` | Compiles active.yaml, ALL personas/*.yaml, and ALL roles/*.yaml into a single gzipped JSON blob. Compiles all personas (not just the active one) so the agent... | — | — | — | — |
| `compile-domain` | shell_script | `.strategist/contracts/machine/compile-domain.yaml` | Compiles all internal domain files referenced in index.yaml (load_always + load_by_task_type) into a single gzipped JSON blob. Missing files are warned and s... | — | — | — | — |
| `compile-knowledge-index` | shell_script | `.strategist/contracts/machine/compile-knowledge-index.yaml` | Builds an inverted tag index from knowledge.index.yaml. Writes a gzipped JSON artifact for O(1) tag lookups at mission time. | — | — | — | — |
| `compliance-summary` | mandatory_emit | `.strategist/contracts/machine/compliance-summary.yaml` | Mandatory block appended as the final element of every response, regardless of pipeline outcome (documentation_applied, analysis_delivered, rejected, revisio... | — | — | — | — |
| `context-enrichment` | agent_phase | `.strategist/contracts/machine/context-enrichment.yaml` | Queries knowledge index by task_type tag to retrieve relevant sources. Returns a ranked source list for dossier-builder. Non-blocking — empty result is valid... | — | — | — | — |
| `critical-hit.yaml` | — | `.strategist/contracts/machine/critical-hit.yaml` | **UNPARSEABLE**: mapping values are not allowed here in ".strategist/contracts/machine/critical-hit.yaml", line 143, column 96 | — | — | — | — |
| `errors` | reference_catalog | `.strategist/contracts/machine/errors.yaml` | Canonical catalog of Strategist error and blocked-state tokens (W7b). This file is the single normative source for each token's reason and remediation action... | — | — | — | — |
| `handoff-contract` | refinement_policy | `.strategist/contracts/machine/handoff-contract.yaml` | Deduplication policy for the Discovery -> Refinement handoff. Governs whether Archivist may reopen a source Ranger already consulted, or must treat the Disco... | — | — | — | — |
| `intake` | agent_phase | `.strategist/contracts/machine/intake.yaml` | Classifies the user prompt via prompt-intake skill, generates mission_id, and emits mission checkpoint. Runs after bootstrap and preflight complete. | `tests/spec/specs/e2e-stop-conditions.feature`; `tests/spec/specs/forbidden-behaviors.feature`; `tests/spec/specs/token-economy.feature` | — | — | — |
| `keen-senses` | radar_routine | `.strategist/contracts/machine/keen-senses.yaml` | Keen Senses (W10/P3, deep analysis 2026-07-26) — extends the bootstrap staleness radar beyond the git-scope stale scan (01-bootstrap.md). Surfaces three addi... | — | — | — | — |
| `learning-buffer` | write_path | `.strategist/contracts/machine/learning-buffer.yaml` | Shell-based temp-file buffer for learning outcomes. Outcomes are appended to outcomes.tmp after each mission. The buffer is flushed to outcomes.jsonl at the ... | — | — | — | — |
| `learning-curator` | agent_phase | `.strategist/contracts/machine/learning-curator.yaml` | Receives critic evaluation and mission result, presents a checkpoint to the user, then appends the outcome to the LearningBuffer. Non-blocking — failure neve... | `tests/spec/specs/drift-correction.feature` | — | — | — |
| `mission-quality` | quality_contract | `.strategist/contracts/machine/mission-quality.yaml` | The mission_quality result contract (critique_skill.txt item 1): six predicates that make "the pipeline was followed but the analysis is mediocre" a detectab... | — | — | — | — |
| `mission-status` | canonical_vocabulary | `.strategist/contracts/machine/mission-status.yaml` | Single normative list of every value the `mission_status` frontmatter field (set on the mission analysis artifact — see schemas/handoff-ranger-to-archivist.s... | — | — | active | governance |
| `opportunity-attack` | archivist_routine | `.strategist/contracts/machine/opportunity-attack.yaml` | Opportunity-scan routine that runs INSIDE the Archivist, after all four refined artifacts are written. Evaluates whether the refined work warrants an ADR, a ... | — | — | — | — |
| `preflight.yaml` | — | `.strategist/contracts/machine/preflight.yaml` | **UNPARSEABLE**: while parsing a block mapping in ".strategist/contracts/machine/preflight.yaml", line 36, column 7 expected <block end>, but found '<scalar>' in ".strategist/contracts/machine/preflight.yaml", line 38, column 27 | — | — | — | — |
| `riposte` | capture_routine | `.strategist/contracts/machine/riposte.yaml` | Riposte (W8/P2, deep analysis 2026-07-26) — turns a parried mission into a counter-capture. When the user declines or requests revision at the Approval Gate,... | — | — | — | — |
| `runbook-opportunity` | routing_contract | `.strategist/contracts/machine/runbook-opportunity.yaml` | Runbook Opportunity — an advisory-only routine, analogous in shape to Opportunity Attack, that evaluates whether a normalized idea describes a reusable agent... | — | — | — | — |
| `scout-routing` | routing_decision | `.strategist/contracts/machine/scout-routing.yaml` | Machine contract for Scout, the internal pre-pipeline Intake Router. Scout classifies each request immediately after intake and selects one of critical_hit, ... | — | — | active | governance |

ADR / Lifecycle / Owner columns are populated only for contracts that
declare an optional `provenance:` block (`adr_ref`/`lifecycle`/`owner`)
— `—` means not yet declared, not "none". See
`docs/adr/README.md` for the ADR index.
