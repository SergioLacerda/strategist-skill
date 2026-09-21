# Runbook: Atlas/Treasure Chest Boundary Evidence

## Purpose

Audit the staged Treasure Chest boundary without treating catalog metadata,
an import exception, or a static package as live delegate evidence.

## Verify the current staging boundary

Run the focused hermetic checks:

```bash
GOCACHE=/tmp/strategist-atlas-gocache go test ./internal/conformance
GOCACHE=/tmp/strategist-atlas-gocache go test -tags=integration ./treasure-chest
```

Expected current state: the package and reviewed direct-import inventory pass;
Jewelcrafter remains `degraded` with `delegate_unwired` until a real delegate
and successful entrypoint probe exist. Do not report that result as external
adapter readiness.

## Review direct-import exceptions

For every package outside `treasure-chest/` importing a Treasure Chest package,
record one of these dispositions:

| Disposition | Required evidence | Meaning |
| --- | --- | --- |
| `mediated` | reviewed delegate path | Delegate is the approved boundary, not an external readiness claim. |
| `reviewed_exception` | importer, owner, reason, scope, review deadline | Temporary staging tooling dependency. |
| `violation` | none | Stop; an unreviewed, incomplete, or expired dependency is present. |

Before the 2026-11-30 review deadline, reassess `cmd/strategist` (`eval
harvest`) and `internal/eval` (scope/policy validation). Remove either
exception only after the caller has a mediated-path test. Do not extend an
exception merely because the static architecture test passes.

## Admit a delegate or external adapter

Collect separate certified evidence for package validity, immutable identity,
provenance, API compatibility, delegate wiring, entrypoint probe and health.
Apply these outcomes exactly:

- missing delegate: `degraded` / `delegate_unwired`;
- absent probe: `unverified` / `probe_unverified`;
- unsupported, blocked, or unauthorized evidence: `blocked`;
- all evidence certified: `ready` / `activation_evidence_verified`.

Static catalog, lock, manifest or cache evidence cannot substitute for probe or
health evidence. Never repair `.strategist/plugins.lock`, choose a fallback,
or promote generated/cache data while resolving a failure.

## Read the delegate lifecycle

The boundary result maps to a delegate state (`unwired` → `wired` → `probed` →
`healthy`, with `degraded` and `blocked` reachable from any state); the full
table is in [ADR-0049](../adr/0049-atlas-treasure-chest-boundary.md#delegate-lifecycle).
Only `healthy` (`ready` / `activation_evidence_verified`) permits invocation.
Losing certified evidence moves the state back without touching legacy data.

## Migration and rollback

Before any transition, record source and target pinned identities, command
parity, API compatibility, verified provenance, completed migration evidence,
and a rollback reference. If any field is absent, stop with
`migration_evidence_incomplete`; legacy data must remain untouched.

For a rollback, disable activation, retain the legacy data and pinned record,
then restore only the recorded prior stage. External repository, OCI,
remote-cache and user-data migration operations require their own approved
change.

## References

- [ADR-0049](../adr/0049-atlas-treasure-chest-boundary.md)
- [ADR-0032](../adr/0032-external-skill-cli-embedding-and-treasure-chest-ownership.md)
- [ADR-0040](../adr/0040-treasure-chest-in-repo-isolation-staging.md)
- [Embedded skill ingestion, migration, and rollback](embedded-skill-ingestion-migration-rollback.md)
