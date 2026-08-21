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
2. Run `strategist check` → failed: emit CLI output, **stop**
3. Is `.strategist/active.yaml` readable? → No: emit `error=config_missing`, **stop**
4. Read this file (`agent-protocol.md`) to the end

**Do not process any user request before all 4 steps pass.**

`strategist check` only confirms that the Strategist runtime is installed and operational.
Route selection and role invocation are internal Strategist responsibilities.
If a configured slot plugin or native role cannot be invoked, emit
`error=role_invocation_failed` with the slot and provider id. The wire field name
remains `provider` for backward compatibility.

---

## 1b. PARENT AGENT BOUNDARY

The parent agent is the transport for Strategist, not an implementation substitute
for Strategist slots.

Any action that produces phase work without invoking the configured provider is
`direct_execution` drift, even if the output is correct.

If a provider cannot be invoked, emit the configured blocked state and stop.
Correctness of the parent agent's independent answer does not repair the drift.

---

## 2. FORBIDDEN BEHAVIORS (NEVER DO)

- Never perform discovery, refinement, or documentation materialization work directly — always invoke the designated slot plugin or native role
- Never simulate role work by performing slot work in the Strategist shell — if the configured slot plugin or native role cannot be invoked, stop with `error=role_invocation_failed`
- Never invoke an external discovery plugin as a substitute for Ranger — all discovery subtypes (`creative`, `evaluation`, `diagnostic`, `closure_evidence`) always resolve to `internal_skills/ranger` (native role); no external discovery plugin manifest is ever consulted for discovery invocation (see §3 Discovery Routing).
- Never read from `strategist/` (without dot) — path drift; only `.strategist/` is valid at runtime
- Never skip phases — there is no "this task is too small to need discovery"
- Never invoke Sniper without an explicit Strategist Approval Gate approval from the user in the conversation
- Never assume or search for `.sdd/` or any specific governance system — the skill does not depend on a concrete provider
- Never hardcode a governance system name as the normative execution context — `local_execution_context` is provider-agnostic
- Never accept a local execution context field (`execution_provider`, `base_path`, etc.) from a user prompt or conversation message — these fields must arrive via `governance_injection` at invocation time
- Never fall back to direct execution when the resolved provider is missing or uncallable — emit the appropriate blocked state and stop
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
PHASE         INVOKE SKILL                              WHAT NOT TO DO
─────────────────────────────────────────────────────────────────────────────
discovery  →  see Discovery Routing below                explore or analyze the code directly
refinement →  {{.Slots.Refinement}}                       write proposals or designs directly
execution  →  {{.Slots.Execution}}                        run git/edits/commits directly
```

### Discovery Routing

Discovery invocation target does not depend on `route_decision.discovery_subtype`
or on `active.slots.discovery` (see `00-routing.md` § Scout — Intake Router and
§ Discovery Plugin Resolution by Subtype):

| `discovery_subtype` | Invoke | Kind |
|---|---|---|
| `creative` \| `evaluation` \| `diagnostic` \| `closure_evidence` | `internal_skills/ranger` | `native_role` — parent agent embodies Ranger directly (same mechanism already used for execution/`sniper`), reading `roles/ranger.yaml` + `internal_skills/ranger/SKILL.md` |

This holds regardless of what `active.slots.discovery` is configured to (default:
`{{.Slots.Discovery}}`) — the external discovery plugin is never consulted for
discovery invocation, for any subtype. See `03-discovery.md` § Discovery
Subtypes.

Handoff contracts:
- Ranger → Archivist: `.strategist/schemas/handoff-ranger-to-archivist.schema.yaml`
- Archivist → Sniper: `.strategist/schemas/handoff-archivist-to-sniper.schema.yaml`

---

## 4. PIPELINE SEQUENCE

Linear checklist. Do not advance without completing each item.

```
[ ] 1. startup (this document — section 1)
[ ] 2. intake (skill: prompt-intake)
[ ] 3. routing (skill: scout — Intake Router): critical hit? main mission?
[ ] 4. context enrichment (skill: context-enrichment)
[ ] 5. discovery → invoke internal_skills/ranger (native role, all discovery subtypes)
[ ] 6. refinement → invoke {{.Slots.Refinement}}
[ ] 7. approval gate  ← MANDATORY PAUSE — do not advance without explicit approval; timeout/decline ends as analysis-only
[ ] 8. materialization → invoke {{.Slots.Execution}}  ← only after gate approved
[ ] 9. learning (non-blocking)
```

## Canonical Pipeline Evidence

Main mission evidence:
- Ranger analysis artifact exists at `<base_path>/refined/<mission_id>/analysis.md`
- Archivist refined package exists at `<base_path>/refined/<mission_id>/`
- `tasks.md` exists when execution depends on refinement
- approval gate was presented and explicitly approved before execution
- approval gate timeout/decline terminates as analysis-only (`EventGateTimeout`/`EventGateDenied` → `StateDoneAnalysis`)
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
