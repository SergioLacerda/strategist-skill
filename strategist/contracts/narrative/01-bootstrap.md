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
- when `governance_injection` is present: resolve and expose `governance_source` and `governance_adapter` in bootstrap diagnostics
- when no `governance_injection`: set `governance_mode=standalone`, `governance_source=none`

## Evidence

- bootstrap origin
- selected persona
- selected output profile
- `governance_mode` + `governance_source` (required in bootstrap diagnostic block)
