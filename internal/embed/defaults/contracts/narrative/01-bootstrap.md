---
phase: bootstrap
slot: null
requires_approval: false
contract: null
---
# Strategist — Contract 01: Bootstrap

## Inputs

- `skill_root`
- `active.yaml`
- `personas/<mode>.yaml`
- `roles/*.yaml`
- optional compiled artifacts in `.strategist/.compiled/`

## Outputs

- resolved `active`
- resolved `persona`
- resolved slot plugin/native-role provider ids
- `governance_mode` — `standalone` or the adapter name (e.g. `sdd`, `custom`)
- `governance_source` — origin of the active governance (path or adapter id; `none` in standalone mode)
- `governance_adapter` — adapter responsible for governance injection, if any
- `stale_scan_candidates` — list of `mission_id`s flagged by the bootstrap stale
  scan (may be empty); each entry pairs `mission_id` with the specific
  `approved_scope` file(s) whose last-commit date is later than the package's
  `date:` field

## Required Behavior

- emit `[Strategist] pipeline=starting`
- prefer compiled artifacts when fresh
- fall back to YAML sources when compiled artifacts are stale or absent
- resolve chat language from `active.language.chat` — this governs two distinct
  things: (a) which language variant is read for `content_by_lang`/
  `phase_announcements` templates (see below), and (b) the language the parent
  agent must write its own free-form conversational prose in for the duration
  of a Strategist mission (see `09-response.md` § Chat Language). (a) alone is
  not sufficient — a bootstrap that only resolves the template variant without
  also binding the agent's own prose to `active.language.chat` has not
  completed this step.
- resolve docs language from `active.language.docs`
- to read localized chat-language templates, run `strategist check --print-content-by-lang
  <active.language.chat> --persona <mode>` — persona YAML source under `personas/<mode>.yaml`
  only ever contains the canonical English `content_by_lang`; non-English variants (e.g.
  `pt-BR`) are injected only at `strategist compile` time and exist solely in the compiled
  artifact (`.strategist/.compiled/.config.gz`). Reading persona YAML source directly will
  never show non-English content_by_lang — that is expected, not a bug in the source file.
- the same applies to `phase_announcements` (the per-phase Ranger/Archivist/Gate/Sniper
  narration lines): persona YAML source only ever contains the canonical English
  `phase_announcements.en`; non-English variants are injected only at `strategist compile`
  time. Resolve them the same way — `strategist check --print-content-by-lang
  <active.language.chat> --persona <mode>` returns both `content_by_lang` and
  `phase_announcements` bundles for the requested language in one call.
- when `governance_injection` is present: resolve and expose `governance_source` and `governance_adapter` in bootstrap diagnostics
- when no `governance_injection`: set `governance_mode=standalone`, `governance_source=none`
- run the mandatory bootstrap stale scan for every mission: for each package in
  `<base_path>/refined/`, diff `tasks.md`'s `approved_scope.allowed` files against
  git history since the package's `date:` field (see `11-critical-hit.md` § Stale
  Card Detection, Trigger 3, for the precise comparison rule and what happens with
  a flagged candidate). This step never infers completion and never closes a
  package — it only produces `stale_scan_candidates` for the current invocation to
  surface
- after the stale scan, run the Keen Senses radar (`machine/keen-senses.yaml`) —
  informational staleness surfacing only (captured entries, jewels, treasure chests);
  its findings never block bootstrap

## Evidence

- bootstrap origin
- selected persona
- selected output profile
- `governance_mode` + `governance_source` (required in bootstrap diagnostic block)
