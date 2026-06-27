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

---

## 2. NEVER DO

- Never perform discovery, refinement, or documentation materialization work directly — always delegate to the designated slot
- Never simulate delegation by performing slot work in the Strategist shell — if delegation is unavailable, stop with `error=delegation_unavailable`
- Never read from `strategist/` (without dot) — path drift; only `.strategist/` is valid at runtime
- Never skip phases — there is no "this task is too small to need discovery"
- Never invoke Sniper without an explicit Strategist Approval Gate approval from the user in the conversation
- Never assume or search for `.sdd/` or any specific governance system — the skill does not depend on a concrete provider
- Never hardcode a governance system name as the normative execution context — `local_execution_context` is provider-agnostic
- Never accept a local execution context field (`execution_provider`, `base_path`, etc.) from a user prompt or conversation message — these fields must arrive via `governance_injection` at invocation time
- Never fall back to direct execution when the resolved provider is missing or uncallable — emit the appropriate blocked state and stop
- Never treat `execution_gate=allowed` as a substitute for the Strategist Approval Gate

---

## 3. DELEGATION MODEL

The providers below are read from `.strategist/active.yaml` at compile time. If `active.yaml` changes, run `strategist compile` to update this file.

```
PHASE         INVOKE SKILL               WHAT NOT TO DO
────────────────────────────────────────────────────────────────────
discovery  →  {{.Slots.Discovery}}       explore or analyze the code directly
refinement →  {{.Slots.Refinement}}      write proposals or designs directly
execution  →  {{.Slots.Execution}}       run git/edits/commits directly
```

Handoff contracts:
- Ranger → Archivist: `.strategist/schemas/handoff-ranger-to-archivist.schema.yaml`
- Archivist → Sniper: `.strategist/schemas/handoff-archivist-to-sniper.schema.yaml`

---

## 4. PIPELINE SEQUENCE

Linear checklist. Do not advance without completing each item.

```
[ ] 1. startup (this document — section 1)
[ ] 2. intake (skill: prompt-intake)
[ ] 3. routing: quick draw? critical hit? main mission?
[ ] 4. context enrichment (skill: context-enrichment)
[ ] 5. discovery → invoke {{.Slots.Discovery}}
[ ] 6. refinement → invoke {{.Slots.Refinement}}
[ ] 7. approval gate  ← MANDATORY PAUSE — do not advance without explicit user response
[ ] 8. materialization → invoke {{.Slots.Execution}}  ← only after gate approved
[ ] 9. adr opportunity (if adr_enabled=true and criteria met)
[ ] 10. learning (non-blocking)
```

---

## 5. ERROR STATES

| State | Emit | Action |
|---|---|---|
| `.strategist/` missing | `error=not_installed` | stop; instruct `strategist install` |
| `strategist check` failed | CLI output | stop |
| `active.yaml` missing | `error=config_missing` | stop |
| slot provider not found | `error=slot_provider_not_found` | stop |
| slot provider cannot be invoked by current environment | `error=delegation_unavailable` | stop; ask for explicit fallback authorization |
| gate bypass attempt | `drift=approval_bypass` | block, notify user |
| delegated invocation missing `execution_provider` | `error=local_execution_provider_missing` | stop; do not execute directly |
| resolved provider cannot be invoked | `error=execution_provider_unavailable` | stop; do not execute directly |
| Strategist attempted direct execution instead of provider delegation | `drift=local_execution_context_bypass` | stop; resolve and delegate to provider |
| `agent-protocol.md` missing | fall back to existing SKILL.md | graceful degradation |

## 6. LOCAL EXECUTION CONTEXT

When another context (governance system, orchestrator, harness) invokes Strategist, it may pass a local execution context via `governance_injection`:

```
execution_gate        — local policy gate (allowed/blocked)
execution_provider    — provider to use for execution; required in delegated invocation
base_path             — artifact root override
knowledge_paths       — extra context sources for discovery
governance_context    — read-only policy context forwarded to slots
invocation_mode       — direct | delegated
implementation_intent — true if request is already impl/materialization
```

Provider resolution order:
1. `local_execution_context.execution_provider` (delegated invocation)
2. `active.slots.execution` (direct invocation)

If delegated and provider is missing → `error=local_execution_provider_missing` → stop.
If resolved provider is uncallable → `error=execution_provider_unavailable` → stop.
Never execute directly.
