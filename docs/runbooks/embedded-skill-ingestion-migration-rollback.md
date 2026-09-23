# Runbook: Embedded Skill Ingestion, Migration, and Rollback

## Purpose

Operate the build and release path for skills that are selected by the project,
embedded into Strategist, or migrated from a core feature to an external skill.
This runbook covers verification, failure handling, legacy-data preservation,
and rollback. It does not authorize source changes, lock updates, or deletion
of user data.

## Applies to

- `strategist plugins prepare-embedded`
- the repository target that prepares embedded skills before build
- release failures involving embedded skill artifacts or generated catalogs
- migration of legacy Treasure Chest data to the external skill
- rollback after failed verification, probe, activation, or migration

## Preconditions

- The change has an approved package/adapter contract and implementation task.
- The committed embedded-skill lock is available.
- The selected artifact source, digest, license, provenance and adapter
  revision are known.
- The external skill's migration/import contract exists before legacy data is
  touched.
- A backup or recoverable copy of workspace runtime data exists before import.
- No release build is allowed to refresh mutable tags implicitly.

## Readiness states

Classify the incident before changing anything:

1. `package_valid`
2. `artifact_digest_verified`
3. `license_policy_passed`
4. `adapter_api_compatible`
5. `embedded_in_build`
6. `runtime_catalog_visible`
7. `command_registered`
8. `permissions_granted`
9. `entrypoint_probed`
10. `invocation_supported`
11. `healthy`

Do not treat a later state as proven when only an earlier state passed.

## Standard ingestion procedure

1. Read the committed lock and identify the package ID, version, digest,
   adapter revision, API range, source class and expected generated outputs.
2. Resolve the artifact from the approved local/cache/published source.
3. Verify the artifact digest against the lock. Stop on mismatch; do not
   rewrite the lock to make the observed artifact pass.
4. Verify license, provenance, package schema, adapter schema, dependencies,
   command paths, entrypoints and compatibility.
5. Check requested permissions against the project policy and connector
   capabilities.
6. Stage the artifact and generate the embedded catalog/registration inputs.
7. Compare generated outputs with the committed state. Stop on drift unless a
   maintainer has explicitly started a lock/catalog update workflow.
8. Run the package, adapter, command-tree and offline reproducibility checks.
9. Build the binary and inspect the embedded skill inventory before release.
10. Record the exact package/adapter digests and readiness evidence in the
    build/release report.

## Failure handling

### Digest or provenance mismatch

- Stop ingestion immediately.
- Preserve the current lock and generated catalog.
- Do not accept a mutable tag, new digest, or unreviewed artifact as a repair.
- Escalate for an explicit package/lock review.

### Schema/API/command collision failure

- Do not register the skill.
- Report the conflicting field, API version, command path, or reserved flag.
- Preserve the last known-good embedded catalog.
- Return the package to its publisher/adapter maintainer for correction.

### Missing artifact or unavailable source

- Keep the previous release/build inputs intact.
- If the skill is optional, build without activating the new candidate and
  report the embedded skill as unavailable.
- If the release contract requires the skill, fail the release rather than
  shipping an incomplete inventory.

### Unsupported connector or invocation

- Record `invocation_unsupported` or the connector-specific reason.
- Do not label the skill healthy because it is embedded or command-registered.
- Preserve native Strategist behavior and any last-known-good binding.

## Legacy Treasure Chest migration

1. Inventory legacy paths and record their hashes before import.
2. Confirm that the external skill version and adapter explicitly support the
   source schema being migrated.
3. Run the external skill's dry-run/import preview.
4. Review mapping for chests, trust policy, jewels, potions, indexes, history,
   source references, lifecycle status and provenance.
5. Execute the import only through the approved migration command.
6. Verify idempotence by repeating the dry-run or checking stable IDs and
   source hashes.
7. Compare counts, statuses, source refs and selected sample content.
8. Preserve the legacy data until the migration acceptance decision is made.
9. Record the migration version, source hashes, destination, result and
   rollback point.

The Strategist core must not delete legacy data merely because the external
skill is absent or the import is incomplete.

## Rollback

### Build-time rollback

1. Restore the previous committed lock/catalog/adapter revision.
2. Remove only temporary staging content created by the failed preparation.
3. Re-run generated-output parity checks against the previous lock.
4. Rebuild from the last known-good inputs.
5. Confirm the release inventory no longer advertises the failed candidate.

### Runtime/activation rollback

1. Keep the last-known-good embedded catalog or workspace binding active.
2. Disable or quarantine the failed skill/instance.
3. Do not switch command bindings to an unprobed artifact.
4. Capture the reason code, package/adapter digests, connector and binding
   generation.
5. Retry only after the package or adapter correction has been reviewed.

### Migration rollback

1. Stop if validation reveals loss, duplicate IDs, provenance mismatch or
   non-idempotent behavior.
2. Do not delete the source legacy data.
3. Use the external skill's documented reverse/import recovery procedure, or
   restore the destination from the pre-import backup.
4. Verify source and destination hashes/counts after recovery.
5. Leave the workspace marked as migration-incomplete until a new approved
   attempt is available.

## Verification checklist

- [ ] Lock is committed and unchanged during ordinary ingestion.
- [ ] Artifact digest, license, provenance and API compatibility pass.
- [ ] Dependencies and requested permissions are resolved.
- [ ] Generated catalog and command registrations are deterministic.
- [ ] Command paths do not collide with core or another embedded skill.
- [ ] Release inventory lists exact embedded IDs, versions and digests.
- [ ] Embedded presence is not reported as live invocation or health.
- [ ] Legacy data was inventoried and preserved before migration.
- [ ] Migration is explicit, idempotent and provenance-preserving.
- [ ] Failed candidates leave the last-known-good state active.
- [ ] User/local plugins remain outside the embedded release catalog.

## Stop conditions

- digest, license, provenance, schema or API mismatch;
- unresolved command collision or permission expansion;
- missing required artifact or generated-output drift;
- migration contract absent or legacy data would be deleted;
- rollback target unavailable;
- attempted mutable-tag refresh during ordinary release build;
- attempt to bypass approval, connector, grant or native fallback policy.

## Escalation record

Record at minimum:

```yaml
incident:
  package_id: <publisher/name>
  package_digest: sha256:<digest>
  adapter_revision: <revision>
  lock_digest: sha256:<digest>
  connector_id: <connector-or-none>
  stage: <resolve|verify|generate|build|probe|invoke|migrate|rollback>
  reason_code: <stable-code>
  last_known_good: <version-or-digest>
  legacy_data_preserved: true
  next_action: <review|repair|retry|rollback|separate-migration>
```

## References

- [ADR-0032 — External Skill CLI Embedding and Treasure Chest Ownership](../adr/0032-external-skill-cli-embedding-and-treasure-chest-ownership.md)
- `docs/adr/0030-world-class-strategist-plugin-architecture.md`
- `docs/adr/0029-external-skill-provider-lifecycle.md`
- `docs/runbooks/provider-fallback-policy.md`
