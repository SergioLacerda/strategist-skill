# Runbook: JEWELCRAFTER Offline Curation and BAU DO TESOURO Recovery

## Scope

JEWELCRAFTER is the public, non-pipeline support Role for the `treasure`
support slot. It is the only Role allowed to handle the BAU DO TESOURO Weapon.
Ranger and Archivist consume its offline query boundary; they do not invoke the
Weapon, inspect provider internals, or select a fallback.

The production provider is an external `skill-for-hire` Weapon. The
`bau-do-tesouro-mock` provider is allowed only when a test or development
profile explicitly selects it.

## Recovery states

| State | Meaning | Consumer behavior |
| --- | --- | --- |
| `ready` | External provider evidence is complete and the offline projection is usable | Return bounded offline data with provenance and freshness |
| `mock` | Explicit Mock profile is active | Return deterministic synthetic data marked `mock`; never treat it as external readiness |
| `unavailable` | Provider is absent or cannot be imported | Return unavailable; do not select Mock implicitly |
| `unverified` | Package, identity, provenance, API, delegate, probe, or health evidence is incomplete | Return the precise readiness reason; do not invoke |
| `degraded` | A previously usable provider or projection lost required health/freshness | Preserve last-known-good data when policy permits and report degradation |
| `expired` | The matching offline projection exceeded its TTL | Return expired; do not synthesize or perform a hidden live fetch |
| `corrupt` | Offline index or provider metadata cannot be parsed | Preserve the last-known-good copy when available and rebuild only from permitted sources |

## First response: identify the active authority

1. Record the mission, requester (`ranger` or `archivist`), correlation id,
   query scope, and active support-slot provider.
2. Confirm that the request went through JEWELCRAFTER. A direct Ranger or
   Archivist call to the external Weapon is a boundary violation.
3. Confirm the selected provider mode:
   - `external` is the production path;
   - `mock` is valid only when explicitly selected by the test/development
     profile.
4. Never repair the lock, catalog, cache, or provider selection by editing a
   generated artifact or by choosing a native-role fallback.

## External provider unavailable

When the external `skill-for-hire` package cannot be imported, installed, or
probed:

1. Keep the `treasure` binding unavailable or degraded.
2. Return the exact missing evidence dimension: package, immutable identity,
   provenance, API, delegate, probe, or health.
3. Do not activate `bau-do-tesouro-mock` automatically in a production
   profile.
4. If integration testing is the objective, select the Mock explicitly in an
   isolated test/development profile and mark all returned data synthetic.
5. Re-run the provider conformance and query checks before promoting the
   external provider to ready.

## Offline query recovery

For a Ranger or Archivist query:

1. Validate requester, mission/correlation id, bounded scope, result limit,
   and freshness policy.
2. Read only the JEWELCRAFTER offline projection. A valid response must carry
   source identity, digest, provenance, captured-at time, TTL/freshness,
   provider mode, and readiness state.
3. If the projection is absent or expired, return `unavailable` or `expired`.
   Do not call the external Weapon as a hidden side effect.
4. If the projection is corrupt, quarantine the invalid derived index, retain
   the last-known-good state when available, and rebuild from readable,
   policy-approved sources only.
5. Never mutate source ADRs, runbooks, generic documentation, Treasure Chest
   data, or Git state as part of recovery.

## Internal-skill migration

The embedded internal Jewelcrafter skill is transitional. Do not remove it
until the following checks pass in an isolated environment:

- public Role and `treasure` support binding validate;
- external and Mock providers satisfy the same query envelope;
- Ranger and Archivist have no direct provider path;
- offline digest/TTL/corrupt-index and last-known-good recovery pass;
- generated parity, rollback, and conformance checks pass;
- the internal skill is not selected as a second production provider.

Removal is a separate controlled migration. Preserve a rollback reference and
remove the source skill and generated mirror together; do not delete either
copy opportunistically during provider recovery.

## Stop conditions

Stop and escalate to the implementation owner when:

- the requested provider is not the configured JEWELCRAFTER binding;
- a Mock result lacks explicit synthetic provenance;
- any consumer attempts a direct Weapon/provider call;
- readiness is inferred from catalog, lock, mirror, or cache presence alone;
- an index rebuild would mutate source or Treasure Chest data;
- the external repository identity, immutable version/digest, or API contract
  is missing.

## References

- `docs/adr/0056-jewelcrafter-public-role-and-bau-tesouro-binding.md`
- `docs/adr/0049-atlas-treasure-chest-boundary.md`
- `docs/runbooks/treasure-chest-partial-write.md`
- `.strategist/contracts/machine/context-enrichment.yaml`
