# Runbook: Concurrent-Session Sniper Collision (ADR-0008 F3)

## Symptom

Two Claude sessions running against the same `.strategist/`/base_path workspace
each have a Sniper materializing to the same file in the same time window. No
Strategist-level warning appears — the collision surfaces as a `git` conflict at
commit time, not before.

## Root Cause

ADR-0008 (Single-Session Workspace Assumption) intentionally does not implement a
cross-session lock registry: the complexity (lock file, timeout, dead-session
detection, rollback) was judged to exceed the benefit, given git already provides a
visible, recoverable mitigation before any push.

## Resolution Steps

1. Avoid running two sessions with Snipers targeting the same file in the same
   time window — this is a workflow discipline, not something Strategist enforces.
2. Commit frequently between sessions to shrink the window in which a collision can
   occur.
3. If a collision does occur, resolve it as a normal `git` merge conflict — there is
   no special Strategist-level recovery procedure; git's conflict markers are the
   full extent of the safety net.
4. If parallel-session use becomes routine, use separate git worktrees per session
   instead of relying on this mitigation.

## Reference

- `docs/adr/0008-single-session-assumption.md`
