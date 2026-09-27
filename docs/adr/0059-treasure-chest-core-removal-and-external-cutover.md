# ADR-0059 — Treasure Chest Core Removal and External Cutover

**Status:** Accepted  
**Date:** 2026-09-27  
**Mission:** `20260927-treasure-chest-removal`  
**Related:** [ADR-0032](0032-external-skill-cli-embedding-and-treasure-chest-ownership.md), [ADR-0040](0040-treasure-chest-in-repo-isolation-staging.md), [ADR-0049](0049-atlas-treasure-chest-boundary.md), [ADR-0056](0056-jewelcrafter-public-role-and-bau-tesouro-binding.md)

## Context

Strategist currently contains a complete `treasure-chest/` Go package with a
CLI surface, domain and storage behavior, indexing, curation, tests, and
multiple direct production consumers. The same domain is intended to be
provided by the external `skill-for-hire` Treasure Chest Weapon. Keeping both
implementations active would create duplicate ownership and could route users,
Ranger, Archivist, or evaluation code to the wrong implementation.

The external material currently available to this workspace is contract-only;
it does not prove that a runnable provider, compatible API, delegate, probe, or
health signal exists. Removing the local package before those facts are
verified would remove the only known working implementation and strand the
existing CLI and consumers.

## Decision

1. **Remove the local core only through a staged cutover.** The in-repository
   package remains intact until external readiness, consumer migration, data
   migration, parity, generated-source, release, and repository-audit gates
   pass.

2. **Make the external Weapon the sole production owner after cutover.** The
   `skill-for-hire` Weapon owns Treasure Chest domain, storage, indexing,
   curation, and migration behavior. The host owns routing, permissions,
   provenance envelopes, compatibility diagnostics, and fail-closed activation.

3. **Use an explicit mediated boundary for consumers.** Ranger and Archivist
   request offline Treasure Chest information through the JEWELCRAFTER
   boundary. The supported CLI namespace delegates through an explicit adapter
   or external command contract during migration. No consumer may retain a
   hidden direct import of the removed package after cutover.

4. **Fail closed when external readiness is incomplete.** Runtime activation
   requires package validity, immutable identity, trusted provenance, API
   compatibility, delegate wiring, a successful entrypoint probe, and health.
   A catalog entry, lock, generated mirror, cache, or contract stub is not
   sufficient. Missing or contradictory evidence returns an actionable
   unavailable, degraded, blocked, or unverified result; it never silently
   activates the old local package or another Role's implementation.

5. **Preserve a deliberate compatibility seam during migration.** The
   `strategist treasure-chest ...` namespace may remain host-visible while the
   external contract is admitted and parity is established. It must delegate
   explicitly or report the readiness/migration error. The seam is transitional
   and does not justify keeping two production owners after the breaking
   release boundary.

6. **Preserve data and provenance.** Before ownership transfer, the migration
   inventories configured Treasure Chest roots, jewels, potions, chest
   manifests, indexes, and related `.strategist` state. An idempotent manifest
   records source and destination identity, item status, provenance, checksums
   where available, and rollback information. Removing code never deletes
   legacy data.

7. **Allow a test-only contract mock.** A deterministic mock or fixture may
   validate request, response, error, and compatibility semantics when the
   external repository is unavailable. Mock responses must identify synthetic
   provenance and `provider_mode: mock`; a production profile selecting the
   external provider must reject the mock rather than selecting it silently.

## Migration Boundary

```text
Ranger / Archivist
        │ bounded offline request
        ▼
JEWELCRAFTER-mediated Treasure Chest boundary
        │ explicit delegation and provenance
        ▼
skill-for-hire Treasure Chest Weapon
        │ external domain, storage, indexing, curation, migration owner
        ▼
legacy data + idempotent migration manifest
```

The local package, its direct imports, CLI registration, architecture-test
exceptions, coverage and quality entries, generated/default references, and
obsolete tests are removable only after the boundary above is active and the
repository-wide audit finds no unexplained dependency.

## Consequences

### Positive

- Treasure Chest has one production owner after the cutover.
- Ranger and Archivist receive bounded, source-backed offline results through a
  governed role boundary.
- External provider failure is visible and recoverable instead of producing a
  silent local fallback or conflicting state.
- Migration can be rehearsed, repeated, and rolled back without deleting
  legacy data.
- Local CI remains hermetic through an explicitly marked contract mock.

### Costs and constraints

- The migration is multi-wave and cannot be completed by deleting the
  `treasure-chest/` directory alone.
- The final external API, identity, provenance, and runtime readiness evidence
  remain inputs supplied by `skill-for-hire`.
- Compatibility, data manifests, source/runtime regeneration, and repository
  audits require additional implementation work.
- Static contract or mock evidence cannot certify live external-provider
  readiness.

## Non-Goals

- This ADR does not remove the local package or alter consumers by itself.
- This ADR does not import, implement, or certify the external
  `skill-for-hire` repository.
- This ADR does not delete Treasure Chest data, indexes, manifests, jewels, or
  potions.
- This ADR does not change JEWELCRAFTER's exclusive BAU DO TESOURO ownership or
  introduce a new Treasure Chest registration.
- This ADR does not authorize silent fallback to the local package, Mock,
  JEWELCRAFTER internals, or a native replacement.

## Implementation Handoff

Implementation remains a separate handoff covering external provider
admission, the mediated boundary, Ranger/Archivist and CLI consumers, the
idempotent migration manifest, contract mock, generated/default parity,
coverage and quality manifests, tests, audits, and eventual package removal.
Approval of this ADR records the architecture decision and does not by itself
authorize those code or runtime mutations.

## Operational Follow-up

Use `docs/runbooks/treasure-chest-external-cutover-and-removal.md`
for readiness diagnosis, migration, partial-write recovery, rollback, and
stop conditions.
