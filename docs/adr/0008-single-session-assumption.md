# ADR-0008 — Single-Session Workspace Assumption

**Status:** Accepted
**Date:** 2026-06-06

---

## Context

The Strategist skill was designed to operate with a single active session per workspace. When two Claude sessions run simultaneously against the same `.strategist/` and `.analysis/` directories, five classes of failure emerge (see analysis in `.analysis/refined/conflito_multi_thread/design.md`).

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
- F3 may be revisited if parallel-session use cases become more frequent and the cost of git conflicts becomes unacceptable.
