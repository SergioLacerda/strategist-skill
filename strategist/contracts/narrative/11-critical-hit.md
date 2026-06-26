---
phase: critical_hit
requires_approval: true
slot: execution
contract: controlled
---

# Strategist — Contract 11: Critical Hit

## Purpose

Decision point between **full pipeline** (main_mission) and **direct route** (direct_execute).
Evaluated after Intake, before any slot provider is invoked.

## When to Apply

Critical Hit fires when **all** conditions are true:

- `task_type` is one of: `doc_edit`, `content_update`, `readme_update`, `comment_update`
- `risk_level` is `low`
- Number of affected files ≤ 5
- No cross-module dependencies
- No ADR required
- No architectural decision involved

And **none** of these are true:

- Prompt contains keywords: `investigate`, `propose`, `audit`, `design`, `redesign`, `architecture`, `refactor`
  (also matched in Portuguese: `investigar`, `propor`, `auditar`, `projetar`, `redesenhar`, `arquitetura`, `refatorar`)
- `task_type` is: `architecture_analysis`, `refactor`, `security_audit`, `new_feature`

**When in doubt → main_mission. Conservatism is the safe default.**

## Pipeline Difference

| Phase | main_mission | direct_execute (Critical Hit) |
|-------|-------------|-------------------------------|
| Ranger discovery | ✅ | ❌ skipped |
| Archivist refinement | ✅ | ❌ skipped |
| Opportunity attack | ✅ | ❌ skipped |
| Approval gate | Full gate (reads tasks.md) | Inline gate (single message) |
| Sniper execution | ✅ | ✅ |
| Artifacts written | analysis.md, proposal.md, design.md, tasks.md | none |

## Inline Gate

```
Critical Hit detected.
Task: <task description>
Files: <file list>
Confirm direct materialization? (yes / no)
```

- `yes` → proceed to Sniper
- `no` → resolve as `analysis_delivered` (nothing written)

## Emit Events

- `critical_hit_triggered` — when conditions match and Critical Hit route is selected
- `critical_hit_gate_approved` — user confirmed
- `critical_hit_gate_declined` — user declined

## FSM States

```
StateInit → [EventDirectHitIntent] → StateDirectGate
StateDirectGate → [EventDirectGateApproved] → StateDirectExec
StateDirectGate → [EventDirectGateDeclined/analysis_delivered] → StateDoneAnalysis
StateDirectGate → [EventDirectGateDeclined] → StateDoneAnalysis
StateDirectExec → [EventSniperDone] → StateDirectDone
StateDirectExec → [EventSlotTransient] → StateRetrying
StateDirectExec → [EventSlotPermanent] → StateBlocked
```

## Invariants

- Cannot fire when `risk_level != low`
- Cannot fire for cross-module or architectural tasks
- Approval is still required (inline gate is not auto-approve)
- If any condition is ambiguous, fall back to `main_mission`
- `StateDirectDone` is absorbing — no further transitions
