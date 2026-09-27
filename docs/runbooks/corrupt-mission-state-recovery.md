# Runbook: Recovering a Corrupt Mission State File

## Trigger

Use this runbook when any `strategist mission` subcommand fails against a mission id with an error
naming the persisted state, for example:

```
mission submit: mission "20260927-example" not found: ...
mission submit: invalid persisted state: unexpected end of JSON input
mission submit: restore mission state: <domain validation error>
```

The file in question is `<strategist-root>/missions/<mission-id>.json`. All three errors come from
`loadMission` (`cmd/strategist/mission_persistence.go:57-71`), and each points at a different cause.

This procedure is a controlled recovery of the authoritative FSM record. It never fabricates a phase
or state the mission did not reach, and it never treats a repaired state file as evidence that the
work of the missing phases was done.

## Safety boundary

- **Read-only first.** Do not write to `missions/<id>.json` until the cause is classified. An
  overwrite destroys the only evidence of what the corruption was.
- **Rule out a live owner.** Another session may be mid-write. If any activity is present, or cannot
  be ruled out, stop with `blocked reason=mission_in_progress`. See the § Lost update section for why
  a *second writer* is a distinct cause from a *truncated write*.
- **Snapshot before repair.** Copy the damaged file to a recoverable location outside
  `missions/` before any edit.
- **No Git mutation.** Nothing in this runbook runs `git add`, `git commit`, `git reset` or any other
  state-modifying Git operation. `.strategist/` is generated and gitignored; there is nothing to
  commit.
- **Repair is not implementation evidence.** Restoring a state file does not advance the mission's
  actual work. A repaired mission resumes from the phase its *artifacts* support, not from the phase
  the JSON claims.

## Step 1 — Classify the cause

Read the file without changing it:

```
cat <strategist-root>/missions/<mission-id>.json
```

| What you see | Cause | Go to |
| --- | --- | --- |
| File does not exist | Never created, or deleted | § Missing file |
| Valid JSON, truncated mid-structure (no closing brace, cut mid-key) | Interrupted write | § Truncated write |
| Complete, parseable JSON, but the phase/state is *earlier* than events you know were accepted | Lost update from a concurrent writer | § Lost update |
| Complete JSON that `RestoreMission` rejects | Invalid state combination | § Invalid state |

The distinction between the middle two matters. A truncated write is a durability failure and the
file is unusable. A lost update produces a *perfectly valid* file that is simply behind — nothing in
it looks wrong, and only your knowledge of which events were submitted reveals the loss.

Both causes are recorded in ADR-0057 (Mission State Concurrency and Atomicity): `saveMission` uses
`os.WriteFile`, which truncates in place, and `RunSubmit` performs an unguarded
`Load → Submit → Save`. Until ADR-0057's decisions are implemented (tasks 2.1 and 2.2 of
`.analysis/refined/20260927-strategist-hardening-review/tasks.md`), both remain reachable.

## Step 2 — Snapshot

```
cp <strategist-root>/missions/<mission-id>.json /tmp/<mission-id>.json.damaged
```

Use `command cp` if `cp` is aliased interactively in your shell. Record the timestamp and the
damaged content in whatever incident note you keep — the file itself is about to change.

## Step 3 — Reconstruct the true phase from artifacts, not from the JSON

The state file is the FSM's record, but it is not the only record of what happened. Reconstruct the
mission's real progress from evidence that survives independently:

| Evidence | Location | Tells you |
| --- | --- | --- |
| Analysis artifact frontmatter | `<base_path>/pending/<id>-analysis.md` or `<base_path>/refined/<id>/analysis.md` | `mission_status` — the phase the *work* reached |
| Refined package presence | `<base_path>/refined/<id>/` (four files) | Refinement completed |
| Route decision | `<strategist-root>/memory/route-decisions.jsonl` | Which route, and therefore which evidence the execution boundary requires |
| Confidence records | `<strategist-root>/memory/confidence-records.jsonl` | Which boundaries recorded claims |
| Handoff metrics | `<strategist-root>/memory/handoff-metrics.jsonl` | Refinement completed and was recorded |
| Handoff challenges | `<strategist-root>/memory/handoff-challenges.jsonl` | A challenge ran, and its outcome |
| Role levels | `<strategist-root>/memory/role-levels.jsonl` | Which roles started |

Resolve `<base_path>` from `base_path` in `<strategist-root>/active.yaml`. It is not necessarily
`.analysis/`.

The `mission_status` frontmatter in the analysis artifact is the most reliable signal, because it is
written by the role that did the work and it lives outside `missions/`. Where the frontmatter and a
repaired JSON disagree, **the frontmatter wins** — it describes the artifacts that actually exist.

## Step 4 — Choose recovery or restart

### Missing file

If no artifacts exist either, nothing was done: start the mission again with
`strategist mission start --mission-id <id>`.

If artifacts exist but the state file does not, prefer **restart with a new mission id** over
reconstructing state by hand. Reconstruction means hand-writing JSON that the FSM will trust, and a
wrong phase can let a mission enter execution without the evidence its route requires. A new mission
id that reuses the existing artifacts costs one `mission start` and keeps every guard intact.

### Truncated write

The file is unusable and its content is not recoverable. Follow § Missing file. Do not attempt to
repair the JSON by closing the structure: the truncation point is arbitrary and a syntactically
repaired file can carry a phase that was never reached.

### Lost update

The file is valid and behind. Resubmit the missing events in order:

```
strategist mission submit --mission-id <id> --event <event>
```

The FSM rejects an event that is invalid for the current state, so a wrong guess fails closed rather
than corrupting further. Resubmit only events whose work you confirmed in Step 3 — an event is a
claim that a phase completed, and the execution boundary
(`internal/mission.EvaluateExecutionEntry`) checks that claim against real evidence when
`handoff_challenge_passed` is submitted.

**Before resubmitting, establish that the concurrent writer is gone.** A lost update means two
writers existed; resubmitting while the second is still running reproduces the race.

### Invalid state

`RestoreMission` rejected a combination of phase and state the FSM considers impossible. This is
either hand-editing or a bug. Do not repair it by guessing a valid combination. Follow § Missing file
and report the damaged snapshot — an invalid combination that was not hand-written is worth a
diagnostic mission of its own.

## Verification

- `strategist check --json` reports `status: ready` (a corrupt mission does not block preflight, so a
  `ready` result here confirms only that the runtime is intact).
- `strategist mission status --mission-id <id>` returns a phase and state without error.
- The phase reported agrees with the `mission_status` frontmatter of the analysis artifact.
- No mission entered `EXECUTION` as part of this recovery without its route's required evidence: for
  `full_pipeline` that is discovery, refinement, tasks and the approved gate.
- The damaged snapshot is preserved outside `missions/`.
- No source, test, configuration or Git mutation was performed.

## Rollback

There is nothing to roll back if Step 1 was honored and the file was not written before
classification. If a repair was attempted and made things worse, restore the snapshot from Step 2 and
follow § Missing file instead. The snapshot is the only rollback point this procedure creates, which
is why Step 2 precedes every write.

## Prevention

This runbook exists because two defects make the failure reachable. Both are recorded in ADR-0057 and
neither is fixed as of 2026-09-27:

- non-atomic write (`os.WriteFile` truncates in place) — ADR-0057 § D1;
- unguarded read-modify-write in `RunSubmit`, and check-then-create in `RunStart` — ADR-0057 § D2.

Until those land, the practical mitigations are: avoid submitting events for one mission id from two
sessions at once, and do not interrupt a `mission submit` mid-run.

## Related

- ADR-0057 — Mission State Concurrency and Atomicity (the decision record for the underlying defects)
- ADR-0008 — Single-session assumption and its F3 revisit tripwire
- `docs/runbooks/restarting-orphaned-strategist-missions.md` — when the *artifact* is orphaned rather
  than the state file
- `docs/runbooks/concurrent-session-sniper-collision.md` — when two sessions collide on a
  documentation target rather than on mission state
