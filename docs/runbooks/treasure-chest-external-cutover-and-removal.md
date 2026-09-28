# Runbook: Treasure Chest External Cutover and Removal

## Purpose

Diagnose and execute the controlled cutover from the in-repository
`treasure-chest/` package to the external `skill-for-hire` Treasure Chest
Weapon. Preserve legacy data, provenance, last-known-good state, and rollback
options while preventing two production owners or a silent local fallback.

This runbook covers the external provider boundary, JEWELCRAFTER-mediated
requests from Ranger and Archivist, the compatibility CLI namespace, migration
manifest, contract mock, partial-write recovery, and final removal gates. It
does not authorize code deletion or runtime changes by itself.

## Trigger

Apply this runbook when:

- `skill-for-hire` Treasure Chest admission, activation, or cutover is
  requested;
- the external package is missing, contract-only, unpinned, incompatible,
  unverifiable, unavailable, or fails its delegate, probe, or health check;
- the local package and external Weapon appear active at the same time;
- Ranger or Archivist cannot retrieve bounded offline data through
  JEWELCRAFTER, or a response lacks provider identity, contract version,
  provenance, or readiness state;
- migration reports duplicates, a partial write, index inconsistency, missing
  provenance, or a mismatch between source and destination data;
- the `strategist treasure-chest ...` namespace returns an unexpected local
  result, silently selects a fallback, or needs deprecation/removal;
- a removal gate is proposed before consumer, data, parity, generated-source,
  or repository-reference evidence is complete.

## Preconditions

- The mission Approval Gate and any separate implementation authorization are
  known. This runbook is not an implementation authorization.
- The external provider identity, immutable reference, provenance, expected
  contract/API version, delegate, probe, health signal, and rollback reference
  are available or the procedure is explicitly in a blocked/readiness-
  diagnosis mode.
- The current host configuration, source checkout, `.strategist/` runtime, and
  legacy Treasure Chest data are available for read-only inspection.
- JEWELCRAFTER is verified as the mediated support boundary for Ranger and
  Archivist. No direct provider call or second local owner is introduced.
- Any migration manifest is versioned, idempotent, provenance-preserving, and
  has a destination and rollback reference before data is touched.

## Resolution Steps

### 1. Capture the boundary state

Record the mission and correlation identifiers, requester Role, provider mode,
provider package identity and digest, contract version, delegate, probe and
health results, active compatibility namespace, configured data roots, current
manifest, source fingerprints, and exact diagnostic. Preserve the current
`.strategist/` state before recovery.

Run the read-only host check:

```bash
strategist check --json
```

Treat static readiness as a metadata observation only. It does not prove that
the external provider was invoked or that its runtime is healthy.

### 2. Confirm ownership and consumers

Verify that:

- the external `skill-for-hire` Weapon is the only proposed production owner
  of Treasure Chest domain, storage, indexing, curation, and migration;
- Ranger and Archivist request offline data through the JEWELCRAFTER boundary;
- the CLI namespace is either explicitly delegated or reports a clear
  unavailable/migration state;
- evaluation, harvest, scope-filter, coverage, quality-budget, generated
  default, and architecture-test references are inventoried;
- no code path silently imports or activates the local package after the
  external cutover.

Stop with an ownership diagnostic if two providers or two bindings are active.
Do not repair locks, catalogs, bindings, or source code in place from this
runbook.

### 3. Classify external readiness

Collect each readiness dimension independently:

1. package validity;
2. immutable provider identity and expected digest;
3. trusted provenance/source;
4. API and response compatibility;
5. host delegate wiring;
6. successful entrypoint probe;
7. health.

Apply these fail-closed outcomes:

- missing delegate → `degraded` / `delegate_unwired`;
- missing probe → `unverified` / `probe_unverified`;
- missing health → `unverified` / `health_unverified`;
- unsupported, unauthorized, or contradictory evidence → `blocked`;
- stale, malformed, unavailable, or failed evidence after a better previous
  state → `degraded` with the affected dimension;
- all dimensions certified → `ready` / `activation_evidence_verified`.

The current contract-only `skill-for-hire` material is not runtime readiness.
Do not delete the local package or migrate data while readiness is merely
static, mocked, or inferred.

### 4. Inventory data and prepare the migration manifest

Enumerate, without deleting or rewriting source data:

- configured Treasure Chest roots and source paths;
- jewels, potions, chest manifests, compiled indexes, and related `.strategist`
  state;
- source identities, digests/checksums where available, provenance, and current
  ownership;
- items already migrated, skipped, conflicting, malformed, or requiring manual
  review.

The manifest must identify the source and destination for every migrated item,
retain provenance, and support a dry run. Repeating the same manifest against
the same source must be idempotent: no duplicate records, no ownership loss,
and a deterministic already-migrated result.

If the provider is unavailable, leave legacy data untouched and report the
blocked migration with the manifest and recovery path.

### 5. Validate the mediated boundary and explicit Mock

For Ranger and Archivist, verify bounded request scope, result limits, provider
identity, contract version, provenance, captured-at time, and readiness state
in every response. A provider error must remain an explicit error; it must not
be converted into a local answer.

For hermetic tests or an explicitly selected development profile, a contract
mock may answer with `provider_mode: mock` and synthetic provenance. Confirm
that its request/response/error semantics match the external contract. If a
production profile selects the mock, stop with a configuration error.

### 6. Perform the staged namespace cutover

Only after readiness, consumer migration, manifest validation, and parity
evidence pass:

1. switch the compatibility CLI namespace to explicit external delegation;
2. run success, unavailable-provider, incompatible-provider, malformed-response,
   and duplicate-owner checks;
3. verify Ranger and Archivist query responses and provenance;
4. verify generated/default source parity and update the repository reference
   audit;
5. retain the previous binding, manifest, provider identity, and rollback
   reference until the breaking-release boundary is complete.

If any check fails, keep the namespace in an actionable migration or
unavailable state. Never re-enable the removed package silently.

### 7. Recover a partial write or inconsistent index

When a migration or chest operation leaves active data, governed data, a
manifest, an index, or jewel/potion state inconsistent:

1. stop further writes and capture the operation/correlation identifier;
2. preserve the source and destination artifacts, manifest, and digests;
3. classify each item as committed, pending, duplicated, conflicting, or
   unreadable;
4. reconcile only through the versioned external migration contract;
5. use the manifest to replay missing idempotent operations or quarantine
   conflicts for review;
6. rebuild or verify the external index without deleting the legacy source;
7. rerun provenance, parity, and bounded-query checks before resuming.

Do not hand-edit an index to conceal a mismatch, delete legacy data, or retry
blindly after an unknown write outcome.

### 8. Roll back safely

Before final package removal, rollback means restoring the recorded last-known-
good delegation/configuration and leaving legacy data untouched. Preserve the
external candidate, manifest, source/destination digests, and failed evidence
for comparison, then rerun readiness, probe, health, and bounded queries.

After the local package has been removed, rollback means restoring a prior
released host artifact or an explicitly versioned compatibility adapter. It
does not recreate an ungoverned second implementation or silently switch to a
Mock/JEWELCRAFTER/native fallback.

### 9. Evaluate final removal gates

The local `treasure-chest/` package may be removed only when all of these are
evidenced:

- external provider readiness and provenance are complete;
- Ranger, Archivist, CLI, evaluation, and harvest consumers use the mediated
  boundary;
- data migration manifest is validated, idempotent, and rollback-ready;
- command and semantic parity checks pass;
- embedded defaults and generated `.strategist` mirrors are synchronized;
- coverage, quality-budget, architecture, lock/catalog, and documentation
  references are reconciled;
- repository-wide import/reference audit has no unexplained active dependency;
- the breaking release/deprecation boundary is explicit and accepted.

If one gate is missing, keep the package and report the exact blocker.

## Verification Checklist

- [ ] Provider identity, immutable reference, provenance, API version, delegate,
      probe, and health are recorded separately.
- [ ] `skill-for-hire` runtime readiness is distinguished from contract-only or
      mock evidence.
- [ ] JEWELCRAFTER mediates Ranger and Archivist offline queries.
- [ ] The compatibility CLI delegates explicitly or reports an actionable
      readiness/migration error.
- [ ] No silent fallback activates the local package, Mock, or another Role.
- [ ] Every legacy data class has a source, destination, provenance, status,
      and rollback reference in the migration manifest.
- [ ] Repeated migration is idempotent and does not duplicate records.
- [ ] Partial writes stop safely and preserve all source/destination evidence.
- [ ] Generated source/runtime parity, coverage, quality, architecture, and
      repository-reference checks pass.
- [ ] The final removal and breaking-release gates are explicitly evidenced.

## Stop Conditions

Stop and escalate when:

- the external provider is mutable, unpinned, contract-only, unverifiable,
  incompatible, unauthorized, or missing probe/health evidence;
- two production owners, bindings, or fallback paths are active;
- a Ranger/Archivist response lacks bounded scope, provenance, identity,
  contract version, freshness, or readiness diagnostics;
- migration would delete, overwrite, or silently reassign legacy data;
- a partial write has an unknown outcome or an index cannot be tied to its
  manifest and source fingerprints;
- recovery requires hand-editing canonical source, locks, catalogs, provider
  data, or unrelated user changes;
- the requested operation is package deletion before all final gates pass;
- Approval Gate, provider trust, migration authority, or rollback authority is
  unavailable.

## References

- [ADR-0059 — Treasure Chest Core Removal and External Cutover](../adr/0059-treasure-chest-core-removal-and-external-cutover.md)
- [ADR-0032 — External Skill CLI Embedding and Treasure Chest Ownership](../adr/0032-external-skill-cli-embedding-and-treasure-chest-ownership.md)
- [ADR-0040 — Treasure Chest In-Repo Isolation Staging](../adr/0040-treasure-chest-in-repo-isolation-staging.md)
- [ADR-0049 — Atlas/Treasure Chest Boundary](../adr/0049-atlas-treasure-chest-boundary.md)
- [ADR-0056 — JEWELCRAFTER Public Role and BAU DO TESOURO Binding](../adr/0056-jewelcrafter-public-role-and-bau-tesouro-binding.md)
- [Embedded Weapon Ingestion, Migration, and Rollback](embedded-weapon-ingestion-migration-rollback.md)
- [Treasure Chest Partial Write Recovery](treasure-chest-partial-write.md)
- [Jewelcrafter Offline Curation Recovery](jewelcrafter-offline-curation-recovery.md)
