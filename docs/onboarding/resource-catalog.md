# Strategist Resource Catalog

This catalog is the onboarding reference for Strategist's current resource model. It uses the canonical seven-family taxonomy:

| Family | What it answers |
| --- | --- |
| Role | Who owns a responsibility and checkpoint? |
| Weapon | Which bounded provider package performs work for a Role or slot? |
| Feat | Which reusable capability or tactical action is available? |
| Tool | Which operational utility supports the pipeline? |
| Mechanism | Which governance or runtime control enforces behavior? |
| Stage | Which phase or route is active? |
| Artifact | Which durable file, record, or generated result carries state or evidence? |

The registry filename `mechanisms.yaml` is a compatibility detail. It contains rows from multiple canonical families and should not be read as a six-family public taxonomy.

## Evidence and status vocabulary

Inventory is not availability. Each entry must be interpreted with its evidence state:

- **Active** — part of the current configured pipeline or identity-bearing runtime.
- **Bound** — selected for a Role/slot by the active configuration and lock data.
- **Cataloged** — present in the source or compiled catalog, but not necessarily selected.
- **Candidate** — available for a future or opt-in binding, without current active use.
- **Internal** — pipeline implementation support, not a user-bindable Weapon.
- **Proposed/history** — useful as architecture or historical context, not a current runtime promise.

`strategist check --json` proves static preflight and binding readiness. It does not by itself prove authenticated live-provider invocation or capability isolation.

## Roles

Roles own responsibility, checkpoints, and handoff semantics. A Role is not automatically a Weapon.

| Role | Responsibility | Current state |
| --- | --- | --- |
| **Scout** | Fixed pre-pipeline intake router; classifies the request and selects the route. | Active |
| **Ranger** | Discovery: gathers evidence, maps scope, identifies uncertainties, and produces the discovery artifact. | Active; bound to `brainstorming` for discovery |
| **Archivist** | Refinement: turns discovery into a reviewable proposal, design, task list, evidence envelope, and handoff. | Active; bound to `openspec-propose` for refinement |
| **Sniper** | Controlled execution: materializes only accepted documentation targets and writes the completion report. | Active native execution Role |

The current identity-bearing Role chain is:

```text
Scout → Ranger → Archivist → Approval Gate → Sniper
```

Approval Gate is a mandatory governance checkpoint between refinement and execution; it is not owned by a Weapon.

## Weapons

Weapons are bounded provider packages. Every Role-bindable skill package is treated as a Weapon, with independent provenance (`embedded` or `custom`) and runtime identity (`embedded`, `host`, `executable`, or `openspec_root`).

### Current binding roster

| Slot | Role | Weapon | Runtime/evidence state |
| --- | --- | --- | --- |
| Discovery | Ranger | `brainstorming@1.0.0` | Active Ranked binding; embedded provider |
| Refinement | Archivist | `openspec-propose@1.0` | Active Ranked binding; private OpenSpec root |
| Execution | Sniper | Native Sniper binding | Active native Role binding; not an external catalog Weapon |

### Cataloged and candidate Weapons

| Weapon | Intended use | Status |
| --- | --- | --- |
| `brainstorming@1.0.0` | Discovery exploration and evidence gathering | Active bound Weapon |
| `openspec-propose@1.0` | OpenSpec-based refinement and planning artifacts | Active bound Weapon |
| `openspec-explore@1.0` | Exploratory discovery option | Cataloged candidate; not the current discovery binding |
| `writing-plans@1.0.0` | Planning/refinement option | Cataloged candidate; not the current refinement binding |
| `openspec-archive-change@1.0` | OpenSpec archival operation | Cataloged execution candidate; not the current execution binding |

Native Role entries are provenance and ownership records, not fallback Weapons. Strategist does not silently replace a missing or incompatible selected Weapon with catalog order, a native Role, or the latest version.

## Tools

Tools are operational utilities used by the pipeline. They do not own a Role checkpoint.

| Tool | Purpose | Status |
| --- | --- | --- |
| `normalize_openspec` | Validates and promotes a completed OpenSpec refinement into the canonical `.analysis/refined/<mission-id>/` package. | Active |
| `leveling` | Resolves model, capability, effort, and provenance decisions for an operation. | Active internal Tool |
| `resolve_weapon_scratch_root` | Resolves the contained scratch/runtime root for a Weapon, including the private OpenSpec root. | Active |

LEVELING is the operational resolver. It is not a provider selector controlled by INITIATIVE.

## Feats

Feats are reusable capabilities or tactical actions exposed to governed workflows.

| Feat | Purpose | Status |
| --- | --- | --- |
| `initiative` | Advisory diligence, alignment, evidence, and outcome tracking. It cannot bypass the Approval Gate or authorize implementation. | Active advisory Feat |
| `critical_hit` | Evidence-backed closure path for a sufficiently proven package. | Active governed Feat |
| `opportunity_attack` | Separates incidental cleanup or follow-up work from the main mission. | Active governed Feat |
| `side_quest` | Records optional work that is deferred, separated, or selected at the gate. | Active governed Feat |
| `search` | Searches the governed evidence/workspace surface. | Active pipeline Feat |
| `select_runbook` | Selects an applicable operational runbook. | Active pipeline Feat |
| `riposte` | Captures a deferred correction or follow-up after review. | Active governed Feat |
| `keen_senses` | Supports evidence awareness and discovery checks. | Active pipeline Feat |

`INITIATIVE` and `LEVELING` are intentionally separate: INITIATIVE consumes LEVELING output but cannot choose provider, model, or effort.

## Supporting families

### Mechanisms

Mechanisms are controls rather than user-selected providers. The current registry includes:

`mission_status_fsm`, `approval_gate`, `pipeline_bypass`, `role_contract`, `weapon_binding`, `preflight_compatibility`, `fingerprint_integrity`, `trust_grants`, `handoff_contract`, `handoff_challenge`, `confidence_governance`, `mission_quality`, `compliance_summary`, `locks_transactions`, and `control_log`.

### Stages

| Stage | Meaning |
| --- | --- |
| **FULL** | Conservative default route with discovery, refinement, gate, and optional execution. |
| **SHORT** | Narrow route allowed only when scope, targets, and intent are already clear. It still requires the Approval Gate. |
| **ROSTER** | Selection and binding stage for registry-derived Weapons. |
| **Approval Gate** | Human review stage between refinement and execution. |

FULL and SHORT are route stages, not resource families. A route selection is not itself a Weapon or Role.

### Artifacts

Important durable artifacts include:

- `.strategist/active.yaml` — active mode, language, slot, and treasure-chest configuration.
- `.strategist/plugins.lock` — durable binding authority for selected Ranked and Custom providers.
- Source and compiled plugin catalogs, locks, and the compiled registry — acquisition and compatibility evidence.
- Route decisions, pending analysis, refined proposal/design/tasks packages, handoffs, gate records, and completion reports.
- Confidence, telemetry, install-manifest, and generated-runtime records.
- **Weapon Roster** — the registry-derived artifact listing Weapons, compatibility, certification, and identity facts.

## Internal pipeline services

These are supporting services, not configurable Weapons:

`prompt-intake`, `context-enrichment`, `dossier-builder`, `response-critic`, `learning-curator`, and native Role skill packages.

Keeping these separate prevents implementation helpers from being presented as installable provider choices.

## Landing-page content map

The following blocks can be reused by a landing page without copying the full reference catalog:

1. **Simple model:** Roles own responsibility; Weapons provide bounded capability; Tools operate the system; Feats express reusable actions.
2. **Governed journey:** Install → Wizard → Roster/Binding → Mission → Approval Gate → Documentation result.
3. **Trust message:** selected providers are pinned and fail closed; readiness is not the same as live invocation.
4. **Role cards:** Scout, Ranger, Archivist, and Sniper with one-line responsibilities.
5. **Learn-more links:** this catalog, the pipeline-flow guide, the canonical taxonomy ADR, and the CLI reference.

The landing page itself is a separate implementation target and is not changed by this catalog.
