---
phase: bootstrap
requires_approval: false
slot: null
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
- governance mode

## Required Behavior

- emit `[Strategist] pipeline=starting`
- prefer compiled artifacts when fresh
- fall back to YAML sources when compiled artifacts are stale or absent
- resolve chat language from `active.language.chat`
- resolve docs language from `active.language.docs`

## Evidence

- bootstrap origin
- selected persona
- selected output profile
