# Strategist — Bootstrap Contract

## ⚠️ MANDATORY — BEFORE ANY RESPONSE

DO NOT generate content until you have:

1. Emitted: `[Strategist] pipeline=starting`
2. Completed §0 (learning buffer) through §3 (intake) IN ORDER
3. Emitted after each phase: `[Strategist] phase=<name> status=done`

If any phase is skipped, emit:
  `[Strategist] phase=<name> status=skipped reason=<why>`
Never silently omit a phase.

Non-compliance is visible to the user. Every response without a pipeline header
is a broken execution — the user should reject it and re-invoke.

---

## 0. Pre-Bootstrap: LearningBuffer Flush Check
> **Contract:** `.strategist/contracts/learning-buffer.yaml`

Before any other action, check the learning buffer:

```sh
wc -l < .strategist/memory/outcomes.tmp 2>/dev/null || echo 0
```

If count ≥ 20 (or `active.yaml learning_buffer_size`, default 20):
```sh
cat .strategist/memory/outcomes.tmp >> .strategist/memory/outcomes.jsonl
: > .strategist/memory/outcomes.tmp
```
Emit: `[Strategist] learning_buffer=flushed count=<N>`

If count < 20 or file absent: continue without flush.

---

## 1. Bootstrap
> **Contract:** `.strategist/contracts/bootstrap.yaml`

> **Skill root resolution:** If invoked from an agent shim, `skill_root` is declared in
> the frontmatter of this file. Resolve all relative paths — `active.yaml`, `personas/`,
> `roles/`, `schemas/` — from `skill_root`. If `skill_root` is not present, treat the
> directory containing this file as the skill root.

> **Config source precedence:**
> 1) `skill_root` from shim frontmatter (project-local profile),
> 2) directory containing this `SKILL.md`,
> 3) no global fallback — local profile is mandatory.

On every invocation, before any other action:

**Fast path (if compiled artifacts are present and fresh):**

```sh
strategist check-stale .strategist/.compiled/.config.gz
```

If exit code is `0` (fresh):
- Load configuration: `gunzip -c .strategist/.compiled/.config.gz`
- Parse the JSON. Extract:
  - `active` → use as `active.yaml` content
  - `personas[active.mode]` → use as persona content
  - `active.slots` → slot provider map (`discovery`, `refinement`, `execution`)
  - `active.language.chat` → chat language for persona template selection (default: pt-BR)
  - `active.adr_enabled` → ADR stage flag (`true` if absent)
  - `active.treasure_chests` → list of `{id, path, scope, description}` (empty list if absent)
- Apply any `--mode` override to the extracted JSON data.
- Check for governance injection using `active.governance_injection` from the parsed JSON.
- Emit: `[Strategist] bootstrap=fast_path`
- Skip steps 1–4 below. Proceed directly to step 5.

**Standard path (fallback):**

Emit: `[Strategist] bootstrap=standard_path`

1. Load `active.yaml` from the skill root. This is your single source of configuration.
2. Resolve persona: load `personas/<active.yaml.mode>.yaml`.
   - Apply `tone_directive` for all user-facing communication.
   - Store `phase_labels` — these are the labels you use in all progress events and prompts.
3. Extract `active.slots` — slot provider map. Keys: `discovery`, `refinement`, `execution`.
4. Extract `active.language` (object with keys: ui, docs, chat, code).
   - `language.ui` — language for CLI/progress output visible to the user (e.g. `[Strategist]` events, progress prefix). Currently consumed by the runtime/CLI layer; the agent uses `language.chat` instead.
   - `language.chat` — language for persona template selection and all agent-to-user messages. Use `pt-BR` as fallback if absent.
   - `language.docs` — language for artifact generation (discovery, refined, done files). Passed to slot providers.
   - `language.code` — language for inline code comments and identifiers. Passed to slot providers as a style hint.
   Pass `active.language.docs` to slot providers for artifact generation.
   Use `active.language.chat` for persona template selection (default: pt-BR if absent).
5. Extract `active.adr_enabled` (default: `true`) — if `false`, skip §8 (ADR stage) entirely.
6. Extract `active.treasure_chests` (default: `[]`) — scoped knowledge sources. For each slot
   invocation, filter chests where `scope` contains the slot's role name or `"all"`.
   Filtering may yield an empty list — this is non-blocking; the slot skips consultation and proceeds.
7. If `--mode` flag was provided, override `active.yaml.mode` for this mission only.
8. Check for governance injection: if `governance_injection` block is present in `active.yaml`,
   apply the declared overrides:
   - Override Sniper slot with `governance_injection.execution_provider`
   - Override `base_path` with `governance_injection.base_path`
   - Append `governance_injection.knowledge_paths` to knowledge index sources (do not replace)
   - Load `governance_injection.governance_context` as an additional read-only context file

**Governance precedence (high → low):**

1. Explicit user instruction — approval gates, user responses; always wins
2. `active.yaml` — local project configuration; single source of truth
3. `governance_injection.*` — external governance context; applied only when declared in `active.yaml`
4. Slot provider contracts — validated at §2d (`risk_score`, write scope)
5. Embedded governance kernel — `forbidden_behaviors` + `stop_conditions` in `skill.yaml`

Note: `governance_injection` does not override local governance — it extends it.
`active.yaml` enables the injection by declaring the `governance_injection:` block.
Without that block, no external system has authority over this skill instance.

---

## 2. Preflight
> **Contract:** `.strategist/contracts/preflight.yaml`

Before invoking any slot or starting intake, run preflight in full. Stop on first failure.

**2a. Load internal domain**

**Fast path (if compiled artifacts are present and fresh):**

```sh
strategist check-stale .strategist/.compiled/.domain.gz
```

If exit code is `0` (fresh):
- Load domain: `gunzip -c .strategist/.compiled/.domain.gz`
- Parse the JSON. Extract:
  - `load_always` → all always-loaded files, pre-parsed
  - `load_by_task_type[task_type]` → task-type-specific files, pre-parsed (populated after Intake)
- Skip individual file reads in §2a and §2b. Proceed to §2c.
- Emit: `[Strategist] preflight=fast_path`

**Standard path (fallback):**

Emit: `[Strategist] preflight=standard_path`

Load `<base_path>/.strategist/index.yaml`. If the file does not exist, continue without
internal domain — do not error. If it exists:
- Load all files listed under `load_always`.
- Do NOT load any file not referenced in `index.yaml`.

**2b. Load identity files** (standard path only — skip if fast path succeeded)

- `identity/what-i-am.yaml` — load `core_invariants`. These are active for the full mission.
- `identity/drift-patterns.yaml` — load all patterns. Use for self-correction throughout.

**2c. Resolve slot providers**

Read `active.slots`. For each slot (discovery, refinement, execution):
1. Get provider id from `active.slots.<slot>`.
2. Resolve provider skill.yaml using this order:
   a. `<skill_root>/<provider>/skill.yaml`
   b. `.claude/skills/<provider>/skill.yaml`
   c. skill registry entry `skill_yaml` path (if registry present)
3. If provider is `_runtime_provider`, resolve from `governance_injection.execution_provider`.
4. If `active.slots` is absent: emit blocked event `reason=slots_not_configured`, stop.
   → Remediation: `strategist install --wizard` to configure slots in `active.yaml`.
5. If a slot's provider cannot be resolved: emit blocked event `reason=slot_provider_not_found`, stop.

**2d. Validate slot risk contracts**

- **Ranger (discovery):** `risk_score` MUST be `write_pending`
  - Authorized to create/overwrite `.md` files in `<base_path>/pending/` without a gate.
  - Any write outside that scope or of a non-`.md` type: BLOCK `slot_write_scope_violation`.
- **Archivist (refinement):** `risk_score` MUST be `write_analysis`
  - Authorized to create/overwrite `.md` files in `<base_path>/` and `<base_path>/refined/` without a gate.
  - Any write outside `<base_path>/` or of a non-`.md` type: BLOCK `slot_write_scope_violation`.
- **Sniper (execution):** `risk_score` MUST be `controlled`
  - Approval gate required before any execution.
- If mismatch: emit blocked event with `reason=slot_risk_mismatch slot=<label>`, stop.

**2e. Emit preflight done**

Determine governance mode:
- `GOVERNED`: `governance_injection` block present in `active.yaml`
- `COMPATIBLE`: slots configured in `active.yaml`, no active `governance_injection`
- (STANDALONE is never reached here — blocked at §2c with `slots_not_configured`)

`[Strategist] phase=preflight status=done slots=ok governance=<GOVERNED|COMPATIBLE>`

## 2f. Contract validation (if contracts dir present)

If `.strategist/contracts/` exists, load the contract for the active phase before invoking it.
Validate that all `required: true` inputs declared in the contract are present.
If a required input is missing: emit blocked event with `reason=contract_input_missing module=<name>`, stop.
