# ADR-0063 — Provider-agnostic governance compatibility boundary

**Status:** Accepted
**Date:** 2026-10-02
- Source package: `.analysis/refined/20261002-governance-compatibility-review-r2/`
- Scope: Contract record; implementation is tracked by ADR-0062

## Context

Strategist must be useful in environments that provide no governance and must
also collaborate with governance that the host explicitly provisions. The core
must therefore be agnostic to governance models. The historical SDD/sdd-ask
names and `.sdd` runtime are legacy implementation vocabulary, not a
governance contract.

The current environment uses Providence and `.providence`, but Providence is
an environment-owned governance provider. Its registry, mandates, and managed
files are not Strategist-owned configuration and are outside this ADR's
materialization scope.

## Decision

Strategist adopts a provider-agnostic governance boundary with four explicit
modes:

1. **Standalone:** when the host supplies no governance source or injection,
   Strategist operates normally and does not search for `.sdd`, `.providence`,
   or any other model-specific directory.
2. **Provisioned:** when the host explicitly supplies a source or injection,
   Strategist consumes it only after preserving source identity, non-empty
   fingerprint, required mandates, declared scope, correlation, and validation
   outcome.
3. **Invalid:** malformed, unavailable, uncorrelated, or otherwise invalid
   provisioned governance produces a deterministic observable failure.
4. **Insufficient authority:** a source that cannot establish authority for
   the requested scope is denied rather than treated as authoritative.

“Acatar” means treating validated provisioned governance as authoritative for
its declared scope and correlation. It does not mean inferring a governance
model, probing conventional directories, or guessing the meaning of missing
mandates.

The generic source, snapshot, synchronization, bridge, injection, compliance,
and telemetry contracts remain model-agnostic. Providence may be implemented
as the first explicit reference adapter, but Providence names must not leak
into generic interfaces or standalone behavior. The Providence adapter may
map `metadata.json`'s `fingerprints.combined` and the required `MANDATE`
entries from `source/governance-core.json` into the generic snapshot. The
legacy fingerprint fallback is not part of that adapter.

Invalid or insufficiently authoritative provisioned governance MUST fail
closed. The operation MUST NOT silently fall back to standalone behavior,
perform a partial governance write, or reinterpret one provider as another.

## Compatibility and breaking boundary

Adoption of the generic boundary is a proposed breaking boundary for the
retirement of SDD-specific flags, aliases, defaults, dual-read behavior, and
fallback. A retired SDD input must be rejected rather than interpreted as
Providence or as generic governance. The exact breaking version and
CHANGELOG policy must be recorded before implementation begins; this ADR does
not invent a version number.

No autodetection, default governance directory, compatibility alias, implicit
dual-read, silent conversion, or second adapter is authorized by this ADR.
Rollback is a release revert, not an implicit runtime fallback path.

## Providence boundary and external coordination

Providence is an optional adapter selected explicitly by the host. Strategist
must remain usable without it or with another provider. The current
`.providence/**` registry discrepancy is an external owner-coordinated side
quest; this ADR does not edit `.providence/**`, its mandates, or its registry.

## Consequences

- Strategist remains standalone when no governance is provisioned.
- A host can provide governance without coupling the core to a named model.
- Authority, scope, identity, correlation, fingerprint, and validation become
  observable requirements at the provider boundary.
- Invalid provisioned governance has a deterministic fail-closed outcome.
- Providence can be supported without making its runtime layout normative.
- Existing SDD surfaces require an explicit breaking release decision and
  implementation plan before retirement.

## Implementation boundary

This ADR records the contract. The related implementation handoff is tracked
by ADR-0062, which records the breaking boundary, implementation decisions,
generated parity, tests, and validation evidence.
