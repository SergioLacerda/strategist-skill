# ADR-0053 — Taxonomy Classification Criterion and Sniper Extensibility

**Status:** Accepted
**Date:** 2026-09-25
**Mission:** `20260924-taxonomy-adherence-review`
**Amends:** ADR-0034 (§1 Sniper extensibility, §4 INITIATIVE wording, §5 seven-family nomenclature)
**Related:** ADR-0030, ADR-0035, ADR-0049, ADR-0051, ADR-0054

## Context

ADR-0034 §5 fixed seven families: Roles, Weapons, Abilities, Pipeline Services,
Mechanisms, Routes, Artifacts. The adherence review (`partially_implemented`,
24-row matrix) found four problems with that model:

- It has no classification criterion. Abilities are "reusable mission
  behaviors", and Mechanisms "govern identity, transfer, authorization, or
  integrity". Under that wording INITIATIVE, Opportunity Attack, LEVELING and
  PRECISE-SHOT cannot be placed without contradiction. The same item appears in
  different families across `SKILL.md`, `README.md`, `strategist-concepts.md`,
  `skill.yaml` and ADR-0034.
- ADR-0034 §1 declared Sniper `native/fixed`. The ranked binding
  (`sniper:sniper` in `plugins.lock`, embedded default) already behaves as a
  binding, and `extensibility` is only read inside `internal/domain/role_*.go`.
- Mechanisms exist only as prose. Agents in a mission have no machine-readable
  way to learn which tools they may use, when to use them, or how to invoke them.
- `EvaluatePipelineBypass` (`internal/domain/pipeline_bypass.go`) had no
  non-test caller. Its invariants were `enforced_by: agent_only`, and Scout does
  not check what it checks: Scout selects a route but never verifies that the
  route's phase artifacts exist before a write.

The human made the decisions below at the Approval Gate on 2026-09-24/25
(ledger DEC-001..DEC-008 in the refined `design.md`) and answered the open
questions on 2026-09-25. This ADR records them. It does not reopen them.

## Decision

### 1. Sniper is a native, pluggable Role (amends ADR-0034 §1)

- Sniper is `origin: native`, `extensibility: pluggable`.
- The ranked binding (embedded `sniper`) is the default and behaves as fixed.
  "Fixed" describes the ranked binding state. It is not an attribute of the Role.
- A custom Weapon on Sniper is allowed only when the user explicitly chooses it
  (custom mode, never the ranked default). Bind-time enforcement applies:
  - trust and grants per ADR-0051;
  - the custom Sniper stays documentation-only;
  - its risk is at least `controlled`;
  - the Approval Gate is unchanged.
- The catalog risk `write_analysis` is reconciled with the execution contract
  `controlled`.
- **Declaration and enforcement change together (R-002).** The
  `extensibility: pluggable` flip must not land without the bind-time checks.
- ADR-0034 §4 still holds: there is no silent native fallback.

### 2. Classification criterion (supersedes ADR-0034 §5 where it conflicts)

| Family | Criterion | Members |
| --- | --- | --- |
| Role | Agents acting as personas | Scout, Ranger, Archivist, Sniper |
| Weapon | External skills: bounded skill packages employed by pluggable Roles | `brainstorming`, `openspec-propose` |
| Ability (Feat, pt-BR "Habilidade") | Behavior the agent performs by probabilistic or judgment-based reasoning | Search, Riposte, judgment facets of hybrids |
| Mechanism | Deterministic rule: the outcome is fixed by its inputs, whether enforced in code or in a contract | LEVELING, INITIATIVE, Critical Hit, Opportunity Attack (activation), Approval Gate rules, Side Quest rules, Handoff, Weapon Binding, fingerprint |
| Pipeline | Fixed items, routines and stages | discovery, refinement, Approval Gate stage, materialization, Prompt Intake, Context Enrichment, Scout routes |
| Artifact | Generated items with internal value | ADR, runbook, analysis/refined package, dossier, evidence pack, reports |

Rules:

- The bare word "Ability" means Feat/Habilidade, nothing broader.
- "Pipeline Services" is renamed "Pipeline". **Routes are no longer a family.**
  Routes are Pipeline paths that Scout selects.
- A **hybrid** is modelled as one item with two facets: a judgment facet
  (Ability) and a deterministic facet (Mechanism).
  - PRECISE-SHOT ("TIRO PRECISO" is its pt-BR label, not a second item) is
    hybrid: agent self-assessment plus bounded escalation
    (`internal/initiative/precise_shot*.go`).
  - Opportunity Attack is hybrid: detection is judgment; the activation
    criteria and the gate are deterministic.
- The Approval Gate splits across two families: the stage is Pipeline, and its
  governing rules are Mechanisms. Side Quest follows the same split.
- Critical Hit is a Mechanism. It will be refactored to follow the same
  activation flow as the other Mechanisms. That is an isolated refactor after
  this implementation, with low confidence. Scout's route value `critical_hit`
  stays for now.
- The `main_mission` vs `full_pipeline` naming is deferred to that refactor.
- The registry field `enforcement_kind` (`code | contract | prose`) records how
  each Mechanism is enforced.

### 3. Mechanisms registry

- One machine-readable catalog, `contracts/machine/mechanisms.yaml`, indexed in
  `contracts/index.yaml`.
- Fields: `id`, `family`, `owner_package`, `contract_file`, `enforcement_kind`,
  plus the agent-facing `when_to_use`/`triggers`, `invoked_by` and
  `how_to_invoke`. Optional fields: `summary`, `phase_scope`, and
  `enforcement_tier` (`machine_enforced | machine_observed | agent_only`).
  `enforcement_tier` shows when code exists but is not on a live path.
- Agents in a mission receive a role-scoped view through the `on_start` hook,
  `strategist mechanisms brief --role <role>`, capped at 3200 bytes (M005 token
  economy). The catalog is not listed in `always_load`, and there are no
  per-phase copies (Q-6).
- Purpose: agents learn which tools they have without relying on prose.

### 4. Pipeline-bypass wiring

- `pipeline_bypass.go` is kept and wired, not deleted.
- Scout's route decision becomes the mandatory Route input to
  `EvaluatePipelineBypass`. Scout records it with `strategist mission route`
  (`memory/route-decisions.jsonl`).
- The evaluation runs at the execution boundary, in `strategist mission submit
  --event handoff_challenge_passed`, before any mutation. It cannot run inside
  Scout: Scout runs before any phase artifact exists and must not read
  artifacts.
  - `full_pipeline` requires the analysis, proposal, design and tasks artifacts
    and an approved gate.
  - `implementation_short_route` and `critical_hit` require only an approved gate.
  - With no recorded route decision, the strictest requirement applies.
- A mapping table resolves the route-vocabulary drift.
- `MissionEngine` state is reused as an input, so no second authority is created.
- After wiring, the bypass invariants move from `agent_only` to their real tier.

### 5. ATLAS

- Gem, Scroll and Itinerary are an unimplemented feature. They moved to a DRAFT
  for a future external skill, ATLAS (`.analysis/pending/atlas-skill-draft.md`),
  and were removed from Strategist taxonomy text after the draft existed.
- Evidence Pack stays a Strategist Artifact.

### 6. Questions settled after the gate

- **Q-6 (registry surface):** the `on_start` command and the 3200-byte cap of §3.
- **Q-7 and Q-10 (windows for L2b, L5 and the compat view generator):** there is
  no calendar date. The compat view is retired by a versioned marker
  (ADR-0054). L2b and L5 keep their exit conditions in the refined `tasks.md`
  (items 9.3 and 9.4).
- **Q-8 (L9 hardcoded fallback provider data):** kept. ADR-0035 is not
  overridden. `knownProviderRisk`, `installableDefaultProviders`, the
  `known-providers.yaml` tier and the empty-catalog fallback stay, because more
  than ten tests depend on them and the active Wizard path already blocks
  before reaching them. Revisit only by a new explicit decision.
- **Q-9 (L16 SDD adapter surface, U-005):** the surface is **legacy**, and its
  removal is scheduled as a separate mission. This repository is governed by
  `.providence/`, has no `.sdd/`, and nothing in the Go code or the embedded
  defaults reads `.providence`, so `strategist sync-governance` fails here. The
  removal covers `sync-governance`, `templates/epic-sdd.yaml`, the `sdd-*`
  catalog entries and the `.sdd/` adapter code. The `GovernanceBridge`
  interface of ADR-0024 is a separate question that the removal mission must
  settle. See `.analysis/pending/20260925-l16-sdd-adapter-surface-removal.md`.

### Alternatives rejected

| Alternative | Reason for rejection |
| --- | --- |
| Sniper `native/fixed` as a Role attribute enforced at bind time | Treats a default binding as a Role property |
| Sniper fixed by documentation only | No enforcement at all |
| Keep ADR-0034's seven families and definitions | No usable criterion; INITIATIVE, LEVELING and Opportunity Attack cannot be placed |
| Critical Hit as Route, or as Ability | Its activation is a deterministic rule |
| Keep Mechanisms as documented prose only | Agents stay unaware of available tools |
| Full catalog in `always_load` | Token cost |
| Per-phase copies of the catalog | Breaks the single catalog |
| Delete `pipeline_bypass.go` | Scout does not cover its checks |
| Treat `MissionEngine` as a full substitute for the bypass check | It orders submitted events but never inspects artifacts |
| Evaluate the bypass inside Scout | Every evidence flag is false at Scout's timing, and it contradicts Scout's no-artifact invariant |
| Map Gem/Scroll/Itinerary to jewel/potion/handoff | Rejected in favour of the ATLAS draft |
| Override ADR-0035 to remove the L9 fallback data (Q-8) | Low benefit, wide test churn, the active path is already fail-closed |

## Consequences

### Positive

- Every item gets a family from one testable question: judgment or
  deterministic, and persona, external skill, fixed stage or generated result.
- Hybrids get two facets instead of being forced into one family.
- Agents in a mission can discover Mechanisms, and gaps such as INITIATIVE's
  missing `how_to_invoke` become visible.
- The pipeline-bypass capability moves from prose to a live check.
- The documented Sniper declaration matches the binding model it describes.

### Negative / risks

- **R-002 (high):** a custom Sniper without bind-time enforcement widens
  execution authority. The declaration and the enforcement landed together.
- **Family divergence (closed 2026-09-25):** the prose, `skill.yaml` and the registry now use
  the same six families. The registry field `prose_family` was removed, and a test keeps the
  `skill.yaml` taxonomy lists consistent with the registry families.
- **Bypass wiring** may block flows that relied on prose-only enforcement.
  Both routes are covered by tests.
- One more catalog to maintain.
- Route names stop being a family but remain as values. The
  `selected_route: critical_hit` value now refers to a Mechanism.

### Implementation status (2026-09-25)

Done: Sniper declaration and bind-time enforcement, catalog risk reconciliation,
the Mechanisms registry and its `on_start` brief, the Scout route wiring and the
execution-boundary guard, the ATLAS draft and removal, legacy waves A, B and C
(except the compat view removal, which follows ADR-0054), the ADR-0030
amendment, and the family reclassification of the live documents (task 9.1).

Not done, tracked in the refined `tasks.md`:

- the Critical Hit activation-flow refactor and the `main_mission` vs `full_pipeline` naming
  (task 9.2), an isolated change with low confidence;
- 9.3 (L2b) and 9.4 (L5) exit conditions;
- the removal mission for the L16 SDD adapter surface.

Rule for every change: edit the authoring source first, then regenerate
`.strategist/`, then re-check parity.
