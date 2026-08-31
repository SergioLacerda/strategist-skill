# ADR-0008 — Single-Session Workspace Assumption

**Status:** Accepted
**Date:** 2026-06-06

---

## Context

The Strategist skill was designed to operate with a single active session per workspace. When two agent sessions run simultaneously against the same configuration and runtime-workspace directories, five classes of failure emerge: competing mission claims, interleaved artifact writes, inconsistent status transitions, conflicting memory updates, and approval/execution state observed out of order.

Three of these failures were mitigated:

- **F4** (partial `.config.gz` read): resolved with atomic write via `os.Rename` in `writeGzJSON`
- **F1** (mission ID collision): mitigated via existence check in the intake contract
- **F5** (gate without session identity): resolved by adding `{mission_id}` to the personas' `approval_prompt`

One failure remains intentionally unmitigated:

**F3 — Sniper cross-session blind spot:** If Mission A and Mission B have a Sniper pointing at the same file, the second Sniper does not detect the other session's conflict. The result is a silent overwrite at the skill level — the collision surfaces as a git conflict at commit time, not before.

## Decision

F3 is accepted as a known limitation of the single-session architecture. The skill does not implement a cross-session lock registry or write coordination between sessions.

**Rationale:** The complexity of a distributed lock registry (lock file, timeout, dead-session detection, rollback) exceeds the benefit for a use case that already has adequate mitigation via git. The git conflict is visible, recoverable, and occurs before any push.

**Recommendation for users:** When running multiple sessions in parallel, avoid having two Snipers edit the same file in the same time window. Commit frequently between sessions to minimize the conflict window.

## Consequences

- The skill explicitly documents that session parallelism is not guaranteed safe for concurrent writes to the same file.
- Users who need full parallelism should use separate git worktrees.
- **F3 revisit tripwire:** F3 remains accepted while concurrent Sniper collisions are rare. Revisit this decision if telemetry shows either of these conditions over a rolling 30-day window:
  - two or more distinct Sniper sessions claim or attempt to materialize the same documentation target under the same `base_path`; or
  - three or more Git conflicts are attributed to files recently materialized by Sniper.

  Until one of these thresholds is observed, the project keeps relying on mission IDs, approval prompts, and Git conflict detection instead of a cross-session lock registry. No telemetry for this signal exists today — instrumenting it is tracked as separate implementation work, not part of this decision record.

---

## Addendum (2026-08-30)

The line above ("no telemetry for this signal exists today") is now only
partially accurate and is left as originally written above — this addendum
corrects the record without rewriting the original decision text.
`internal/telemetry/sniper_conflict.go` has since implemented
`F3ConflictThresholdMet` and `EmitSniperConflictSignal`, which measure the
**second** revisit tripwire (three or more Git conflicts attributed to
recently-materialized files). The **first** tripwire (two sessions claiming
or attempting to materialize the same documentation target) remains
uninstrumented, since it depends on the cross-session claim registry this
ADR still declines to build. See
`.analysis/refined/20260830-skill-gaps-triage/tasks.md` Task 2 for the
tracked follow-up.
