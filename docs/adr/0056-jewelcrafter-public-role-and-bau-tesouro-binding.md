# ADR-0056 — JEWELCRAFTER Public Support Role and BAU DO TESOURO Binding

**Status:** Accepted
**Date:** 2026-09-27
**Related:** ADR-0032, ADR-0040, ADR-0049, ADR-0053

## Context

JEWELCRAFTER began as an internal offline-curation skill. The intended
integration is now stronger: it is the only Strategist Role that handles the
BAU DO TESOURO capability. Ranger and Archivist need to ask JEWELCRAFTER for
bounded offline data rather than invoking a provider or reading provider
internals directly.

The production BAU DO TESOURO capability will be supplied by an external
`skill-for-hire` Weapon. The current embedded internal Jewelcrafter skill is a
transition source and is scheduled for removal. A deterministic Mock is needed
for hermetic integration tests and explicitly selected development profiles
when the external package is unavailable.

The current runtime does not support this with a role file alone. Public role
activation currently rejects `jewelcrafter`, Role/provider contracts require a
valid slot for pluggable roles, and the supported slot vocabulary is limited
to `discovery`, `refinement`, and `execution`.

## Decision

JEWELCRAFTER is approved as a public, non-pipeline support Role. It uses a
dedicated `treasure` support slot, which is distinct from the three mission
pipeline slots and does not create a fourth mission phase.

The Role owns the single provider binding and broker boundary for BAU DO
TESOURO:

```text
Ranger ───────┐
              ├─ offline query ─> JEWELCRAFTER Role ─> BAU DO TESOURO Weapon
Archivist ────┘                         │
                                        └─ .strategist/.data-offline/
```

The binding uses the existing provider catalog, trust/grant, plugin lock,
installed-instance, provenance, readiness, and last-known-good lifecycle. It
does not introduce a second Role/provider registry or lock store.

The production provider is the external `skill-for-hire` BAU DO TESOURO
Weapon. Its package identity, immutable version or digest, provenance, API
compatibility, delegate wiring, probe, and health are independent readiness
evidence. Catalog, lock, cache, or generated mirror presence alone never makes
the provider invocable.

The `bau-do-tesouro-mock` provider implements the same bounded query and
response envelope for hermetic tests and explicitly selected development
profiles. Mock responses and telemetry must identify themselves as synthetic.
Mock selection is configuration, not automatic fallback: if a production
profile selects the external provider and it is unavailable, JEWELCRAFTER
returns an unavailable or degraded result rather than silently selecting Mock.

## Query boundary

Ranger and Archivist may query JEWELCRAFTER only through a read-only offline
interface. A request contains at least:

- requester Role (`ranger` or `archivist`);
- mission and correlation identifiers;
- bounded query scope and result limit;
- freshness/TTL policy.

A response contains bounded results, source identity and digest, provenance,
captured-at time, expiry/freshness state, provider mode (`external` or
`mock`), and readiness diagnostics. A missing, expired, corrupt, or unavailable
projection produces a truthful response and never triggers a hidden live fetch,
provider bypass, synthesized production result, or native-role fallback.

## Migration and removal

The embedded internal Jewelcrafter skill and generated mirror remain
transitional until all of the following are verified:

1. the public Role and `treasure` binding validate and persist through the
   existing lifecycle;
2. external and Mock providers conform to the same query contract;
3. Ranger and Archivist use only the JEWELCRAFTER query boundary;
4. offline projection, parity, readiness, rollback, and hermetic tests pass;
5. no production path treats the internal skill as a second provider.

After that evidence exists, a separate controlled migration removes the
internal skill source and generated mirror together, updates conformance
fixtures and documentation, and preserves a rollback reference. This ADR does
not authorize that source removal by itself.

## Consequences

Positive consequences:

- BAU DO TESOURO has one explicit Role owner and one binding authority.
- Ranger and Archivist remain consumers of offline evidence instead of
  provider implementations.
- Tests can exercise the complete integration without a network or external
  repository.
- The external provider can be unavailable without corrupting pipeline state.

Costs and constraints:

- The host must extend its Role, support-slot, provider, lock, and validation
  contracts; this is implementation handoff work, not a YAML-only change.
- The `treasure` support slot must remain isolated from mission routing and
  required pipeline-slot checks.
- Mock evidence cannot certify the external provider.
- The external repository identity and final provider API remain to be fixed
  by the implementation handoff.

## Non-Goals

This decision does not select an external repository or release protocol,
implement the Role/provider/query code, remove the internal skill immediately,
add a fourth mission phase, or migrate Treasure Chest data.
