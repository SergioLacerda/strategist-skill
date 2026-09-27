# ADR-0057 — Mission State Concurrency and Atomicity

**Status:** Proposed
**Date:** 2026-09-27
**Related:** ADR-0008 (single-session assumption), ADR-0044 (preflight result)
**Mission:** `20260927-strategist-hardening-review` (finding F-H2)

## Context

Mission state is persisted as one JSON file per mission under `<strategist-root>/missions/<id>.json`.
Two call paths write it, both in `cmd/strategist/`:

- `RunSubmit` (`cmd/strategist/mission/submit.go:27-40`) — `deps.Load` → `engine.Submit(event)` →
  `deps.Save`.
- `RunStart` (`cmd/strategist/mission/start.go:47-69`) — `deps.RequireNoExisting` → … →
  `deps.Save`.

Both resolve to `saveMission` (`cmd/strategist/mission_persistence.go:43-55`):

```go
func saveMission(root string, status domain.MissionEngineStatus) error {
	if err := os.MkdirAll(filepath.Join(root, "missions"), 0o755); err != nil {
		return fmt.Errorf("create mission directory: %w", err)
	}
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("encode mission state: %w", err)
	}
	if err := os.WriteFile(missionPath(root, status.MissionID), append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("write mission state: %w", err)
	}
	return nil
}
```

There is no file lock, no compare-and-swap on a version field, and no atomic temp-plus-rename. This
yields two distinct defects that happen to share one function:

**1. Lost transition.** The `Load → Submit → Save` sequence is an unguarded read-modify-write. Two
commands that submit events for the same `mission_id` concurrently both read the same prior state,
each applies its own event to that state, and the later `Save` overwrites the earlier one. The FSM
accepted both events; only one survives on disk. This is not hypothetical for this workspace:
parallel sessions are the normal operating mode, and `docs/runbooks/concurrent-session-sniper-collision`
already treats cross-session collision as a real condition.

`RunStart` carries the same shape in smaller form: `RequireNoExisting` then `Save` is a
check-then-create window that two simultaneous starts with the same id both pass.

**2. Unrecoverable truncation.** `os.WriteFile` opens with `O_TRUNC`. An interrupt between the
truncate and the write leaves a partial JSON. `loadMission` then fails with `invalid persisted
state`, and every subsequent command against that `mission_id` fails the same way. The
`restarting-orphaned-strategist-missions` runbook exists for the symptom; this write path is the
cause.

ADR-0008 recorded a single-session assumption and named an F3 revisit tripwire for when that
assumption stops holding. The tripwire's two signals are Git-conflict attribution
(`internal/telemetry/sniper_conflict.go`) and claim collision
(`internal/telemetry/sniper_claim.go`). Both are about *documentation targets* Sniper writes. Neither
covers mission state itself, which is the authoritative FSM record and is corrupted by a narrower,
more mechanical race than the one ADR-0008 anticipated.

The repository already owns a working answer to the same problem for a different file.
`internal/leveling/ledger_lock.go` wraps the role-level ledger in `withLedgerLock`, with
`flock`-based Unix (`ledger_lock_unix.go`) and Windows (`ledger_lock_windows.go`) implementations.
Mission state is simply not covered by it.

## Decision

Two decisions, deliberately separated because they have different risk profiles.

### D1 — Atomic write, unconditionally

`saveMission` writes to a sibling temporary file and renames it over the target. `os.Rename` is
atomic within a filesystem on POSIX, so a reader either sees the whole previous state or the whole
new state, never a truncated file.

This needs no trade-off analysis and no migration. It is unconditionally correct, it eliminates
defect 2 on its own, and it should land first and separately, so that the more debatable D1/D2
sequencing cannot delay it.

### D2 — Mutual exclusion via `flock`, generalized from the leveling ledger

The `Load → Submit → Save` window (and `RunStart`'s `RequireNoExisting → Save` window) is guarded by
an exclusive file lock, using a shared helper extracted from `internal/leveling/ledger_lock.go` and
its two platform implementations rather than a second copy of them.

The alternative considered was a `version` field on `domain.MissionEngineStatus` with a
compare-and-swap on save:

| | `flock` (chosen) | version + CAS |
| --- | --- | --- |
| Protects against | concurrent processes on one machine | concurrent writers on any filesystem |
| On conflict | serializes (second writer waits) | detects and reports the conflict |
| Over NFS | unreliable | reliable |
| State migration | none | needs a path for existing `missions/*.json` without the field |
| Code reuse | reuses two proven platform implementations | new code |

`flock` is chosen because it matches the actual deployment: parallel local sessions on one machine,
which is what this workspace does and what ADR-0008's tripwire is about. It reuses code that already
exists and already has a Windows path. It needs no migration of persisted state.

CAS would be the better answer if mission state were ever shared across machines or filesystems.
No ADR proposes that, and nothing in the runtime layout (`.strategist/` is generated, gitignored and
machine-local per ADR-0054) suggests it. If that changes, this decision is the one to revisit, and
D1 remains correct either way.

## Consequences

**Accepted:**

- A second concurrent `mission submit` for the same mission waits rather than failing. Serialization
  is the intended behavior: both events are legitimate and both should be applied, in some order.
- A stale lock file left by a killed process can block a mission until it is cleared. This is the
  same exposure the leveling ledger already carries, and it is why D1 lands separately: an atomic
  write is valuable even in a session that cannot acquire the lock.
- `flock` semantics do not protect against a writer on a different machine sharing the directory over
  NFS. This limitation is recorded rather than mitigated.

**Rejected:**

- Serializing through a single long-lived process or a daemon. Strategist is a CLI invoked as
  subprocesses by an agent; there is no process to hold the state.
- Leaving the race documented but unfixed on the grounds that ADR-0008 assumed a single session. The
  assumption is already known not to hold — two runbooks exist for its consequences — and the cost of
  the fix is a shared helper plus a rename.

## Verification

The test suite was green (937 tests across `internal/telemetry`, `internal/handoff`,
`internal/leveling`, `internal/mission`, `internal/runbook`, `cmd/strategist`) while both defects were
present, so passing tests are not evidence here. Each decision needs a test at the seam:

- **D1** — a test that injects a failure between write and rename and asserts the previous state is
  still parseable by `loadMission`.
- **D2** — a concurrent-submit test asserting no event accepted by the FSM is absent from the final
  persisted state, and a concurrent-start test asserting exactly one mission is created.

## Status note

This ADR records a decision about how to fix the defect. It does not claim the fix is implemented.
The implementation is tasks 2.1 and 2.2 of
`.analysis/refined/20260927-strategist-hardening-review/tasks.md`, both classified
`implementation_handoff` — they mutate Go source and are outside Sniper's and Strategist's execution
scope. Approval of the analysis package that produced this ADR did not authorize executing them.
