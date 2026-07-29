# Runbook: Legacy Jewel `active` Status Migration

## Symptom

`loadJewels` fails loudly (not a silent fallback) on a jewel entry with
`status: active`.

## Root Cause

The pre-`treasure-chest-index-mine-pipeline` schema used a two-state
`active | deprecated` model. `active` is a removed legacy status — the current
lifecycle is `proposed → accepted → verified`, or `→ deprecated`.
`ValidateJewelStatus` (`internal/domain/jewel_grade.go`) rejects `active` outright
rather than silently coercing it, so workspaces created before the lifecycle
pipeline need an explicit one-time migration.

## Resolution Steps

1. Run `strategist treasure-chest items migrate-status` once.
2. The command rewrites every `status: active` entry to `status: accepted`, in
   place, across both monolithic (`jewels.yaml`) and partitioned
   (`jewels/<chest-id>.yaml`) manifests.
3. The command is idempotent and reports how many entries it migrated —
   `0` is a valid, non-error outcome if there was nothing to migrate.

## Reference

- `docs/cli-reference.md:570-577`
- `docs/adr/0012-jewel-lifecycle-statuses.md`
