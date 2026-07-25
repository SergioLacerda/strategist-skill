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
- resolved slot providers
- `governance_mode` — `standalone` or the adapter name (e.g. `sdd`, `custom`)
- `governance_source` — origin of the active governance (path or adapter id; `none` in standalone mode)
- `governance_adapter` — adapter responsible for governance injection, if any

## Required Behavior

- emit `[Strategist] pipeline=starting`
- prefer compiled artifacts when fresh
- fall back to YAML sources when compiled artifacts are stale or absent
- resolve chat language from `active.language.chat`
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

## Evidence

- bootstrap origin
- selected persona
- selected output profile
- `governance_mode` + `governance_source` (required in bootstrap diagnostic block)
