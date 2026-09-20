# Phase D context cache

## Boundary

The cache is local to one workspace process. It is not shared across runtimes
or providers. Every entry is addressed by source, runtime, provider, policy,
and content digests.

## Retention and privacy

Retention is at most 24 hours. The cache stores only materialized context for
the current process; it does not persist credentials or provider payloads.

## Outcomes

A hit requires an exact identity match. Miss, stale, corrupt, and unavailable
cache states fall back to bounded source materialization. A cache error must
never return content from another identity.

## Migration and rollback

Enable the cache only after identity validation succeeds. To disable it, pass
no cache to materialization; source materialization remains correct and no
source evidence is deleted.
