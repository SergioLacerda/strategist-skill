# JEWELER — Treasure Chest support boundary

JEWELER is the canonical non-pipeline support Role for Treasure Chest. It is
not Ranger, Archivist, or Sniper and never becomes a fourth mission phase.

The external Treasure Chest provider owns domain data, indexing, curation,
and provider-side mutations. Strategist owns routing, authorization,
provenance envelopes, readiness diagnostics, and unavailable-state behavior.

Ranger and Archivist may use only the bounded read-only query boundary. A
consumer request must identify the requester, mission, correlation key,
scope, result limit, and freshness policy. Every response carries provider
identity, provenance, captured-at, freshness, and readiness. A mutation in a
consumer envelope is rejected before the provider is called.

Indexing, curation, migration, reconciliation, probe, and health operations
are maintenance operations. They require an explicit JEWELER maintenance
authorization and an attribution record; they are never exposed through the
consumer query interface.

JEWELCRAFTER may appear only as an observable, time-bounded migration alias
resolving to JEWELER. There is no silent local, native, or mock fallback when
the external provider is unavailable or not ready.
