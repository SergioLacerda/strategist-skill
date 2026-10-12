# ADR-0070 — JEWELER boundary for the temporary Treasure Chest Weapon

**Status:** Accepted  
**Date:** 2026-10-11  
**Scope:** Sprint 1 only

## Decision

JEWELER is the only Strategist-owned access boundary for Treasure Chest. The
active ROSTER contains exactly one `treasure` Weapon: `treasure-chest-tmp@0.1.0`.
It is a host-runtime adapter over the complete root `treasure-chest/`
implementation and is invoked only through an explicit executable binding.

The production path is:

```text
Ranger or Archivist -> JEWELER -> ROSTER treasure-chest-tmp -> host executable -> root implementation
```

The temporary package is local to this repository and declares local provenance.
It does not claim Skill-for-Hire provenance. Its unavailable behavior is the
explicit `role_invocation_failed` path; native substitution, PATH discovery,
compatibility aliases, and local fallback are forbidden.

The old `strategist treasure-chest` command namespace is removed. The generic
`strategist runbook select` command is retained only as a JEWELER consumer and
does not import the Treasure Chest implementation or inspect local sidecars.

The root implementation and its unit tests remain in place only as the
temporary Weapon backend. The official Skill-for-Hire Weapon and shadow
comparison are Sprint 2 work; deleting the root implementation is Sprint 3
work.

## Evidence and invariants

- `external-skills-source/` contains `treasure-chest-tmp` as the only active
  Treasure package; the future external package is parked outside that scan
  root.
- The generated catalog, versioned mirror, and lock are produced by
  `strategist plugins prepare-embedded`.
- JEWELER validates the exact package, version, digest, role, slot, host API,
  connector, entrypoint, and explicit executable before invocation.
- Every provider response carries the temporary identity, schema version,
  freshness, limitations, and `FallbackState: none`.
- Provider readiness is probed before query or maintenance. Missing, malformed,
  unhealthy, or unavailable delegates remain explicit failures.
- Only `internal/weapon/treasuretmp` imports the root implementation in production
  code. Mission, pipeline, mechanism, and evaluation consumers use their own
  boundaries or JEWELER.

## Consequences

This sprint intentionally does not provide backward compatibility for the old
namespace or a second accepted provider identity. Migration comparison with
Skill-for-Hire and removal of the temporary backend require separate approved
sprints.
