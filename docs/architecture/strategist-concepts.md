# Strategist — Core Concepts

**Status:** Accepted
**Last Updated:** 2026-10-02 (seven-family taxonomy and evidence states)

See [`strategist-philosophy.md`](strategist-philosophy.md) for the rationale
behind the fixed pipeline, replaceable weapons, evidence authorities, and
approval-gated materialization described below.

Reference for the core concepts of the Strategist skill: what it is, how it routes work internally, and the canonical taxonomy that makes up its architecture.

---

## What Strategist Is

Strategist is an **analysis and documentation skill**. It evaluates demands, detects gaps between requirements and delivery, refines requirements, and produces approved documentation or implementation handoffs. Strategist never mutates source code.

Callers delegate a request to Strategist as a single skill. Strategist decides the route internally — callers do not need to name a route, slot, or role.

**No code mutation, ever.** "Execution" in Strategist means materializing documentation, handoffs, or analysis artifacts. It never means changing source code or running git mutations.

## Canonical Taxonomy

The public vocabulary has seven families, fixed by
[ADR-0064](../adr/0064-canonical-seven-family-taxonomy.md). ADR-0053 remains
historical evidence of the prior six-family model; current runtime contracts may
still expose that vocabulary until a separately approved implementation wave
migrates them.

| Family | Criterion | Current examples |
|--------|-----------|------------------|
| **Role** | Agents acting as personas that own a responsibility | Scout, Ranger, Archivist, Sniper |
| **Weapon** | External skill packages employed by a pluggable Role | `brainstorming`, `openspec-propose` |
| **Feat** | Contextual behavior performed through judgment | INITIATIVE, Search, Riposte, Critical Hit, Opportunity Attack |
| **Tool** | Operation that evaluates inputs and produces a result | LEVELING, Weapon Discovery, Compatibility Resolver, Binding Resolver |
| **Mechanism** | Deterministic rule whose outcome is fixed by explicit inputs | Approval rules, handoff validation, binding rules, fingerprint integrity, budget enforcement |
| **Stage** | Governed operational flow that may contain several phases | FULL, SHORT, ROSTER |
| **Artifact** | Materialized state, evidence, decision, or knowledge | analysis, dossier, refined package, ADR, Weapon Roster, Weapon Binding |

Routes are not a family. Scout selects a route identifier, which chooses an
eligible Stage or bounded capability under current runtime vocabulary.

An item may expose multiple facets, but each facet must be classified explicitly.
Opportunity Attack is a Feat when Archivist recognizes a documentation opportunity;
its activation criteria and gate rules are Mechanisms. A serialized handoff is an
Artifact; validation of that handoff is a Mechanism.

`LEVELING` is an immutable operational resolver and a Tool, not a Role, Weapon,
Mechanism, provider, or execution authority. It consumes structured mission and
confidence inputs and produces an effort resolution. Its versioned policy and budget
enforcement are Mechanisms.

`INITIATIVE` is a Feat. It consumes the LEVELING resolution and applies contextual
judgment about diligence, alignment, evidence, and escalation. It may request a new
resolution when permitted, but cannot mutate an existing resolution, select a provider,
bypass the Approval Gate, or authorize execution.

`PRECISE-SHOT` remains the stable identifier for confidence self-assessment. Its
evaluation method may be supplied by a Weapon; confidence validation and request bounds
remain Mechanisms.

`Critical Hit` is a Feat with deterministic eligibility Mechanisms. Scout applies the
eligibility rule, and the route value `critical_hit` selects the bounded flow. Only the
declared `plain` and `closure` modes can activate; an unknown or unsatisfied mode fails
closed to `full_pipeline`.

The agent-facing registry currently named `contracts/machine/mechanisms.yaml` catalogs
Tools, Mechanisms, and Feats for runtime consumption. That filename is current-runtime
compatibility evidence, not a competing conceptual taxonomy. Each Role receives a scoped
brief at phase start (`strategist mechanisms brief --role <role>`).

Role ownership and extensibility remain orthogonal: `origin: native|external`
identifies the contract owner, while `extensibility: fixed|pluggable` identifies
whether a compatible Weapon/provider may fill the role. “Internal role” and
“external role” are historical compatibility wording only, never aliases for
either property. The existence and availability of Pathfinder, Cartographer,
Jeweler, Jewelcrafter, and Sharpshooter must be stated through the evidence-state
matrix below rather than inferred from a name or an accepted ADR.

### Evidence-state matrix

| State | Required meaning |
| --- | --- |
| **Conceptual** | The entity is defined in the target model. |
| **Accepted** | An authoritative ADR or gate accepts the entity or boundary. |
| **Implemented** | Current source or runtime represents the entity. |
| **Verified** | Current evidence validates the stated implementation scope. |
| **Invocable** | A working and authorized runtime path can invoke it. |

These states are independent. For example, ADR acceptance can establish a Role as
accepted without proving that it is registered, verified, or invocable. Current
documentation must name the strongest evidenced state and must not promote one state
to another by implication.

---

## Routing

When a request arrives, the intake/routing layer classifies it and selects one of three routes:

| Route | When | Sequence |
|-------|------|----------|
| **Critical Hit** | Move/archive `.md` artifacts inside `<base_path>` | `intake → inline_gate → sniper` |
| **Implementation Short Route** | Already-refined materialization with sufficient context | `intake → validation → approval_gate → execution` |
| **Full Pipeline** | Everything else (default) | `intake → discovery → refinement → approval_gate → execution` |

The caller does not specify a route. When in doubt, Strategist defaults to **Full Pipeline** (conservatism is the safe default).

### Critical Hit

Critical Hit is a narrow short route for **artifact maintenance** only — moving, archiving, or reopening `.md` files within the workspace folders (`pending/`, `refined/`, `archived/`). It does **not** perform analysis, evaluate implementation, detect gaps, or redesign requirements. Those tasks always go through Full Pipeline.

Critical Hit is a **Feat** with deterministic eligibility Mechanisms (see § Tools, Mechanisms, and Feats below). Scout applies its activation rule and resolves the route before Ranger/Archivist ever run, so it is not a Role routine.

### Opportunity Attack

Opportunity Attack is an **Archivist Feat** that evaluates ADR, Runbook, and Treasure Chest necessity after all four refined artifacts are written (see `contracts/machine/opportunity-attack.yaml`). Each of the three outputs is offered independently as its own side quest at the approval gate. Opportunity Attack is not a route selector — it does not decide between SHORT and FULL. Routing is owned by the intake/routing layer.

### Runbook Domain Model (typed sidecars)

Beyond the write-side candidate flow above, every accepted runbook under `docs/runbooks/*.md` also carries a co-located, typed `docs/runbooks/<slug>.runbook.yaml` sidecar (schema documented in `docs/runbooks/README.md`) — `applies_when`, `objective`, `preconditions`, `analysis`/`decision_gates` (analytical runbooks) or `verification` (operational runbooks), and leveled `checks` (`mandatory`/`recommended`/`conditional`/`informational`). The `internal/runbook` Go package (`Runbook`, `Select()`, `EvaluateStep()`, `ValidateCompletion()`) parses and operates on these sidecars, with `Select()` enforcing a reasoned, bounded choice (`max_primary`/`max_supporting`/`require_reason`) rather than silent auto-application.

`Select()` has a concrete caller: `strategist runbook select` scores the sidecars against mission signals. `roles/ranger.yaml#canonical.select_runbook` makes it a required step of Ranger's Retrieval Cascade stage 6 (`contracts/narrative/03-discovery.md`): Ranger runs the command with one `--signal` per mission signal and records the result in `selected_runbooks_hint` (an empty list means it ran and nothing matched; `null` means it was not run). The invocation is agent-executed — nothing in Go runs it on Ranger's behalf — and unstructured runbook content still also reaches Ranger through the generic `runbooks` Treasure Chest (Potion/jewel relevance matching, § Abilities table below).

---

## Pipeline Overview

Before the three-slot pipeline runs, **Scout** (the Intake Router) classifies the
request and selects a route: `critical_hit`, `implementation_short_route`, or
`full_pipeline`. Only `full_pipeline` reaches the slot pipeline below:

```
Scout (route decision) → Ranger (discovery) → Archivist (refinement) → [gate] → Sniper (execution)
```

The Strategist never executes work directly — it delegates. Each slot receives a **Weapon** configured in `active.yaml`. A **role** is the combination of a slot with its behavior contract; the Weapon is the skill the role wields.

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
- The standalone `strategist handoff verify` command remains deliberately
  invocable by an LLM agent, script, or human. Live mission orchestration now
  invokes the same deterministic verifier at the Archivist-to-Sniper boundary;
  install and compile remain configuration-only operations.

---

## Role

A role is the combination of a slot with its behavior contract. There are four
canonical identity-bearing roles. Ownership and extensibility are independent:
Scout is `native/fixed`, and Ranger, Archivist and Sniper are `native/pluggable`.
The embedded `sniper` ranked binding is Sniper's default and behaves as fixed; a
custom Weapon is accepted on the execution slot only when the user binds one
explicitly (`strategist provider add --slot execution`), and only if it is
documentation-only and declares `risk_score: controlled`.

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

### Role registry and role definition shape

> Status: Tiers A and B are implemented. Tier C (one full template for every
> role, dynamic slots) is planned and needs an approved RFC first; rollout is keyed by
> tier (A first, B after A, C only after an approved RFC). Tracked by the refined
> package `20260921-role-template-generalization`.

`domain.RoleRegistry` is the single authority for role facts and covers the
identity-bearing roles only — Scout, Ranger, Archivist and Sniper. Inline
sub-routines (`prompt-intake`, `context-enrichment`, `dossier-builder`,
`learning-curator`, `response-critic`) are not roles and stay out. The leveling
label, the wizard, handoff-schema lookup, confidence-producer validation and
provider role affinity all read the registry; a test fails if another list of role
names appears in Go source.

Each role has a definition (`roles/<id>.yaml`) of the same shape, including Scout
(`roles/scout.yaml`):

| Field | Meaning |
|-------|---------|
| `role`, `slot` | Role id and its slot; Scout has no slot |
| `phase` | Position in the mission checkpoint; `0` is pre-pipeline. The approval gate is the step immediately before the execution role, and the `Fase: NN/MM` total is the last phase, so adding a role changes it without a code edit |
| `origin` | Contract owner: `native` or `external`; independent from extensibility |
| `extensibility` | Whether a provider may fill the role: `fixed` or `pluggable` |
| `pluggable` | Removed. A role file that still declares this boolean is rejected; declare `extensibility` instead |
| `handoff_schema` | Schema the role hands downstream; empty for the terminal role |
| `leveling` | Optional name of the LEVELING policy role to use; defaults to the role id |
| `on_start` | Commands the role runs when its phase starts; the default resolves and records the level (`strategist leveling label --role {role} --mission {mission_id}`), so the label is part of role invocation |
| `must`, `must_not`, `custom_brief` | Behavior contract, unchanged |

The registry loaded at runtime is the workspace's `roles/*.yaml` laid over the
built-in registry: a file replaces the role with the same id and a new file adds
a role, so a customized role file is honored. The built-in registry is checked
against the embedded role files by a parity test. Editing a role file changes the
Ranked certification `host_api_digest` of that role; after changing an embedded
role file run `strategist plugins prepare-embedded` and commit the result.
Whether the LEVELING policy is read at all stays governed by the `leveling.mode`
switch (`manual` never reads it, `automatic` reads it on demand).

Tier B replaced the hand-written per-role message strings with generic templates.
A persona (or the pt-BR bundle in `internal/i18n`) declares `role_start`,
`role_done` and `role_task_done` templates plus a `role_phrases` table (per-role
emoji, wording and artifact label, with a `_default` entry). The compile step
expands them once per registered role with a checkpoint phase into the
`<role>_start`, `<role>_done` and `<role>_task_done` keys agents already read, so
those keys are aliases (`compile.RoleEventAliases` lists them) and a hand-written
key still wins. The progress bar and percentage come from the role's phase and the
registry total, so adding a role needs no new strings and rescales the others. A
golden test pins the generated messages to the pre-generalization output.
Narration lines name the role with an inline level tag (`Ranger(Sonnet-High)`) and
Scout has its own narration line; content templates keep the stacked header.
Tier C (one full template for every role, dynamic slots) is a breaking change and
requires an RFC first.

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
  parent agent bootstraps, obtains the Ranked invocation envelope, executes its
  exact payload once under the emitted execution and output contracts, submits the
  completion, presents the gate, and relays the pipeline output. This scoped
  current-host adapter is the Weapon invocation; it does not add the parent's own
  workflow, provider, or conclusions.
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

The `--host codex|claude` executable bridge is a standalone-shell convenience for
an operator whose child process has working network and auth. A managed Codex or
Claude session does not recursively spawn the same host because the child may
inherit a sandbox without network access; it uses the current-host adapter above.

---

## Ranked Class

> Corrected 2026-09-15 (mission `20260915-ranked-binding-vs-provider-class-naming-residual`).
> This section previously described a `provider_class: rankeado` field as
> current/live. That field was removed by ADR-0034 — no code branches on it,
> and its absence is enforced by `internal/install/plugin_catalog_test.go` and
> `internal/embed/embed_test.go`. "Ranked" is redefined below; the name is kept
> deliberately, not coincidentally reused.

A **ranked class** is a Role that, at compile/build time, is already bound to a
specific weapon (an embedded Weapon, `origin: embedded`) — the binding is known before the mission
runs, not resolved by the Wizard or discovered by the agent at runtime. This
compile-time binding is built on the same certification pipeline the pending
"Ranked Binding" draft describes (manifest/dependency/affinity/contract-test/
handoff-schema validation, digest calculation): the pipeline is the core
mechanism: a ranked class is one implementation built on that core, and other
future functionality is expected to build on the same core too.

In the Wizard, this surfaces as two paths for a pluggable Role:

- **Ready Role** — pick a ranked class: the Role arrives with its embedded
  weapon already bound.
- **Custom** — bind the Role to an external weapon yourself (today's existing
  flow, unchanged).

> **Pipeline scope** (updated 2026-09-29, mission
> `20260916-ranked-vs-custom-binding-pipelines`): Ready Role/Ranked and Custom
> are two separate pipelines, not two layers of the same mechanism. Custom's
> runtime machinery — the readiness vector (`domain.PluginReadinessVector`),
> trust-policy verification, permission-grant evaluation, and digest-pinned
> `SlotBinding` in `plugins.lock` — exists because Custom accepts a
> wizard-time-selected, potentially external weapon that needs runtime
> verification. A ranked class's binding is already certified at build time,
> so it does not go through any of that: no wizard-time validation, no
> runtime trust check, no permission-grant negotiation. Ranked never calls
> into Custom's checks. The compiled registry is authoritative for the
> immutable Role/Weapon association; `plugins.lock` is the workspace-local
> materialized choice and must preserve the compiled Role, Weapon, runtime,
> execution mode, connector, entrypoint, and digests. Runtime resolution
> compares both before dispatch.

A large part of the existing catalog/selection structure is reused: whatever is
deterministic resolves at the CLI/build step, while the compiled registry and
embedded payloads remain the runtime authority. `active.yaml` records intent;
it cannot create a new Ranked association. Custom bindings still use their
explicit external connector lifecycle.

The implementation details are governed by the approved
`docs/plans/2026-09-29-binding-weapons-runtime.md` plan and its design source
`.analysis/pending/2026-09-29-binding-weapons-design.md`: build generation is
the registration boundary, the wizard materializes the selected immutable
binding, and runtime dispatch uses that binding without external Ranked
resolution.

---

## Weapons

Weapons are the skill packages a pluggable role wields, configured in each slot. The metaphor: the role (Ranger, Archivist, Sniper) is the warrior; the weapon is the skill they wield to do their work.

Configuration in `.strategist/active.yaml`:

```yaml
slots:
  discovery: brainstorming       # Ranger's weapon
  refinement: openspec-propose   # Archivist's weapon
  execution: sniper              # Sniper's weapon
```

Each weapon is a skill package registered in the plugin catalog at build time by
the Strategist. A Ranked Weapon is not resolved from a host skill directory at
runtime. The weapon's risk contract (`risk_score`) must match the slot contract:

| Slot | Expected risk_score |
|------|---------------------|
| discovery | `write_analysis` |
| refinement | `write_analysis` |
| execution | `controlled` |

### Weapon origin and runtime

Every Role-bindable package is a Weapon, atomic or composite; "skill" is only the upstream `SKILL.md` file format and never a Strategist package category. A Weapon declares two independent facts:

| Field | Values | Meaning |
|-------|--------|---------|
| `origin` | `embedded`, `custom` | Where the Weapon comes from: the canonical embedded source, or an operator-provided package. |
| `runtime.kind` | `embedded`, `host`, `executable`, `openspec_root` | How the Weapon is invoked. A custom Weapon may run through any of them. |

The cutover is strict. The former runtime kinds `embedded_skill` and `host_skill` are rejected in manifests, catalogs, locks, and readiness with a diagnostic that directs the operator to regenerate or reinstall the workspace; they are never aliased, dual-read, dual-written, or converted, and no native fallback runs. The plugin catalog schema (`strategist-plugin-catalog/v2`), package contract (`skill-package/v2`), and embedded lock schema (`strategist-embedded-skill-lock/v2`) carry the new vocabulary. `internal/embed/defaults/` and `external-skills-source/` are the authoring sources; `.strategist/` is generated from them and is never edited by hand.

To swap a weapon, validate and onboard its local package with `strategist provider validate <source>` and `strategist provider add <source> --slot <slot>`. The catalog or package/adapter contracts plus `plugins.lock` own identity, compatibility, and binding; provider compatibility views are not generated or consulted at runtime. Ranger invokes the selected discovery Weapon, normalizes its untrusted result, and fails closed with `role_invocation_failed` when invocation evidence is unavailable; it never silently substitutes native behavior.

A delegated sub-role (a Role run as a sub-agent rather than in the primary conversation) reaches a Ranked Weapon through Strategist's in-process embedded connector. Codex, Claude, and other hosts may supply the explicitly registered prompt bridge, but they do not resolve a second copy from a user skill directory. The bridge receives the payload selected by the compiled binding; it cannot read `external-skills-source`, `skill-for-hire`, or substitute a native Role. An explicitly typed Custom Weapon may use its declared host connector; its invocation evidence is separately authenticated. `weapon_invocation` in the discovery handoff (`schemas/handoff-ranger-to-archivist.schema.yaml`) records what happened, required for a delegated run: `invoked`, `resolved_from`, `steps_dropped`, and `resolved_digest` (`sha256:<64 hex>` of the bytes resolved by the selected runtime). Ranked embedded invocation records internal evidence without fabricating a host receipt; a missing dispatch or prompt bridge fails closed with `role_invocation_failed`.

---

## Tools, Mechanisms, and Feats

These families may operate inside a Role or Stage, but they express different
responsibilities. The current runtime registry filename
`contracts/machine/mechanisms.yaml` is retained for compatibility and may contain
entries from all three families until the implementation taxonomy is migrated.

| Tool | Runs in | What it does |
|------|---------|--------------|
| **LEVELING** | Before Role invocation | Evaluates mission inputs and resolves capability and effort. The resulting model and effort are shown on Role log lines and recorded in telemetry (see `docs/configuration.md` § Role level label). |
| **Weapon Discovery** | Configuration and readiness | Finds declared Weapon packages without deciding whether they may be bound. |
| **Compatibility Resolver** | Configuration and readiness | Evaluates compatibility inputs and returns a result. |
| **Binding Resolver** | Configuration and readiness | Resolves a permitted Role-to-Weapon binding from explicit contracts. |

| Feat | Runs in | What it does |
|------|---------|--------------|
| **INITIATIVE** | Role entry and handoff boundaries | Applies contextual judgment to immutable LEVELING output and mission evidence. It may request a new resolution when permitted, but cannot mutate an existing resolution, select a provider, bypass the Approval Gate, or authorize execution. |
| **Search** | Discovery (Ranger); cache reused by Refinement (Archivist) | Filters candidate Jewels and Potions before opening a Treasure Chest in full. |
| **Opportunity Attack** | Refinement (Archivist), post-refinement | Detects whether refined work warrants an ADR, Runbook, or Treasure Chest registration, each surfaced independently at the gate. |
| **Critical Hit** | Scout, before the main pipeline | Recognizes a bounded artifact-maintenance request; deterministic eligibility Mechanisms decide whether the Feat may activate. |
| **Side Quest** | Discovery or Refinement | Detects adjacent work and surfaces it at the gate instead of silently expanding mission scope. |

| Mechanism | Deterministic responsibility |
|-----------|------------------------------|
| **Approval rules** | Decide whether accepted evidence authorizes documentation materialization. |
| **Handoff validation** | Validate the schema, integrity, and semantic acknowledgment of Role transitions. |
| **Binding rules** | Enforce declared Role-to-Weapon compatibility and fail-closed resolution. |
| **Fingerprint integrity** | Detect drift between governed sources and generated runtime artifacts. |
| **Budget enforcement** | Apply explicit effort and retry limits to mission execution. |

**Treasure Chest is a resource, not a Feat.** It is the offline knowledge source
that Search consults; it never runs, decides, or executes on its own. A Treasure
Chest holds two kinds of entries: **Jewel** (a fact extracted from a past mission)
and **Potion** (an index entry for a runbook under `docs/runbooks/`).

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
- **manifest_checks**: catalog provider entries exist with required fields

### Dojo state ownership and lifecycle

The configured `base_path` is the compatibility owner of the dojo domain. The
public paths are stable and relative to that root:

```
<base_path>/dojo/<scenario>/criteria.yaml       # durable scenario input
<base_path>/dojo/.last-run/<scenario>/emit.log  # replaceable diagnostic evidence
<base_path>/dojo/.last-run/<scenario>/result.json
<base_path>/dojo/.last-run/<scenario>/lesson.md # durable failure learning
<base_path>/dojo/.history.jsonl                 # durable aggregate history
```

Criteria, lessons, and aggregate history belong in workspace backups. The
latest result is retained derived evidence; the emit log is replaceable and may
be recreated by a later run. Both remain in the workspace dojo domain for
compatibility. Reinstalling or regenerating `.strategist` must not remove
durable dojo state.

The scenario identity is `dojo/<scenario>` under the configured base path. The
CLI rejects path-like scenario names, so a check cannot write outside that
domain. Persistence uses an atomic latest-result replacement and serializes
state writes; a malformed or orphaned result/history is reported for recovery.
A persistence warning is separate from the checker verdict: a failed check
remains failed, and a passed check remains passed even when its learning write
needs retry.

No automatic migration, first-run move, cleanup, symlink, or dual-write path is
provided. A future relocation requires an explicit accepted implementation that
first discovers existing state, previews the operation, preserves the source on
partial failure, and defines rollback before changing writes.

### Layer 2 — LLM (real pipeline with synthetic input)

```
/strategist dojo <scenario>
```

Runs the full pipeline with input from `<base_path>/dojo/<scenario>/input.yaml`, writes artifacts to `<base_path>/dojo/run/` (isolated from production), and automatically calls Layer 1 at the end.

### Available scenarios

| Scenario | What it validates |
|----------|------------------|
| `treasure-chest` | Planted chest found and canary `TORNEIO_DO_DOJO` incorporated in the analysis |
| `ranger-weapons` | Discovery catalog entry exists with a `canonical_role` field |

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
