# ADR-0047 — Hardening Authorities and Fail-Closed Outcomes

**Status:** Accepted  
**Date:** 2026-09-17

## Context

The runtime has compiled-artifact manifests, stale checks, mission snapshots,
and telemetry evidence. These mechanisms answer different questions, but an
operator must not mistake a diagnostic mirror or incomplete evidence for an
authoritative success.

## Decision

Hardening uses the following ownership map:

| Fact | Authority | Mirror policy |
|---|---|---|
| Runtime configuration | `.strategist/active.yaml` | no fallback authority |
| Custom Role→Weapon binding | `.strategist/plugins.lock` | authoritative |
| Ranked Role→Weapon binding | build/catalog certification | runtime copy is diagnostic only |
| Mission transitions | `domain.MissionEngine` | telemetry cannot advance state |
| Context identity | `domain.ContextMaterializer` | provider content cannot redefine it |
| Observability | `telemetry.Event` | evidence only, never transition authority |

Hardening outcomes use the closed vocabulary `verified`, `unknown`,
`unavailable`, `degraded`, `failed`, `stale`, and `blocked`. Only `verified`
permits continuation. Missing, conflicting, corrupt, or divergent evidence is
reported explicitly and cannot silently fall back to another authority.

The existing compiled manifest remains the runtime integrity authority. Phase B
does not create a second manifest or durable event journal. Snapshot restoration
and replay evidence must still pass through `MissionEngine` transition rules.

## Consequences

- Existing manifest and stale checks can be composed into one result without
  invalidating legacy installs silently.
- Telemetry sequence gaps remain valuable evidence while remaining unable to
  mutate mission state.
- Future cache, cross-client conformance, or event-journal work must preserve
  this ownership map and propose a separate change when it changes behavior.
