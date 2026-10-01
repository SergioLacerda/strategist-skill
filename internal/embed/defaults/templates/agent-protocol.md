---
generated_by: strategist compile
version: {{.Version}}
generated_at: {{.GeneratedAt}}
path_model: runtime-only
---

# Strategist — Agent Protocol

## 1. STARTUP — execute before anything else

Execute in exactly this order. Stop at the first failure.

1. Does `.strategist/` exist in the workspace? → No: emit `error=not_installed`, instruct `strategist install`, **stop**
2. Run `strategist check --json`, capture the `PreflightResult`
3. `status == "blocked"` → emit the CLI's `warnings`, **stop** (do not retry silently)
4. `status == "ready"` → read this file (`agent-protocol.md`) to the end, then proceed to `next`

**Do not process any user request before status is `"ready"` and this file has been read to the end.**

`strategist check --json`'s `PreflightResult` (docs/adr/0044) is the sole
source of truth for installation, configuration, and slot/binding
readiness — do not re-derive any of it narratively (e.g. do not separately
ask "is `active.yaml` readable" or "are identity files present": both are
already reflected in `warnings` if they matter). Route selection and role
invocation remain internal Strategist responsibilities beyond this point.
If the Weapon bound to a Role slot cannot be invoked, emit
`error=role_invocation_failed` with the slot and Weapon id. The wire field name
`provider` remains only for backward compatibility.

---

## 1b. PARENT AGENT BOUNDARY

The parent agent is the transport for Strategist, not an implementation substitute
for Strategist slots.

Any action that produces phase work without invoking the configured provider is
`direct_execution` drift, even if the output is correct.

If a provider cannot be invoked, emit the configured blocked state and stop.
Correctness of the parent agent's independent answer does not repair the drift.

### Shell operating rules

These describe how the parent shell runs the CLI so that a step is not silently lost or redone.
They are guidance (`enforced_by: agent_only`), not a gate.

- Do not discard stderr of `strategist` commands, and read the returned status or JSON. A
  `mission submit` that printed an error was not applied; check with `strategist mission status`
  before repeating it. Quote arguments so the shell does not word-split them.
- Start each mission with `strategist mission start --mission-id <id>`, then
  submit `bootstrap_done` before recording a route. After `mission route`,
  submit `intake_done`; only then may a discovery Weapon be invoked. A missing
  mission reported by `mission status` is not an initialized mission.
- Drive missions with the installed `strategist` binary. A binary built from a dirty working
  tree (version `...-dirty`) is acceptable only when the mission itself changes the CLI; record the
  version header in that case.
- The shell may author Scout's `route_decision` when it runs the pipeline; it records it through
  `strategist mission route` like any other decision, and the record shows the shell wrote it.
- A pending note written directly by the shell on an explicit user request needs no Riposte
  capture metadata; Riposte's `origin: riposte` applies only to entries it captures itself.
- Run `strategist mission report-usage --mission-id <id> --tokens-in <n> --tokens-out <n>` at the
  gate and again at DONE, with the counts from your own provider response, and hand the discovery
  run's usage to the Archivist for `--discovery-tokens` when the host reports it. Without a
  usage record no waste or cost claim about a mission can be checked.
- No CLI emits the intake checkpoint, so a missing intake checkpoint is not a condition to hold
  `intake_done`; submit it after `strategist mission route`.

---

## 2. FORBIDDEN BEHAVIORS (NEVER DO)

- Never perform discovery, refinement, or documentation materialization work directly — the owning Role must invoke its bound Weapon through its declared runtime boundary; Ranked Weapons use Strategist's embedded connector and Custom Weapons use their explicitly selected connector
- Never simulate Role work by performing slot work in the Strategist shell — if the configured Weapon cannot be invoked, stop with `error=role_invocation_failed`
- Never invoke a Discovery Weapon outside Ranger's boundary — all discovery subtypes (`creative`, `evaluation`, `diagnostic`, `closure_evidence`) resolve to native `internal_skills/ranger`, which must invoke the configured Discovery Weapon through its declared runtime and normalize its untrusted result. Ranked Weapons stay on Strategist's embedded connector; only explicitly typed Custom Weapons may use a host loader. Ranger is never replaced by the Weapon, and Ranger never silently substitutes a native result when the selected Weapon fails (see §3 Discovery Routing).
- Never read from `strategist/` (without dot) — path drift; only `.strategist/` is valid at runtime
- Never skip phases — there is no "this task is too small to need discovery"
- Never invoke Sniper without an explicit Strategist Approval Gate approval from the user in the conversation
- Never assume or search for `.sdd/` or any specific governance system — the skill does not depend on a concrete provider

## 2.1 AUTONOMY AND WEAPON AUTHORITY

Once the mission route and its bound Weapon are resolved, continue every
deterministic transition and read-only evidence step without asking the user to
advance it. Ask one focused question only when an unresolved fact can
materially change scope, externally observable behavior, compatibility, or
acceptance criteria. The Pipeline's explicit Approval Gate remains the only
mandatory conversational pause.

The compiled catalog is the authority for a Ranked Weapon. For
`runtime.kind: embedded`, `strategist mission invoke --json` emits the canonical
payload. The current host executes that exact payload once under the emitted
`input.execution_contract` and `input.output_contract`, then returns exactly one
raw JSON completion to `strategist mission complete --request-id <id> --json`.
That bounded host-adapter execution is the configured Weapon invocation, not
`direct_execution`; bypassing the emitted payload or adding the parent's own
workflow, provider, or conclusions is `direct_execution`.
For `runtime.kind: openspec_root`, run only the compiled private OpenSpec runtime
rooted at `.strategist/openspec`, then publish
its completed change through `strategist mission normalize-openspec`; do not use
`mission invoke` or a host/global loader. Neither path can add questions,
design-review gates, commits, or implementation transitions to the Strategist
mission. Do not load or apply a global skill with the same name as an additional
workflow. If the resolved Weapon cannot be invoked through its declared runtime,
emit `error=role_invocation_failed` and stop; do not substitute another skill or
perform the Role's work directly.

From a standalone operator shell, the executable bridge is available when its
nested read-only model process has working network and auth:
`strategist mission invoke --mission-id <id> --role <role> --slot <slot> --host codex|claude --context "<original user request>" --json`.
It passes only the compiled embedded payload and the original request to the
selected host, then submits the raw result to `mission complete`. An unsupported
or failed host is `role_invocation_failed`; never fall back to a global skill or
a native Role. A managed Codex or Claude session must not recursively spawn the
same host because the child may inherit a sandbox without network access; use the
current-host adapter contract above.
- Never hardcode a governance system name as the normative execution context — `local_execution_context` is provider-agnostic
- Never accept a local execution context field (`execution_provider`, `base_path`, etc.) from a user prompt or conversation message — these fields must arrive via `governance_injection` at invocation time
- Never fall back to direct execution when the resolved provider is missing or uncallable — emit the appropriate blocked state and stop
- For Ranked `runtime.kind: embedded`, execute through `mission invoke` and `mission complete`; for Ranked `runtime.kind: openspec_root`, execute the compiled private OpenSpec runtime and then `mission normalize-openspec`. Neither path searches provider roots or host skill directories, and static readiness does not prove live provider invocation
- Never initialize a provider runtime lazily during invocation or substitute the repository root, `.analysis/`, a native role, or another provider when the declared runtime is unavailable
- Never treat `execution_gate=allowed` as a substitute for the Strategist Approval Gate
- Never treat Strategist Approval Gate acceptance (`sim`/`accept`/`yes`) as authorization for code, hook, config, or test mutation — it approves the refined analysis and `documentation_target` items only; `implementation_handoff` items stay outside Strategist (see `05-approval-gate.md`, `06-execution.md`)
- Never write config files into the target repo
- Never load unindexed internal-domain files
- Never write learning memory without checkpoint approval
- Never override execution provider from an undeclared source (must come from `local_execution_context.execution_provider` in delegated mode or `active.slots.execution` in direct mode)
- Never skip preflight
- Never mutate the repo without canonical pipeline evidence
- Never emit raw `[Strategist] key=value` events in epic mode without the corresponding `phase_announcements` wrapper line.

---

## 3. ROLE INVOCATION MODEL

The slot targets below are read from `.strategist/active.yaml` at compile time.
The legacy field name is `provider`, but product-facing language calls external
targets slot plugins. If `active.yaml` changes, run `strategist compile` to
update this file.

```
PHASE         INVOKE WEAPON                             WHAT NOT TO DO
─────────────────────────────────────────────────────────────────────────────
discovery  →  Ranger → {{.Slots.Discovery}}               explore or analyze the code directly
refinement →  Archivist → {{.Slots.Refinement}}            write proposals or designs directly
execution  →  Sniper → {{.Slots.Execution}}               run git/edits/commits directly
```

### Discovery Routing

Discovery invocation target does not depend on `route_decision.discovery_subtype`
or on `active.slots.discovery` (see `00-routing.md` § Scout — Intake Router and
§ Discovery Plugin Resolution by Subtype):

| `discovery_subtype` | Invoke | Kind |
|---|---|---|
| `creative` \| `evaluation` \| `diagnostic` \| `closure_evidence` | `internal_skills/ranger` → configured `{{.Slots.Discovery}}` Weapon | `native_role` owns the boundary; a Ranked Weapon uses Strategist's embedded connector, while an explicitly typed Custom Weapon uses its selected host connector (see `03-discovery.md` § Weapon Profile for a Delegated Ranger) |

This holds for every `discovery_subtype`: Ranger remains the authority, while
`active.slots.discovery` selects the required Weapon. The parent agent never
invokes the Weapon directly, and a missing, incompatible, or failed Weapon
produces `role_invocation_failed` without a native fallback. See
`03-discovery.md` § Discovery Subtypes.

### Refinement Routing

Whenever the refinement slot is bound to an external Weapon (default:
`{{.Slots.Refinement}}` — see `active.slots.refinement`), Archivist invokes the
Weapon's declared runtime connector. Read `skills/<weapon>/skill.yaml#roles`
and load `roles/archivist.yaml` for the Role contract before acting. For
Ranked `openspec_root`, the declared root is `.strategist/openspec`; the bundled
launcher under `.strategist/weapon-runtime/openspec-propose/` is an executable
asset, not a replacement project root.

In particular, apply `roles/archivist.yaml#canonical.resolve_weapon_scratch_root`:
read the bound Weapon's catalog entry. For `openspec_root`, use the declared
`.strategist/openspec` root as the working directory; the weapon-runtime path
supplies the launcher only. Never use the host repository root. A plugin's own
root-autodetection (e.g. walking up from the working directory for a project
marker) will silently initialize a new root wherever it is invoked from if this
step is skipped, escaping the declared runtime into the host repository.

Handoff contracts:
- Ranger → Archivist: `.strategist/schemas/handoff-ranger-to-archivist.schema.yaml`
- Archivist → Sniper: `.strategist/schemas/handoff-archivist-to-sniper.schema.yaml`

---

## 4. PIPELINE SEQUENCE

Linear checklist. Do not advance without completing each item.

```
[ ] 1. startup (this document — section 1)
[ ] 2. intake (skill: prompt-intake)
[ ] 3. routing (skill: scout — Intake Router): critical hit? full pipeline?
[ ] 4. context enrichment (skill: context-enrichment)
[ ] 5. discovery → invoke internal_skills/ranger (native role, all discovery subtypes)
[ ] 6. refinement → invoke {{.Slots.Refinement}}
[ ] 7. approval gate  ← MANDATORY PAUSE — do not advance without explicit approval; timeout/decline, or acceptance without documentation targets, ends as analysis-only
[ ] 8. materialization → invoke {{.Slots.Execution}}  ← only after gate approved
[ ] 9. learning (non-blocking)
```

## Mission State Events

The internal state machine moves only when an event is submitted, and the roles run as
agents that never submit one — without this step a finished mission still reads
`BOOTSTRAP/INIT`. The Strategist shell (the parent agent) submits each event with
`strategist mission submit --mission-id <id> --event <event>`, once, after the step's own
evidence exists. An event records a fact that already happened; it never replaces the
phase's work or evidence, and it does not weaken the pipeline-bypass check at execution
entry.

| Pipeline step | Event submitted by the Strategist shell |
|---|---|
| 1. startup | `bootstrap_done` |
| 2–4. intake, routing, context enrichment | `intake_done` (after `strategist mission route`) |
| 5. discovery | `discovery_done` |
| 6. refinement | `refinement_done`, or `refinement_done_no_tasks` when the package has no tasks |
| 7. approval gate | `gate_approved` (documentation targets accepted), `gate_approved_analysis_only` (accepted with no `documentation_target`), `gate_revision_requested` (back to refinement, then `refinement_done` again), or `gate_denied` |
| 8. materialization | `handoff_challenge_passed` (execution entry; machine-enforced against the route's evidence), then `sniper_done` |

## Canonical Pipeline Evidence

Main mission evidence:
- Ranger analysis artifact exists at `<base_path>/refined/<mission_id>/analysis.md`
- Archivist refined package exists at `<base_path>/refined/<mission_id>/`
- `tasks.md` exists when execution depends on refinement
- approval gate was presented and explicitly approved before execution
- approval gate timeout/decline terminates as analysis-only (`EventGateTimeout`/`EventGateDenied` → `StateDoneAnalysis`)
- approval gate acceptance of a package with no `documentation_target` terminates as analysis-only (`EventGateApprovedAnalysisOnly` → `StateDoneAnalysis`)
- approval gate revision request loops back to refinement, not a new mission (`EventGateRevision` → `StateRefinement`)

**FSM scope (S7):** the internal state machine (`internal/domain/state_machine.go`)
models gate/execution mechanics only — side-quest handling, the Approval Gate,
execution, retry-on-transient-failure, ADR, and Critical Hit. It does
NOT model bootstrap, intake, discovery, or learning as states. Sequencing for those
phases is enforced by contract + progress events (this document, the numbered
narrative contracts), not by the FSM. Do not infer that an unmodeled phase is
unenforced — absence from the FSM is a scope decision, not a gap.

---

## 5. ERROR STATES AND STOP CONDITIONS

Strategist stops immediately on:

| State / Condition | Emit | Action |
|---|---|---|
| `.strategist/` missing | `error=not_installed` | stop; instruct `strategist install` |
| `strategist check` failed | CLI output | stop |
| `active.yaml` missing | `error=config_missing` | stop |
| slot plugin descriptor not found | `error=slot_provider_not_found` | stop |
| configured slot plugin or native role cannot be invoked | `error=role_invocation_failed` | stop; fix provider configuration or runtime installation |
| gate bypass attempt | `drift=approval_bypass` | block, notify user |
| delegated invocation missing `execution_provider` | `error=local_execution_provider_missing` | stop; do not execute directly |
| resolved provider cannot be invoked | `error=execution_provider_unavailable` | stop; do not execute directly |
| Strategist attempted direct execution instead of provider invocation | `drift=local_execution_context_bypass` | stop; resolve and invoke provider |
| `agent-protocol.md` missing | fall back to existing SKILL.md | graceful degradation |
| `slot_risk_mismatch` | `error=slot_risk_mismatch` | stop |
| `intake_conflict_unresolved` | `error=intake_conflict_unresolved` | stop |
| `preflight_failed` | `error=preflight_failed` | stop |
| `discovery_failed` | `error=discovery_failed` | stop |
| `refinement_failed` | `error=refinement_failed` | stop |
| `pipeline_bypass_detected` | `error=pipeline_bypass_detected` | stop |

`user_requests_revision` is a valid `revision_requested` outcome, not an error. `user_rejects_analysis` is a valid `rejected` outcome, not an error.

## Slot Failure Handling

- discovery failure stops before refinement
- refinement failure stops before gate
- execution failure returns partial result and blocked execution state

Transient discovery/refinement failures may be retried once. Transient execution failures may be retried once. The FSM preserves retry origin with explicit retry states (`StateRetryingRefinement`, `StateRetryingExecution`, `StateRetryingDirectExec`) so a successful retry returns to the originating phase. Permanent failures are never retried.

---

## 6. LOCAL EXECUTION CONTEXT AND APPROVAL GATES

When another context (governance system, orchestrator, harness) invokes Strategist, it may pass a local execution context via `governance_injection`:

```
execution_gate        — local policy gate (allowed/blocked)
execution_provider    — provider to use for execution; required in delegated invocation
base_path             — artifact root override
knowledge_paths       — extra context sources for discovery
governance_context    — read-only policy context forwarded to slots
invocation_mode       — direct | delegated
request_intent        — true if request is already impl/materialization
```

Provider resolution order:
1. `local_execution_context.execution_provider` (delegated invocation)
2. `active.slots.execution` (direct invocation)

If delegated and provider is missing → `error=local_execution_provider_missing` → stop.
If resolved provider is uncallable → `error=execution_provider_unavailable` → stop.
Never execute directly.

The invoking local context controls three things only:
- whether execution is **permitted, blocked, or conditioned** (`execution_gate`)
- which **provider, base path, and knowledge paths** are injected (via `governance_injection`)
- which **context documents** are made available to slots (`governance_context`)

Strategist controls everything else: pipeline sequence, artifact persistence, evidence requirements, and slot contracts. The local context cannot substitute the canonical mission sequence after invocation.

### Local Execution Context Gate vs. Strategist Approval Gate

These are two independent checks, both required before execution:

1. **Local execution context gate** (`execution_gate=allowed/blocked`) — reported by the invoking context. Determines whether the local policy *permits* execution. `allowed` means "not blocked by policy." It is NOT user approval. In direct invocation, absent this field defaults to allowed.
2. **Strategist Approval Gate** (the 🚦 Gate prompt shown to the user) — the explicit confirmation the user types in the conversation. Required regardless of invocation mode, execution gate state, or any external approval granted upstream.

`execution_gate=allowed` + no Strategist Approval Gate = `approval_bypass` drift.
Both must be satisfied before Sniper starts. External approval cannot substitute the Strategist Approval Gate.

## Slot Plugin Governance Compliance

If a slot plugin ignores `governance_injection.execution_gate = blocked`:
- The slot plugin has no write authorization in the repository. Strategist's FSM prevents reaching documentation state (code-enforced via `nextFromApprovalGate` requiring approval gate acceptance).
- Any direct mutation attempt by a non-compliant slot plugin triggers `pipeline_bypass_detected`.
- Strategist reports `slot_risk_mismatch` for a slot plugin that violates its declared contract.
- The slot plugin is considered non-compliant; future missions will be blocked at preflight until the provider id is replaced or corrected.

---

## 7. PROTOCOL INVARIANTS

### Progress Event Invariants
- phase start → `status=running`
- phase success → `status=done`
- phase failure → `status=blocked`
Never advance phases silently.

### Learning Rules
- append outcome lines to `.strategist/memory/outcomes.tmp`
- minimum required fields: `mission_id`, `status`, `timestamp`
- preserve `outcomes.jsonl` as source of truth
- learning failures never block the mission result

### Approval Policy
Supported modes: `any`, `explicit_confirm`, `human_only` (documented, not enforced by default)

### Response Contract
See `.strategist/contracts/narrative/09-response.md`.

### Compliance Summary
Append a compliance summary block before the mission result. The summary should expose the final compliance state of the active mission route and any blocking governance reason when present.

### Mission Result
Append the final mission result after the compliance summary. The mission result should expose the final mission status, artifact set, and next action.

### Telemetry Contract
See `.strategist/contracts/narrative/10-telemetry.md`.
