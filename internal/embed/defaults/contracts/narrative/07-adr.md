---
phase: adr
slot: execution
requires_approval: true
contract: execution_task
---
# Strategist — Contract 07: ADR

## Purpose

Create an ADR (Architectural Decision Record) when the refined work contains decisions worth documenting. Evaluated by Archivist during refinement; approved and executed at the main gate. Opportunity Attack is the ADR evaluation routine only; it does not evaluate implementation completion and does not move analysis cards.

## Inputs

- refined package under `<base_path>/refined/<mission_id>/`
- gate approval of the `[OA-ADR-{mission_id}]` side quest

## Outputs

- optional ADR artifact, destination resolved from `active.yaml#adr.canonical_path`:
  - configured → `<adr.canonical_path>/<adr_filename>` only (see Canonical Destination Resolution below)
  - absent → `<base_path>/archived/<mission_id>-adr.md` (unchanged fallback)

## Required Behavior

- **Archivist** evaluates ADR necessity after writing all four refined artifacts, using the criteria in `machine/opportunity-attack.yaml`
- If criteria met → Archivist surfaces `[OA-ADR-{mission_id}]` at the approval gate as a side quest
- If user approves at gate → **Sniper** resolves the destination (below) and creates the ADR as an execution task
- If user declines at gate → ADR is not created; outcome logged
- Pending/refined card closure remains a **Critical Hit** responsibility and requires the closure evidence defined in `11-critical-hit.md`

## Canonical Destination Resolution

Before writing, Sniper reads `active.yaml#adr.canonical_path` (optional string, project-relative path; no default — absence is a fully supported state, not an error):

- **Configured** → the ADR is written **only** to `<adr.canonical_path>/<adr_filename>`. It is not also written to `<base_path>/archived/`. This avoids two sources of truth for the same decision.
- **Absent** → today's behavior is unchanged: `<base_path>/archived/<mission_id>-adr.md`.

This field is never auto-detected or inferred from an existing `docs/` treasure chest declaration — a project must opt in explicitly, consistent with this skill's doctrine of never writing to a destination it wasn't told about.

### Filename resolution when `canonical_path` is configured

Sniper scans `adr.canonical_path` for existing files matching `^\d+-.*\.md$`:

- **Convention found** → take the highest numeric prefix, zero-padded to the same width, increment by one, and name the file `<NNNN>-<slug-from-title>.md`, continuing the project's own sequence.
- **No convention found** (empty or non-numbered directory) → fall back to `<mission_id>-adr.md`, matching the naming already used under `<base_path>/archived/`.

This scan is read-only and runs at materialization time, not at discovery/refinement time, so numbering reflects the state of the directory at the moment of writing rather than a stale snapshot from an earlier phase.

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
