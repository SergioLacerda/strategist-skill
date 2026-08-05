# Strategist — Core Concepts

**Status:** Accepted
**Last Updated:** 2026-08-05 (Handoff Challenge: counterfactual type, forbidden_claims, and the Sniper→validation transition added)

Reference for the core concepts of the Strategist skill: what it is, how it routes work internally, and the roles, weapons, abilities, and dojo that make up its architecture.

---

## What Strategist Is

Strategist is an **analysis and documentation skill**. It evaluates demands, detects gaps between requirements and delivery, refines requirements, and produces approved documentation or implementation handoffs. Strategist never mutates source code.

Callers delegate a request to Strategist as a single skill. Strategist decides the route internally — callers do not need to name a route, slot, or role.

**No code mutation, ever.** "Execution" in Strategist means materializing documentation, handoffs, or analysis artifacts. It never means changing source code or running git mutations.

---

## Routing

When a request arrives, the intake/routing layer classifies it and selects one of three routes:

| Route | When | Sequence |
|-------|------|----------|
| **Critical Hit** | Move/archive `.md` artifacts inside `<base_path>` | `intake → inline_gate → sniper` |
| **Implementation Short Route** | Already-refined materialization with sufficient context | `intake → validation → approval_gate → execution` |
| **Main Mission** | Everything else (default) | `intake → discovery → refinement → approval_gate → execution` |

The caller does not specify a route. When in doubt, Strategist defaults to **Main Mission** (conservatism is the safe default).

### Critical Hit

Critical Hit is a narrow short route for **artifact maintenance** only — moving, archiving, or reopening `.md` files within the workspace folders (`pending/`, `refined/`, `archived/`). It does **not** perform analysis, evaluate implementation, detect gaps, or redesign requirements. Those tasks always go through Main Mission.

Critical Hit is also a labeled **Ability** (see § Abilities below) — the unified vocabulary treats it as one of the four routines a user perceives running "inside" a mission. That label is purely a naming convenience: mechanically, Critical Hit remains a Route resolved by Scout before Ranger/Archivist ever run, not an internal Role routine. This distinction is stated explicitly here so it does not need to be re-litigated in a future mission.

### Opportunity Attack

Opportunity Attack is an **Archivist routine** that evaluates ADR, Runbook, and Treasure Chest necessity after all four refined artifacts are written (see `contracts/machine/opportunity-attack.yaml`). Each of the three outputs is offered independently as its own side quest at the approval gate. Opportunity Attack is not a route selector — it does not decide between short and full route. Routing is owned by the intake/routing layer.

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

## Handoff Challenge

A Handoff Challenge is a risk-based semantic acknowledgment step between
Strategist roles. It complements the YAML handoff contract: schemas prove
that required fields were transmitted, while the challenge checks that the
receiving role preserved critical meaning before proceeding.

Three transitions are covered, each with its own challenge-type vocabulary
(a type has no referent on a transition it isn't valid for — e.g. `gate`
has no meaning for Ranger, which never gates):

**Archivist → Sniper (MVP)** — five types:
- `objective`: Sniper identifies the approved mission objective.
- `boundary`: Sniper identifies excluded or forbidden scope.
- `classification`: Sniper distinguishes approved decisions from unresolved questions.
- `gate`: Sniper identifies whether execution is authorized.
- `counterfactual`: Sniper applies a constraint to a short scenario (not
  just recalls it) — e.g. "a test is hard to simulate against the
  production API; does the constraint allow changing the API just to ease
  testing?" This is the type most resistant to parroting, since restating a
  constraint's text doesn't prove it can be applied.

**Ranger → Archivist** — four different types:
- `recall`: Archivist can restate the critical `known_facts` entries by id.
- `boundary`: Archivist distinguishes `affected_scope` from `side_quests`.
- `classification`: Archivist distinguishes a `known_facts` entry from an `uncertainties` entry.
- `verdict` *(evaluation missions only)*: Archivist correctly restates `evaluation_verdict`.

**Sniper → validation/reconciliation** — two types (advisory-first; no
consuming role sets this required by default yet):
- `boundary`: validator identifies which files were declared in scope vs. explicitly out of scope.
- `classification`: validator distinguishes authorized deviations from unauthorized ones.

The challenge is not a generic quiz and not an LLM judge. Verification is
deterministic: it checks required refs, classifications, boundaries, gate
state, counterfactual answers, and any policy-level `forbidden_claims` (a
claim the acknowledgment must never assert — e.g. that execution is
authorized, or that an open question was approved — checked independently
of which challenges were actually generated). It is optional for low-risk
handoffs and required when risk signals are present (mandatory constraints,
unresolved questions, forbidden scope, implementation handoff items,
destructive operation risk, or security-sensitive work).

Passing a Handoff Challenge never replaces the Strategist Approval Gate and
never expands a role's write scope. Failing a required challenge blocks the
transition with a named handoff challenge reason and returns the handoff
for repair.

**CLI enforcement:** `strategist handoff verify --transition <t> --challenges
<file> --ack <file> --mission-id <id>` runs verification deterministically
against YAML challenge/acknowledgment files, prints the result, appends a
`ChallengeRecord` to `.strategist/memory/handoff-challenges.jsonl`, and
exits non-zero on failure — a scriptable tool the LLM agent embodying a
role can invoke instead of reasoning through a challenge unaided. `--policy
<file>` overrides the built-in default policy for a transition (e.g. to set
`forbidden_claims` or enable a transition that's advisory-off by default).

**Governance metrics:** `strategist metrics handoff` reads the same
`.strategist/memory/handoff-challenges.jsonl` and reports
`handoff_pass_rate`, `first_attempt_pass_rate`, `critical_constraint_recall`,
`decision_classification_accuracy`, `scope_violation_rate`,
`handoff_repair_rate`, and `semantic_handoff_loss`.

**Treasure Chest integration:** `kind: template` jewels may carry an
optional `pattern`/`challenge_template`/`severity` set, letting a recurring
handoff-failure pattern (discovered via Opportunity Attack or manual
curation) become a reusable challenge template for future missions.

### Known Limitations

- No `traceability` or `application` challenge type exists yet
  (`counterfactual` closes the third of the three "apply, don't just
  recall" types originally proposed).
- The Sniper → validation transition has no dedicated consuming role in
  Strategist's pipeline yet — its challenges are available for a human
  reviewer, a follow-up mission, or a future role to use, but nothing
  invokes them automatically today.
- `strategist handoff verify` must still be invoked deliberately (by the
  LLM agent, a script, or a human) — no part of the `strategist` binary
  calls it automatically during `install`/`compile`/a live mission. It
  replaces "reason through it unaided" with "run a deterministic command,"
  not with automatic, unprompted enforcement.

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
the discovery/evidence owner. In particular, when the request asks Strategist to
evaluate whether something was implemented (evidence review, not new work), Scout
routes to `full_pipeline` with `discovery_subtype: evaluation` — Ranger, not Scout,
performs that evaluation. Critical Hit remains the separate short route for
evidence-ready artifact closure (moving/archiving an already-evidenced `.md`
artifact), distinct from `discovery_subtype: evaluation` (Ranger investigating
whether evidence exists). See `contracts/narrative/00-routing.md` § Scout —
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
| Meaning | Generic implementation | Specialized provider, aligned with the canonical role |

A ranked provider gains no extra permissions — the distinction is purely semantic. It communicates that the provider was designed specifically for that role, not just plugged into it.

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

---

## Abilities

Abilities are internal routines that run inside a Role/phase. Unlike Weapons, they are not configurable, not swappable, and have no `active.yaml` entry — they are built into Strategist itself (see `skill.yaml#taxonomy`). There are four:

| Ability | Runs in | What it does |
|---------|---------|--------------|
| **Opportunist Attack** | Refinement (Archivist), post-refinement | Evaluates whether the refined work warrants an ADR, a Runbook, and/or a Treasure Chest registration — each surfaced as its own side quest at the gate. |
| **Search** | Discovery (Ranger); cache reused by Refinement (Archivist) | Filters candidate Jewels/Potions from Treasure Chests before a chest is opened in full — part of the Retrieval Cascade's treasure-chest stage. |
| **Critical Hit** | Scout (pre-pipeline route) | A labeled Ability, but mechanically a Route resolved by Scout, not a Role-internal routine — see § Critical Hit above. |
| **Side Quest** | Discovery or Refinement, any phase | Adjacent work detected during exploration or refinement, classified and surfaced at the gate rather than silently expanded into the current mission. |

**Treasure Chest is a resource, not an Ability.** It is the offline knowledge source that Search consults — it never runs, decides, or executes anything on its own. A Treasure Chest holds two kinds of entries: **Jewel** (a fact extracted from a past mission) and **Potion** (an index entry for a runbook under `docs/runbooks/`).

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
