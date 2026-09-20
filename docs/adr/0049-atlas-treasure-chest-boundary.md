# ADR-0049 — Atlas/Treasure Chest Staged Capability Boundary

**Status:** Accepted
**Date:** 2026-09-19
**Related:** ADR-0032, ADR-0040, ADR-0048

## Context

ADR-0032 assigns Treasure Chest domain ownership to an independently versioned
skill while retaining the host CLI namespace, routing, permissions, telemetry,
compatibility and diagnostics in Strategist. ADR-0040 introduces an in-repo
staging seam through Jewelcrafter. Direct command and evaluation importers
exist today, and their former allowlist did not distinguish a reviewed staging
exception from a working delegate.

## Decision

The Atlas/Treasure Chest boundary is a conformance projection, not a binding
authority. It evaluates one ordered stage at a time:

1. `in_repo_staging` — Strategist owns host policy; the root-level Treasure
   Chest implementation owns domain behavior, storage, indexing, curation and
   migration.
2. `external_adapter` — the same ownership split applies after an explicitly
   compatible external adapter is admitted. Declaring this stage never makes it
   active.

For either stage, the projection retains independent package, pinned identity,
provenance, API, delegate, probe and health evidence. A missing delegate is
`degraded` with `delegate_unwired`; missing probe evidence is `unverified`;
unsupported, blocked or unauthorized evidence is `blocked`. Only all certified
evidence makes an invocation path `ready`.

The host owns routing, permissions, envelopes, telemetry, the compatibility
CLI namespace and readiness diagnostics. Treasure Chest owns its domain and
data lifecycle. Equal or absent host/skill owners are an
`ownership_conflict` and fail closed.

External direct importers are classified deterministically as:

- `mediated` — a reviewed Jewelcrafter delegate path;
- `reviewed_exception` — an incremental staging exception with importer,
  owner, reason, scope and review deadline; or
- `violation` — an unreviewed, incomplete or expired direct dependency.

The current `cmd/strategist` and `internal/eval` imports are reviewed staging
exceptions through 2026-11-30. They are tooling-only and do not prove a
delegate exists. No other direct importer is permitted.

Migration records require distinct source and target identities, command
parity, API compatibility, verified provenance, completed migration evidence,
and a rollback reference. Incomplete evidence blocks the transition and
preserves legacy data.

Custom bindings remain owned by `.strategist/plugins.lock`. Ranked
certification remains owned by catalog/build/probe evidence. Caches and
indexes remain reconstructible projections; none may select a binding, repair
a lock, or promote itself to authority.

## Delegate lifecycle

The Jewelcrafter delegate (`treasure_chest_delegate` in its skill contract) moves
through the states below. The state is derived from the evidence vector; nothing
stores it as an authority, and no state is reached by declaring metadata.

| State | Evidence | Boundary result | Invocation |
| --- | --- | --- | --- |
| `unwired` | delegate evidence `unknown` (field null or package not imported by Jewelcrafter) | `degraded` / `delegate_unwired` | no |
| `wired` | delegate `certified`, probe not yet `certified` | `unverified` / `probe_unverified` (or `degraded` / `probe_<state>`) | no |
| `probed` | delegate and probe `certified`, health not yet `certified` | `unverified` / `health_unverified` (or `degraded` / `health_<state>`) | no |
| `healthy` | package, identity, provenance, API, delegate, probe and health all `certified` | `ready` / `activation_evidence_verified` | yes |
| `degraded` | any dimension `failed`, `stale`, `timeout`, `malformed`, `unavailable` or `teardown_failed` after having been better | `degraded` / `<dimension>_<state>` | no |
| `blocked` | any dimension `blocked`, `unsupported` or `unauthorized` | `blocked` / `<dimension>_<state>` | no |

Transitions are evidence-driven and reversible: losing certified evidence
returns the boundary to `degraded`, `unverified` or `blocked` without touching
legacy data (`LegacyDataPreserved` is always true). Today the in-repo stage is
`unwired`: the delegate path names a real package but Jewelcrafter does not yet
import and call it. The `treasure_chest_delegate_unwired` failure mode in the
Jewelcrafter contract stays accurate until that changes.

## Consequences

The import-boundary test queries direct Go imports (`.Imports`), not transitive
dependencies. It reports an expired or unknown direct import as a deterministic
failure. This ADR neither wires the Jewelcrafter implementation nor moves a
caller; such work must remove an exception only after mediated-path evidence
passes.

No new pipeline slot, binding file, fallback selection, mutable-artifact
refresh, external repository, OCI artifact, or data migration is authorized by
this decision.
