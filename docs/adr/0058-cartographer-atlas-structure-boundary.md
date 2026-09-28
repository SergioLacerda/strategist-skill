# ADR-0058 — CARTOGRAPHER and ATLAS Structure Boundary

**Status:** Accepted  
**Date:** 2026-09-27  
**Mission:** `20260927-cartographer-role-refinement`  
**Related:** [ADR-0049](0049-atlas-treasure-chest-boundary.md), [ADR-0051](0051-remote-provider-acquisition-and-trust-policy.md), [ADR-0056](0056-jewelcrafter-public-role-and-bau-tesouro-binding.md)

## Context

Strategist needs a single owner for structural information about its own
business and architecture documentation: modules, dependencies, contracts,
entrypoints, relationships, relevant ADRs, and runbooks. The pending
CARTOGRAPHER proposal assigns that responsibility to an external ATLAS skill
from `skill-for-hire` and calls for a derived runtime structure such as
`.strategist/.data-structure/`.

The current runtime has only the `discovery`, `refinement`, and `execution`
mission slots and still treats CARTOGRAPHER as an inactive proposal. A role
definition alone cannot safely activate it. JEWELCRAFTER already owns a
different public support boundary: the `treasure` binding and BAU DO TESOURO's
dynamic offline curation, including Treasure Chest knowledge lifecycle,
runbooks, and ADR precedents.

The observed `skill-for-hire` material does not prove a ready ATLAS runtime.
External package identity, provenance, API compatibility, delegate wiring,
probe, and health therefore remain separate activation evidence.

## Decision

1. **CARTOGRAPHER is a public, non-pipeline support Role.** It works across
   discovery and refinement without creating a fourth mission phase.

2. **CARTOGRAPHER exclusively owns the `atlas` support binding.** The ATLAS
   Weapon supplied by `skill-for-hire` is admitted through this binding only.
   It is not bound through `discovery`, `refinement`, `execution`, or
   JEWELCRAFTER's `treasure` binding.

3. **Ranger and Archivist consume a bounded, read-only query boundary.** A
   request includes the requester Role, mission and correlation identifiers,
   bounded scope, result limit, and freshness/TTL policy. A response includes
   bounded source-backed results, source identity and digest, provenance,
   captured-at time, freshness state, provider mode, and readiness diagnostics.

4. **The structure map is a derived projection.** Runtime data belongs under
   `.strategist/.data-structure/` or a versioned equivalent. It is rebuildable,
   fingerprinted, and freshness-aware. Canonical code, documentation, ADRs,
   runbooks, contracts, locks, and provider data remain authoritative and are
   never replaced or mutated by the projection.

5. **ATLAS activation is fail-closed.** Package validity, immutable identity,
   provenance, API compatibility, delegate wiring, entrypoint probe, and health
   must all be certified before invocation. Static catalog, cache, lock,
   generated mirror, or contract-stub presence is insufficient. Loss of
   evidence preserves the last-known-good projection and reports an explicit
   unverified, degraded, or blocked state.

6. **`atlas-mock` is explicit and contract-compatible.** It may be selected by
   hermetic tests and explicit development profiles. Its responses identify
   synthetic provenance. A production profile selecting ATLAS never silently
   switches to Mock, JEWELCRAFTER, or native structure synthesis.

7. **Ownership remains distinct from JEWELCRAFTER.** CARTOGRAPHER may locate
   and cite ADRs or runbooks related to a mapped component, but it does not
   curate Treasure Chest content or generate those documents. JEWELCRAFTER
   remains the exclusive BAU DO TESOURO/Treasure Chest owner.

## Boundary

```text
canonical Strategist sources
          │ indexed by ATLAS
          ▼
.strategist/.data-structure/ ──> CARTOGRAPHER ──> bounded query response
                                      │                 │
                                      │                 ├─ Ranger
                                      │                 └─ Archivist
                                      │
JEWELCRAFTER ── `treasure` ──> BAU DO TESOURO / Treasure Chest
```

The host owns role routing, permissions, envelopes, telemetry, compatibility,
and readiness diagnostics. ATLAS owns structural indexing and its external
domain behavior. CARTOGRAPHER translates mission needs and normalizes the
provider response; it does not implement a parallel parser or index.

## Consequences

### Positive

- One explicit authority exists for Strategist's business and architecture map.
- Ranger and Archivist can obtain source-backed context without direct provider
  calls or dependency on ATLAS internals.
- The three-stage mission pipeline remains unchanged.
- Stale or unavailable external capabilities cannot fabricate current structure
  facts or silently change provider mode.
- JEWELCRAFTER and CARTOGRAPHER have distinct ownership, projections, and
  recovery paths.

### Costs and constraints

- The host must extend its public Role and support-binding contracts; this ADR
  does not implement that work.
- The final ATLAS API, immutable package identity, trust policy, and repository
  pin remain external inputs.
- A derived structure projection requires source fingerprints, TTL handling,
  rebuild, and rollback behavior.
- Mock conformance cannot certify external ATLAS readiness.

## Non-Goals

- Adding a fourth pipeline phase.
- Reusing or mutating the `treasure` support binding.
- Defining Gem, Scroll, Itinerary, or a new Strategist artifact family.
- Implementing an internal ADR parser, dependency analyzer, or parallel
  Treasure Chest index.
- Removing the internal JEWELCRAFTER skill or migrating Treasure Chest data.
- Automatically generating ADRs or runbooks from structure queries.

## Implementation Handoff

The following remain separate coding and integration work: public role and
support-binding validation, provider catalog/lock/readiness integration, the
versioned query envelope, `.data-structure` projection, Ranger/Archivist
consumer integration, ATLAS adapter, explicit Mock, and hermetic tests. Approval
of this ADR authorizes none of those mutations by itself.

