# Runbook: CARTOGRAPHER and ATLAS Structure Recovery

## Purpose

Diagnose and recover the CARTOGRAPHER support boundary when ATLAS is missing,
unverified, degraded, stale, corrupt, incompatible, or unavailable. Preserve
the canonical Strategist documentation and the last-known-good derived
structure projection throughout the procedure.

This runbook covers the external `skill-for-hire` ATLAS Weapon and the explicit
`atlas-mock` mode used by hermetic tests or development profiles. It does not
authorize code changes, binding rewrites, provider fallback, or deletion of
canonical data.

## Trigger

Apply this runbook when any of these conditions is observed:

- CARTOGRAPHER's `atlas` support binding is missing, duplicated, or owned by a
  Role other than CARTOGRAPHER;
- ATLAS reports `unverified`, `degraded`, `blocked`, `unavailable`,
  `probe_unverified`, `health_unverified`, or a similar readiness diagnostic;
- the structure projection under `.strategist/.data-structure/` is stale,
  corrupt, missing, or inconsistent with a canonical source fingerprint;
- Ranger or Archivist receives structure evidence without source identity,
  digest, provenance, captured-at time, freshness, provider mode, or readiness;
- a production profile appears to switch from ATLAS to `atlas-mock`,
  JEWELCRAFTER, or native structure generation without explicit selection;
- an ATLAS package is a contract-only stub, uses a mutable reference, or lacks
  the evidence needed for safe activation.

## Preconditions

- The mission's Approval Gate and any separate implementation authorization are
  known; this runbook never substitutes for either.
- The canonical source checkout and `.strategist/active.yaml` are available for
  read-only inspection.
- The active provider mode is recorded as `external` or explicitly selected
  `mock`; do not infer it from a cache, generated mirror, or catalog entry.
- The current support owner is verified as CARTOGRAPHER, and JEWELCRAFTER's
  `treasure` binding remains a separate authority.
- If migration or import is contemplated, a pinned source identity, digest,
  provenance record, API contract, and rollback reference exist before any data
  is touched.

## Resolution Steps

### 1. Capture the observed boundary state

Record the mission/correlation identifier, requester Role, provider mode,
binding generation, ATLAS package identity and digest, projection path, source
fingerprints, freshness/TTL, and the exact diagnostic. Preserve the current
`.strategist/` state before attempting recovery.

Run the read-only readiness check:

```bash
strategist check --json
```

Do not interpret `ready` static metadata as proof of a live ATLAS invocation.
The provider still needs compatible API, delegate, probe, and health evidence.

### 2. Confirm ownership and binding uniqueness

Verify that:

- CARTOGRAPHER is the only Role bound to the `atlas` support capability;
- no `discovery`, `refinement`, or `execution` slot points at ATLAS as an
  undeclared fourth-phase substitute;
- JEWELCRAFTER remains the only owner of the `treasure`/BAU DO TESOURO
  capability;
- Ranger and Archivist use the CARTOGRAPHER query boundary rather than calling
  an external provider or reading provider internals directly.

Stop with an ownership or binding diagnostic if any condition fails. Do not
repair a lock, provider catalog, or binding in place as part of this runbook.

### 3. Classify external ATLAS readiness

Collect each evidence dimension independently:

1. package validity;
2. immutable identity and expected digest;
3. trusted provenance/source;
4. API compatibility;
5. CARTOGRAPHER delegate wiring;
6. successful entrypoint probe;
7. health.

Apply the fail-closed outcomes:

- missing delegate → `degraded` / `delegate_unwired`;
- missing probe → `unverified` / `probe_unverified`;
- missing health → `unverified` / `health_unverified`;
- unsupported, blocked, or unauthorized evidence → `blocked`;
- any stale, malformed, unavailable, or failed evidence after a previously
  better state → `degraded` with the affected dimension;
- all dimensions certified → `ready` / `activation_evidence_verified`.

Catalog, lock, cache, generated mirror, or contract-stub presence alone never
permits invocation.

### 4. Check projection freshness and provenance

Compare `.strategist/.data-structure/` metadata with the canonical source
fingerprints and the projection TTL. Classify the result as fresh, stale,
unknown, missing, or corrupt.

- Fresh and complete: return structure evidence with its source digest and
  captured-at time.
- Stale or unknown: return an explicit stale/unknown response; do not present
  it as current evidence.
- Missing or corrupt: preserve the directory for diagnosis, quarantine only
  disposable derived data under an approved temporary path, and rebuild only
  through the ATLAS provider contract after readiness is restored.
- Source digest mismatch: stop and review provenance or package migration; do
  not rewrite the source digest to make the projection pass.

Canonical code, ADRs, runbooks, contracts, locks, Treasure Chest data, and
provider packages are never repaired by editing or deleting the projection.

### 5. Handle Mock explicitly

If a hermetic test or development profile explicitly selects `atlas-mock`,
verify that its response envelope matches the CARTOGRAPHER contract and that
the response identifies synthetic provenance and `provider_mode: mock`.

If production selected ATLAS and ATLAS is unavailable, return an unavailable or
degraded result. Do not select Mock, JEWELCRAFTER, or a native structure
generator automatically.

### 6. Recover or roll back

Proceed only after the failed dimension has a reviewed repair and the required
evidence can be collected again. For a failed activation or migration:

1. disable or quarantine the failed ATLAS candidate;
2. preserve the last-known-good support binding, projection, lock, and source
   data;
3. retain package, adapter, projection, and source digests for comparison;
4. restore only the previously recorded stage or last-known-good identity;
5. rerun readiness, probe, health, and bounded query checks;
6. leave the boundary marked unavailable/degraded until all evidence passes.

Do not accept mutable tags, silently refresh a remote reference, rewrite
`plugins.lock`, or delete the internal JEWELCRAFTER skill as a recovery step.

## Verification Checklist

- [ ] CARTOGRAPHER remains the sole owner of the `atlas` support binding.
- [ ] JEWELCRAFTER remains the sole owner of `treasure`/BAU DO TESOURO.
- [ ] The three mission slots and one-pipeline sequence are unchanged.
- [ ] Package, identity, provenance, API, delegate, probe, and health evidence
      are recorded separately.
- [ ] The projection has source fingerprints, freshness/TTL, provenance, and a
      captured-at time.
- [ ] Stale, missing, corrupt, or unavailable data is reported truthfully.
- [ ] Production ATLAS failure did not activate Mock or another Role.
- [ ] Ranger and Archivist receive only bounded, source-backed responses.
- [ ] Canonical sources, locks, provider data, and last-known-good projection
      remain preserved.
- [ ] Any migration has a pinned identity, API compatibility evidence, and a
      rollback reference.

## Stop Conditions

Stop and escalate when:

- the ATLAS source is mutable, unpinned, unverifiable, or contract-only;
- package, identity, provenance, API, delegate, probe, or health evidence is
  missing or contradictory;
- ownership is duplicated or the `treasure` boundary is involved;
- the projection cannot be tied to canonical source fingerprints;
- recovery would require mutating code, contracts, locks, provider catalogs,
  canonical documents, Treasure Chest data, or user data;
- a production fallback to Mock, JEWELCRAFTER, or native structure generation
  is proposed;
- Approval Gate, trust policy, provider admission, or rollback authority is
  unavailable.

## References

- [ADR-0058 — CARTOGRAPHER and ATLAS Structure Boundary](../adr/0058-cartographer-atlas-structure-boundary.md)
- [ADR-0049 — Atlas/Treasure Chest Staged Capability Boundary](../adr/0049-atlas-treasure-chest-boundary.md)
- [ADR-0051 — Remote Provider Acquisition and Trust Policy](../adr/0051-remote-provider-acquisition-and-trust-policy.md)
- [ADR-0056 — JEWELCRAFTER Public Support Role and BAU DO TESOURO Binding](../adr/0056-jewelcrafter-public-role-and-bau-tesouro-binding.md)
- [Atlas/Treasure Chest Boundary Evidence](atlas-treasure-chest-boundary.md)
- [Embedded Weapon Ingestion, Migration, and Rollback](embedded-weapon-ingestion-migration-rollback.md)

