---
phase: adr
slot: execution
requires_approval: true
contract: execution_task
---
# Strategist — Contract 07: ADR

## Purpose

Create an ADR (Architectural Decision Record) when the refined work contains decisions worth documenting. Evaluated by Archivist during refinement; approved and executed at the main gate.

## Inputs

- refined package under `<base_path>/refined/<mission_id>/`
- gate approval of the `[OA-ADR-{mission_id}]` side quest

## Outputs

- optional ADR artifact at `<base_path>/archived/<mission_id>-adr.md`

## Required Behavior

- **Archivist** evaluates ADR necessity after writing all four refined artifacts, using the criteria in `machine/opportunity-attack.yaml`
- If criteria met → Archivist adds `[OA-ADR-{mission_id}]` to `opportunity_manifest` and surfaces it at the approval gate as a side quest
- If user approves at gate → **Sniper** creates the ADR as an execution task
- If user declines at gate → ADR is not created; outcome logged

## ADR Activation Criteria (evaluated by Archivist)

- `new_pattern`: new interface, contract, schema, or abstraction introduced
- `breaking_change`: field removed, signature changed, behavior changed
- `documented_tradeoff`: tasks.md/design.md describe a choice with discarded alternatives
- `new_external_dependency`: library, service, or protocol added

## Language Mapping

- Generate ADR headings and body content using the language selected by `active.language.docs`.
- `pt-BR` → `Contexto`, `Decisão`, `Consequências`
- `en` → `Context`, `Decision`, `Consequences`

## Canonical Machine Contract

`contracts/machine/adr.yaml`
