# ADR-0062 — Provider-Agnostic Governance Compatibility Implementation

**Status:** Accepted for implementation
**Date:** 2026-10-02
**Related:** [ADR-0063](0063-governance-compatibility.md) (accepted governance contract record, renumbered to resolve the historical ADR-0055 collision)
**Implementation plan:** `docs/plans/2026-10-02-governance-compatibility-implementation.md`

## Release boundary

The repository currently uses a flat sequential release scheme and is at
`v1.0.27`. This breaking boundary is targeted for `v1.0.28`. The `[Unreleased]`
section of `CHANGELOG.md` records the removed SDD-specific surfaces and the
changed generic governance contract; the release maintainer remains responsible
for the final tag and release publication.

## Implementation decision

Implement the provider-agnostic contract recorded in ADR-0063. Strategist has
no implicit governance model or directory. No source or injection means
standalone operation. An explicitly selected source is normalized, correlated,
validated, and consumed only when it has sufficient authority; malformed,
uncorrelated, unavailable, or insufficient sources fail closed without fallback.

The generic source/snapshot, sync, bridge, injection, compliance, and telemetry
interfaces remain provider-neutral. Providence is the first explicit adapter.
It reads only `metadata.json`'s `fingerprints.combined` and required `MANDATE`
entries from `source/governance-core.json`, with deterministic mandate ordering
and no legacy fingerprint fallback. Providence names stay at the adapter
boundary and do not become core defaults.

The `--sdd` flag, SDD aliases, implicit `.sdd` defaults, dual-read, and silent
fallback are removed as breaking behavior. `.providence/**` remains owned by
the provisioned governance environment and is not modified by this change.
