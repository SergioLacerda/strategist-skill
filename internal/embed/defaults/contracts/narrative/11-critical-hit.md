---
phase: critical_hit
requires_approval: true
slot: execution
contract: controlled
---

# Strategist — Contract 11: Critical Hit

## Purpose

Internal capability for **workspace artifact management** — moving, archiving,
reopening, or closing `.md` analysis/documentation artifacts between the workspace
folders (`pending/`, `refined/`, `archived/`, `done/`) without running the full
Ranger/Archivist pipeline.

Critical Hit is the only Strategist capability that closes finalized analysis cards
from `pending/` or `refined/` into `done/`. Opportunity Attack is unrelated to this
movement; it only evaluates whether refined work should produce an ADR side quest.

Critical Hit is not a route mutually exclusive with the pipeline — it is an internal
capability that may fire at intake, or mid-mission at any phase boundary, whenever the
current request reduces to a pure artifact move/closure. Firing Critical Hit mid-mission
does not abandon or restart the enclosing mission; it handles the move and returns
control to wherever the mission was.

Critical Hit has two modes:

- **Plain move** — relocate an artifact between `pending/`, `refined/`, or `archived/`.
  No content evaluation, no completion claim, no evidence required.
- **Closure move** — relocate a `pending/` or `refined/` package into `done/` because the
  underlying demand has been implemented or validated. This mode requires an explicit
  completion claim and a supplied evidence summary (see Closure Requirements below); it
  never infers completion on its own.

In both modes, Critical Hit does not perform analysis and does not evaluate document
content beyond what closure mode explicitly requires (the evidence summary, supplied by
the user or delegating agent — never invented). If a request is ambiguous between a plain
move and a closure, or between Critical Hit and full analysis, it falls back to
`main_mission`.

A rota é selecionada internamente pela camada de routing após o intake, ou
re-avaliada a qualquer transição de fase durante uma missão em andamento. O delegatário
não precisa solicitar Critical Hit explicitamente, mas deve fornecer confirmação e
evidência quando o gate de fechamento (closure) exigir.

## When to Apply

### Plain Move

Fires when **all** conditions are true:

- `task_type` is `analysis_move`
- Source path is within `<base_path>/pending/`, `<base_path>/refined/`, or `<base_path>/archived/`
- Target path is within `<base_path>/pending/`, `<base_path>/refined/`, or `<base_path>/archived/`
- Files are `.md` only
- `risk_level` is `low`
- Number of files ≤ 5
- No completion claim is present

### Closure Move

Fires when **all** conditions are true:

- `task_type` is `analysis_move`
- Source path is within `<base_path>/pending/` or `<base_path>/refined/`
- Target path is `<base_path>/done/<id>`
- The user has explicitly stated the demand is complete/implemented/validated, or a
  stale-card detection (see below) surfaced the package and the user confirmed it
- An evidence summary is available (supplied by the user or the delegating agent —
  never invented by Strategist)

And **none** of these are true, for either mode:

- Source or target is outside `<base_path>`
- Files include non-`.md` types
- The demand is only partially implemented, with declared residual work (closure move only)
- Completion is being inferred solely from code changes with no explicit evidence (closure move only)

**When in doubt → main_mission. Conservatism is the safe default.**

Critical Hit never evaluates whether a demand was implemented beyond the evidence
explicitly supplied for a closure move. It never decides implementation status on its own.

## Stale Card Detection

A Strategist analysis/refinement mission finishing is not, by itself, evidence that the
underlying demand was implemented or validated. Leaving a package in `refined/` after a
main_mission completes is the normal, expected end state — not a gap to be closed. Stale
Card Detection exists only to surface packages that already carry implementation or
validation evidence but have not yet been moved, not to nudge every completed mission
toward `done/`.

Critical Hit self-checks at two points:

1. **Discovery (Ranger).** While Ranger surveys the workspace for the current mission, any
   other `pending/` or `refined/` package it encounters that already carries an explicit
   implementation/validation claim and evidence (e.g., a prior conversation recorded that
   the demand was implemented/validated, or a completion-report draft already exists) but
   was never moved, is flagged as a closure candidate. A package whose `tasks.md` is merely
   fully checked, or that reached `documentation_applied`, is NOT sufficient on its own —
   see Insufficient Evidence below. Ranger does not detour into evaluating or closing the
   candidate; it carries the flag forward to the next mission-close or intake moment.
2. **Stale scan on intake/bootstrap.** Same rule as discovery: intake/bootstrap only flags
   a `pending/` or `refined/` package as a closure candidate when it already carries a
   recorded implementation/validation claim and evidence, not merely a complete-looking
   status field. This is a flag, not a background job — Strategist does not go hunting
   through `pending/`/`refined/` unprompted outside of discovery and intake/bootstrap.

Reaching `documentation_applied` at the end of main_mission execution does **not** trigger
a closure check. It is documentation completion only (see `06-execution.md`). A completed
main_mission ends with its package in `refined/`, and that is correct — it does not require
Critical Hit to fire, and Critical Hit does not run an automatic candidacy check at that
point.

In both remaining trigger cases, detection only surfaces a candidate — it never
substitutes for the evidence requirement or the approval gate: the closure move still
requires explicit user confirmation and an evidence summary before Sniper writes the
completion report or moves the package.

## Insufficient Evidence

None of the following, alone or in combination, make a package a closure candidate or
satisfy the closure move's evidence requirement:

- `documentation_applied` (Sniper completed documentation materialization)
- a Sniper report existing under `<base_path>/archived/`
- the user accepting the Strategist Approval Gate for the refined package
- a refined package simply containing `tasks.md` with items in it
- `tasks.md` checkboxes being fully checked
- `implementation_handoff` items being marked ready

Closure requires an explicit statement that the underlying demand was implemented or
validated, plus a supplied evidence summary (tests/checks run, files changed, links, or
other concrete evidence — supplied by the user or delegating agent, never invented or
inferred by Strategist from code inspection).

## Valid Moves

| From | To | Mode | Use case |
|------|----|------|----------|
| `pending/<id>-analysis.md` | `archived/` | Plain | Abandon a stale pending analysis |
| `refined/<id>/` | `archived/` | Plain | Archive a completed refined set |
| `archived/<id>-*.md` | `pending/` | Plain | Reopen an archived analysis (rare) |
| `pending/<id>-analysis.md` | `done/<id>` | Closure | Completed demand, evidence supplied |
| `refined/<id>/` | `done/<id>` | Closure | Completed refined package, evidence supplied |
| Any `.md` within `pending/`, `refined/`, `archived/` | Any of those three folders | Plain | General artifact management |

## Closure Requirements

A closure move MUST collect or receive, at minimum:

- what was completed (summary)
- evidence supplied (tests run, files changed, links, user confirmation — as given,
  not invented)
- checks/tests run, if any
- unresolved residuals, if any

If evidence is missing or insufficient, the closure move MUST NOT proceed. Fall back to
requesting the missing evidence, or to `main_mission` if the request is ambiguous.

This is the same `evidence_state: explicit` bar Scout uses to route a request to
`critical_hit`/short route instead of `full_pipeline` (see `00-routing.md` § Scout —
Intake Router and § Annotation Limits). It does not relax or replace the Insufficient
Evidence list above — closure evidence must still be explicit and supplied, never
invented or inferred.

## Pipeline Difference

| Phase | main_mission | Critical Hit (plain) | Critical Hit (closure) |
|-------|-------------|----------------------|-------------------------|
| Ranger discovery | ✅ | ❌ skipped | ❌ skipped |
| Archivist refinement | ✅ | ❌ skipped | ❌ skipped |
| Opportunity attack | ✅ | ❌ skipped | ❌ skipped |
| Approval gate | Full gate | Inline gate | Inline gate |
| Sniper execution | ✅ | ✅ (move only) | ✅ (completion report + status update + move) |
| Artifacts written | analysis.md, proposal.md, design.md, tasks.md | none | completion report inside package |

## Inline Gate

Plain move:

```
Critical Hit detected.
Move: <source_path>
    → <target_path>
Confirm? (sim / nao)
```

Closure move:

```
Critical Hit detected (closure).
Package: <source_path>
Target:  <base_path>/done/<id>
Evidence summary required.
Confirm? (sim / nao)
```

- `sim/yes` → proceed to Sniper
- `nao/no` → resolve as `analysis_delivered` (nothing moved, nothing updated)

## Sniper Behavior (execution)

Plain move: relocate the file(s) as declared. No other write.

Closure move, on approval, Sniper (execution slot):

1. Writes a completion report into the package (e.g. `<source_path>/completion-report.md`):
   - what was completed
   - evidence supplied
   - tests/checks run, if any
   - unresolved residuals, if any
2. Updates `tasks.md` checkboxes **only** for tasks explicitly covered by the supplied
   evidence. Tasks not covered by evidence are left unchanged.
3. Moves the package to `<base_path>/done/<id>`.
4. Emits completion status and final artifact path.

Sniper never infers completion from source code inspection. Sniper never mutates source
code as part of Critical Hit — this remains a documentation-only operation, in both modes.

## Emit Events

- `critical_hit_triggered` — when conditions match and Critical Hit fires (plain or closure)
- `critical_hit_gate_approved` — user confirmed
- `critical_hit_gate_declined` — user declined
- `critical_hit_evidence_missing` — closure move attempted without sufficient evidence, blocked/fell back
- `critical_hit_stale_candidate_detected` — stale-card detection surfaced a closure candidate (see Stale Card Detection)

## FSM States

```
StateInit         → [EventCriticalHitIntent]                      → StateDirectGate
StateDirectGate   → [EventDirectGateApproved + execution_authorized] → StateDirectExec
StateDirectGate   → [EventDirectGateDeclined]                     → StateDoneAnalysis
StateDirectExec   → [EventSniperDone]                             → StateDirectDone
StateDirectExec   → [EventSlotTransient]                          → StateRetryingDirectExec
StateDirectExec   → [EventSlotPermanent]                          → StateBlocked
```

The same FSM shape covers both plain and closure moves; the mode only changes what
Sniper writes at `StateDirectExec` and what the inline gate displays. Stale-card
detection happens before `StateInit` — it decides whether `EventCriticalHitIntent` fires
proactively at discovery or intake, never automatically from a main_mission reaching
`documentation_applied` alone.

## Invariants

- Cannot fire for `task_type` other than `analysis_move`
- Cannot fire when source or target is outside `<base_path>` analysis folders
- Closure mode target MUST be `<base_path>/done/<id>` and MUST have an explicit evidence summary
- Approval is still required (inline gate is not auto-approve), for both modes
- Never marks a task complete without evidence explicitly covering it
- Never infers completion from code changes alone
- Never mutates source code
- Stale-card detection surfaces candidates only — it never auto-closes a package
- Reaching `documentation_applied` does NOT trigger a closure candidacy check and does NOT
  imply the package should move to `done/` — a main_mission ending with its package in
  `refined/` is the normal, expected terminal state, not a condition to correct
- None of `documentation_applied`, a Sniper report, Approval Gate acceptance, or a
  fully-checked `tasks.md` alone constitutes closure evidence (see Insufficient Evidence)
- Ranger MUST NOT detour into evaluating or closing a stale candidate encountered during
  discovery — it only flags the candidate for a later mission-close/intake moment
- If any condition is ambiguous, fall back to `main_mission`
- `StateDirectDone` is absorbing — no further transitions
