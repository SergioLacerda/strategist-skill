---
phase: critical_hit
requires_approval: true
slot: execution
contract: controlled
---

# Strategist — Contract 11: Critical Hit

## Purpose

Rota direta para **manutenção de artefatos de análise e documentação** — mover,
arquivar ou reabrir arquivos `.md` entre as pastas do workspace (`pending/`, `refined/`, `archived/`)
sem passar pelo pipeline completo de Ranger/Archivist.

Critical Hit não realiza análise. O conteúdo dos documentos não é avaliado —
apenas movido ou organizado dentro do escopo do workspace.

A rota é selecionada internamente pela camada de routing após o intake.
O delegatário não precisa solicitar Critical Hit explicitamente.

## When to Apply

Critical Hit fires when **all** conditions are true:

- `task_type` is `analysis_move`
- Source path is within `<base_path>/pending/`, `<base_path>/refined/`, or `<base_path>/archived/`
- Target path is within `<base_path>/pending/`, `<base_path>/refined/`, or `<base_path>/archived/`
- Files are `.md` only
- `risk_level` is `low`
- Number of files ≤ 5

And **none** of these are true:

- Source or target is outside `<base_path>`
- Files include non-`.md` types

**When in doubt → main_mission. Conservatism is the safe default.**

## Valid Moves

| From | To | Use case |
|------|----|----------|
| `pending/<id>-analysis.md` | `archived/` | Abandon a stale pending analysis |
| `refined/<id>/` | `archived/` | Archive a completed refined set |
| `archived/<id>-*.md` | `pending/` | Reopen an archived analysis (rare) |
| Any `.md` within the three folders | Any of the three folders | General artifact management |

## Pipeline Difference

| Phase | main_mission | Critical Hit |
|-------|-------------|--------------|
| Ranger discovery | ✅ | ❌ skipped |
| Archivist refinement | ✅ | ❌ skipped |
| Opportunity attack | ✅ | ❌ skipped |
| Approval gate | Full gate | Inline gate |
| Sniper execution | ✅ | ✅ |
| Artifacts written | analysis.md, proposal.md, design.md, tasks.md | none |

## Inline Gate

```
Critical Hit detected.
Move: <source_path>
    → <target_path>
Confirm? (sim / nao)
```

- `sim/yes` → proceed to Sniper
- `nao/no` → resolve as `analysis_delivered` (nothing moved)

## Emit Events

- `critical_hit_triggered` — when conditions match and Critical Hit route is selected
- `critical_hit_gate_approved` — user confirmed
- `critical_hit_gate_declined` — user declined

## FSM States

```
StateInit         → [EventCriticalHitIntent]                      → StateDirectGate
StateDirectGate   → [EventDirectGateApproved + execution_authorized] → StateDirectExec
StateDirectGate   → [EventDirectGateDeclined]                     → StateDoneAnalysis
StateDirectExec   → [EventSniperDone]                             → StateDirectDone
StateDirectExec   → [EventSlotTransient]                          → StateRetrying
StateDirectExec   → [EventSlotPermanent]                          → StateBlocked
```

## Invariants

- Cannot fire for `task_type` other than `analysis_move`
- Cannot fire when source or target is outside `<base_path>` analysis folders
- Approval is still required (inline gate is not auto-approve)
- If any condition is ambiguous, fall back to `main_mission`
- `StateDirectDone` is absorbing — no further transitions
