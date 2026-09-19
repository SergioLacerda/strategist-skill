# Phase D context-materialization extension

## Admission

The only supported Phase D extension is `context_materialization` at the
existing `discovery` slot. Registration requires version `v1`, a declared
authority, empty write scope, and a rollback action. Unknown slots, versions,
or write scopes are rejected without execution.

## Compatibility and migration

Treat the adapter as optional. Existing discovery behavior remains compatible
when it is absent. Register `v1` only after validating the context identity
and cache contract; no new phase, slot, dynamic acquisition, or provider
fallback is introduced.

## Failure and rollback

The extension receives no core pipeline state and cannot mutate declared
paths. On adapter failure, retain the last-known-good materialization value,
report the failure, and disable the adapter. Do not change approval-gate or
provider-resolution decisions during recovery.
