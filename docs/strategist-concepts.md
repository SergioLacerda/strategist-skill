# Strategist — Core Concepts

**Status:** Accepted
**Last Updated:** 2026-06-30

Reference for the core concepts of the Strategist skill: what it is, how it routes work internally, and the roles, weapons, initiative, and dojo that make up its architecture.

---

## What Strategist Is

Strategist is an **analysis and documentation skill**. It evaluates demands, detects gaps between requirements and delivery, refines requirements, and produces approved documentation or implementation handoffs. Strategist never mutates source code.

Callers delegate a request to Strategist as a single skill. Strategist decides the route internally — callers do not need to name a route, slot, or role.

**No code mutation, ever.** "Execution" in Strategist means materializing documentation, handoffs, or analysis artifacts. It never means changing source code or running git mutations.

---

## Routing

When a request arrives, the intake/routing layer classifies it and selects one of four routes:

| Route | When | Sequence |
|-------|------|----------|
| **Quick Draw** | Explicit quick-capture / note-append | `intake → ranger_qd → archivist_qd → gate → sniper_qd` |
| **Critical Hit** | Move/archive `.md` artifacts inside `<base_path>` | `intake → inline_gate → sniper` |
| **Implementation Short Route** | Already-refined materialization with sufficient context | `intake → validation → approval_gate → execution` |
| **Main Mission** | Everything else (default) | `intake → discovery → refinement → approval_gate → execution` |

The caller does not specify a route. When in doubt, Strategist defaults to **Main Mission** (conservatism is the safe default).

### Critical Hit

Critical Hit is a narrow short route for **artifact maintenance** only — moving, archiving, or reopening `.md` files within the workspace folders (`pending/`, `refined/`, `archived/`). It does **not** perform analysis, evaluate implementation, detect gaps, or redesign requirements. Those tasks always go through Main Mission.

### Opportunity Attack

Opportunity Attack is an **Archivist routine** that evaluates ADR necessity after all four refined artifacts are written. It is not a route selector — it does not decide between short and full route. Routing is owned by the intake/routing layer.

---

## Pipeline Overview

Before the three-slot pipeline runs, **Scout** (the Intake Router) classifies the
request and selects a route: `critical_hit`, `implementation_short_route`, or
`full_pipeline`. Only `full_pipeline` reaches the slot pipeline below:

```
Scout (route decision) → Ranger (discovery) → Archivist (refinement) → [gate] → Sniper (execution)
```

The Strategist never executes work directly — it delegates. Each slot receives a **provider** (weapon) configured in `active.yaml`. The combination of provider + slot + contract defines a **role**.

---

## Role

A role is the combination of a slot with its behavior contract. There are three canonical, pluggable roles:

| Role | Slot | Contract | Authorized writes |
|------|------|----------|------------------|
| **Ranger** | `discovery` | `write_analysis` | `.md` in `<base_path>/pending/` |
| **Archivist** | `refinement` | `write_analysis` | `.md` in `<base_path>/refined/` |
| **Sniper** | `execution` | `controlled` | Approved documentation/handoff, only after approval gate |

**Scout** is a fourth role, but it is internal and pre-pipeline, not a slot — it has
no `active.yaml` entry and no configurable provider:

| Role | Slot | Contract | Authorized writes |
|------|------|----------|------------------|
| **Scout** | pre-pipeline (internal, never a slot) | `read_only` | none — emits a `route_decision`, logged/telemetered only |

## Scout — Intake Router

Scout classifies each request and decides the route before any slot runs. It is
internal Strategist behavior, analogous in scope-boundedness to Sniper but
positioned before the pipeline instead of at the end of it — there is no
`roles/scout.yaml` and no way to configure a different Scout provider. `Scout` is
the internal persona name; `Intake Router` is the same entity's public/pragmatic
contract label used in narrative-mode responses.

Scout may NOT perform deep discovery, invoke Sniper directly, bypass the
Strategist Approval Gate, or replace Ranger when evidence review is required. When
a request needs evidence gathering, Scout routes to `full_pipeline` with a
`discovery_subtype` (see `contracts/narrative/03-discovery.md`) and Ranger remains
the discovery/evidence owner. See `contracts/narrative/00-routing.md` § Scout —
Intake Router and `internal_skills/scout/SKILL.md` for the full contract.

Each role has a contract declared in `.strategist/roles/<role>.yaml` with `must` and `must_not` clauses. Example (Ranger):

```yaml
must:
  - separate facts, hypotheses, and ambiguities clearly
  - include all handoff contract fields in the analysis artifact
  - surface scope_observations (side quests and unexpected items) in the response to the user

must_not:
  - propose a final plan as if it were approved
  - execute any changes
  - pass raw context to Archivist (compress to evidence cards)
  - run opportunity_attack (Archivist responsibility after the four refined artifacts)
```

Canonical responsibilities by role:

- **Ranger** captures discovery and may report side quests. Does not run Opportunity Attack.
- **Archivist** classifies side quests from discovery, writes four refined artifacts (`analysis.md`, `proposal.md`, `design.md`, `tasks.md`), and runs Opportunity Attack (ADR evaluation) after all four are written.
- **Sniper** materializes approved tasks and reports newly discovered side quests. Does not run analyses or ADR evaluations.

Sniper requires explicit user approval before any execution — no exceptions. Under the current contract, execution means materializing documentation, diagrams, analyses, or approved handoffs; it does not mean changing source code.

---

## Parent Agent Role Lock

When Strategist is invoked, the parent agent (Codex, Claude, or any host) becomes a
constrained orchestrator shell — it must not solve the user's task directly. See the
"Role Lock" section in `SKILL.md` and "Parent Agent Boundary" in `agent-protocol.md`
for the normative rules. Examples of correct and incorrect behavior:

- **Read-only analysis request** — user asks Strategist to evaluate a proposal. The
  parent agent bootstraps, invokes the discovery/refinement providers, presents the
  gate, and relays their output. It never inspects or judges the code itself.
- **Code/test mutation request** — user asks Strategist to "clean up duplicated
  tests." The parent agent produces analysis/handoff artifacts only; it does not edit
  the test files, because the default Sniper contract forbids code/test mutation.
  Implementation happens outside Strategist or through a separately configured
  execution provider whose contract permits mutation.
- **Unavailable provider** — the configured discovery provider cannot be invoked. The
  parent agent stops and emits `error=role_invocation_failed slot=discovery
  provider=<configured_provider>`. It does not perform discovery itself to "help."
- **Anti-example (drift)** — the parent agent reads `SKILL.md` and `agent-protocol.md`,
  then performs discovery, refinement, or execution itself instead of invoking the
  configured provider. This is `direct_execution` drift even if the resulting answer
  is correct — correctness does not repair the drift.

---

## Ranked Role

A ranked role is a provider that declares `provider_class: rankeado` in its `skill.yaml`.

```yaml
# .strategist/skills/brainstorming/skill.yaml
specialization_taxonomy:
  canonical_role: ranger
  provider_class: rankeado     # ← ranked role
```

The difference from a `(base)` provider:

| | Base | Ranked |
|---|------|--------|
| `provider_class` | absent or `base` | `rankeado` |
| `specialization_taxonomy` | not declared | `canonical_role` + `provider_class` filled in |
| Display in `initiative` | `(base)` | `rankeado` |
| Meaning | Generic implementation | Specialized provider, aligned with the canonical role |

A ranked provider gains no extra permissions — the distinction is semantic and visible in the `initiative` command. It communicates that the provider was designed specifically for that role, not just plugged into it.

Ranked providers installed in this workspace:

| Provider | Slot | Canonical role |
|----------|------|---------------|
| `brainstorming` | discovery | Ranger |
| `openspec-explore` | refinement | Archivist |

---

## Weapons

Weapons are the concrete providers configured in each slot. The metaphor: the role (Ranger, Archivist, Sniper) is the warrior; the weapon is the skill they wield to do their work.

Configuration in `.strategist/active.yaml`:

```yaml
slots:
  discovery: brainstorming       # Ranger's weapon
  refinement: openspec-explore   # Archivist's weapon
  execution: sniper              # Sniper's weapon
```

Each weapon is a skill with its own `skill.yaml` resolved in preflight by the Strategist. The weapon's risk contract (`risk_score`) must match the slot contract:

| Slot | Expected risk_score |
|------|---------------------|
| discovery | `write_analysis` |
| refinement | `write_analysis` |
| execution | `controlled` |

To swap a weapon, change the slot value in `active.yaml` and ensure the new provider's `skill.yaml` exists at `.strategist/skills/<provider>/skill.yaml`.

## Initiative

`strategist initiative` is the CLI command that answers: *"who is armed in each slot right now?"*

```bash
$ strategist initiative

SLOTS                                                  
discovery      brainstorming      Ranger rankeado      ✓ manifest OK
refinement     openspec-explore   Archivist rankeado   ✓ manifest OK
execution      sniper             Sniper (base)        ✓ manifest OK
                                                       
WORKSPACE                                              
mode           epic                                    
base_path      .analysis                               
pending        0 cards                                 
done           49 missions                              
last mission   20260620-feature-xyz                    
```

The **SLOTS** section displays, for each slot:

| Column | Meaning |
|--------|---------|
| slot | `discovery`, `refinement`, `execution` |
| provider | ID of the skill configured in `active.yaml` |
| canonical_role | Canonical role declared in the manifest (`ranger`, `archivist`, `sniper`) |
| class | `rankeado` if `provider_class: rankeado`, otherwise `(base)` |
| manifest status | `✓ manifest OK` if `.strategist/skills/<provider>/skill.yaml` exists and is valid; `⚠ manifest missing` otherwise |

The **WORKSPACE** section displays: `mode` and `base_path` from `active.yaml`, counts of pending cards and completed missions, and the ID of the last mission from `memory/outcomes.jsonl`.

The command makes no LLM calls — it only reads `active.yaml` and local `skill.yaml` files. Useful for quickly verifying the workspace is healthy before starting a mission.

```bash
# With custom root
strategist initiative --root /path/to/.strategist
```

---

## Dojo

The Dojo is the Strategist skill's training system — a two-layer health check that validates whether the skill is installed, whether roles are filled, and whether the pipeline operates correctly.

### Layer 1 — Offline (zero LLM)

```bash
strategist dojo check <scenario>            # validates artifacts, emit log, and manifests
strategist dojo check <scenario> --files-only  # validates files only (no emit log)
strategist dojo list                        # lists available scenarios
```

Reads the scenario's `criteria.yaml` and verifies:
- **files_created**: files exist, contain required sections and canary strings
- **emit_log**: expected OTEL events present/absent in `.last-run/<scenario>/emit.log`
- **manifest_checks**: provider manifests exist with required fields

### Layer 2 — LLM (real pipeline with synthetic input)

```
/strategist dojo <scenario>
```

Runs the full pipeline with input from `<base_path>/dojo/<scenario>/input.yaml`, writes artifacts to `<base_path>/dojo/run/` (isolated from production), and automatically calls Layer 1 at the end.

### Available scenarios

| Scenario | What it validates |
|----------|------------------|
| `quick-draw` | Raw idea converted to a todo item with canary `KATA_RAPIDO`; pipeline stops at gate |
| `treasure-chest` | Planted chest found and canary `TORNEIO_DO_DOJO` incorporated in the analysis |
| `ranger-weapons` | Discovery provider manifest exists with `canonical_role` and `provider_class` fields |

### Scenario structure

```
<base_path>/dojo/<scenario>/
├── input.yaml      # synthetic input for the LLM layer
├── criteria.yaml   # validation contract (files, emit, manifests)
├── golden/         # reference artifacts (optional)
└── chests/         # planted treasure chests (treasure-chest scenario)
```

### Harmlessness rule

Every dojo `input.yaml` must be harmless: idea prefixed with `[dojo-fixture]`, targeting new paths (e.g. `docs/dojo/`). If Sniper fires accidentally, no production code is touched.

### Adding a new scenario

1. Create `<base_path>/dojo/<name>/`
2. Write `input.yaml` with a harmless idea and a unique canary string
3. Write `criteria.yaml` referencing the canary in `must_contain`
4. Validate syntax: `strategist dojo check <name> --files-only`
