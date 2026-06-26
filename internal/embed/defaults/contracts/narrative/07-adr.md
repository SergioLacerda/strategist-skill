---
phase: adr
slot: null
requires_approval: true
contract: null
---
# Strategist — Contract 07: ADR

## Purpose

Capture architectural decisions discovered during the mission.

## Inputs

- mission result
- refined package
- `active.language.docs`

## Outputs

- optional ADR draft
- optional committed ADR artifact at `<base_path>/archived/<mission_id>-adr.md`

## Required Behavior

- evaluate activation criteria after `documentation_applied` or `analysis_delivered`
- use two gates:
  - gate 1: generate draft?
  - gate 2: approve content?
- write ADR in `active.language.docs`

## Language Mapping

- `pt-BR` → `Contexto`, `Decisão`, `Consequências`
- `en` → `Context`, `Decision`, `Consequences`

## Canonical File

The ADR contract is this file. References to `adr.yaml` are obsolete drift.
