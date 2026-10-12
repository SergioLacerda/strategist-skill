# ADR-0069 — Mission and phase token telemetry: lifecycle ledger, usage ledger and shared read surface

**Status:** Accepted (decision and contract amendment applied; Waves 1 to 4 implemented in this repository: lifecycle ledger, usage ledger, `mission usage`, `mission observe-usage`, event export and `mission verify-telemetry`; Wave 5 reframed by the operator as a non-blocking per-stage volume report at the Approval Gate, with no price table, budget or blocking threshold)
**Date:** 2026-10-10
**Mission:** `20261010-token-telemetry-waves-refinement` (refined package `.analysis/refined/20261010-token-telemetry-waves-refinement/`; original demand `.analysis/pending/ondas-implementacao-telemetria-tokens-strategist.md`)
**Related:** [ADR-0050](0050-confidence-governance-contract.md), [ADR-0057](0057-mission-state-concurrency-and-atomicity.md), [ADR-0064](0064-canonical-seven-family-taxonomy.md), [ADR-0044](0044-preflight-startup-consumes-preflightresult.md)

## Context

The operator needs to know how many tokens a mission consumed end to end and between the start
and end of each phase, with the quality of every number stated. The refinement of the demand
established, from code and contracts inspected on 2026-10-10:

- the only usage record is flat and agent-reported (`mission_id`, `tokens_in`, `tokens_out`,
  `source`, `reported_at`) with no phase, role, invocation or idempotency key
  (`internal/telemetry/mission_token_usage.go`);
- the mission state file stores `mission_id`, `phase` and `state` only; no phase timestamps are
  persisted;
- CLI timing metrics are `scope=per_invocation`;
- **the telemetry contract (`10-telemetry.md`, finding F6) forbade rendering that timing as
  mission or phase elapsed time and forbade deriving a "phase total" from the usage ledgers**,
  which is exactly what the requested capability needs;
- `mission view` sums the legacy ledger as if every record were incremental
  (`internal/mission/view/token_usage.go`), while the agent protocol asked for cumulative reports
  at the gate and at DONE (finding F8), so the sum can double count;
- the Strategist binary never calls an LLM provider and cannot verify agent-reported counts.

## Decision

1. **Separate durable ledgers.** An append-only lifecycle ledger (mission and phase markers,
   idempotent by a deterministic key) and an append-only usage ledger (observations declaring
   phase, role, invocation, attempt, scope, record kind, source and quality). The mission state
   file stays the FSM authority and is not extended.
2. **Canonical phases are the pipeline phases**; Discovery, Refining and Delivery are display
   labels. The Approval Gate wait is its own duration-only row (gate answer Q3).
3. **Quality is part of every number**: `provider_measured`, `host_measured`,
   `reported_unverified`, `tokenizer_estimated`, `heuristic_estimated`, `unavailable`.
   Unobserved values render as unavailable, never as zero; estimates are never labeled measured.
4. **One aggregation, two entry points.** A shared builder in `internal/mission/view` feeds both
   `mission view` and a new `mission usage` command (gate answer Q1).
5. **No backward-compatibility layer.** The legacy ledger is shown on its own row, outside the
   mission total, and is never imputed to a phase; no migration is designed (gate answer Q2).
6. **The artifact requirement for usage reports is kept.** A usage report is accepted only for a
   mission that already has an artifact; usage that precedes the first artifact is reported
   afterwards with an explicit phase or stays unavailable. Lifecycle markers do not depend on an
   artifact (follow-up answer to Q4).
7. **Contract amendment first.** The `10-telemetry.md` rule that forbade phase totals and phase
   elapsed time is amended before any implementation; per-invocation timing remains forbidden as
   elapsed time. The amendment is applied together with this ADR (see Consequences).
8. **Automation is gated by a spike.** Host usage observation for Claude Code and Codex is
   investigated before any adapter is committed (gate answer Q5). Waves 4 (shared sinks,
   OpenTelemetry export, integrity verifier) and 5 (budgets, pricing, history) stay deferred.

## Consequences

- Waves 1 and 2 are useful even if the spike returns no-go (quality `reported_unverified`).
- Legacy records stop contributing to any total once the shared aggregation exists; operators
  lose the already misleading legacy sum in `mission view`.
- **Applied with this ADR:** the `10-telemetry.md` amendment (phase elapsed time and phase usage
  may be rendered only from the two new ledgers; nothing is rendered until they exist), the
  matching `04-refinement.md` sentence, the agent-protocol guidance (one observation per
  completed invocation, never a running total), and a "planned" section in
  `docs/observability-contract.md`.
- **Deliberately not applied yet:** the new `report-usage` flags (`--phase`, `--invocation-id`,
  `--record-kind`, …) and any wording that instructs agents to use them. Those flags do not
  exist; they belong to Wave 2 together with the guidance replacement (task 3.4 of the package).
  The guidance applied now therefore uses only today's flags.
- Because the protocol now asks for one incremental observation per invocation, the existing
  summing reader becomes correct for new reports; records written before this change remain
  ambiguous.
- A new ledger adds a multi-writer append concern that must be settled before Wave 2.

## Alternatives considered

- **Timestamps in the mission state file.** Rejected: it couples telemetry durability to the FSM
  authority and its corruption-recovery path.
- **Extend `mission view` only.** Rejected at the gate in favor of a dedicated command over a
  shared builder.
- **Keep the legacy ledger additive and summed.** Rejected: its semantics are unknown.
- **Derive per-phase usage by subtracting context-window readings.** Rejected: window occupancy
  is not cumulative consumption.
- **Apply the full protocol wording now.** Rejected: it would instruct agents to pass flags the
  CLI does not yet accept.

## Open points

- Final telemetry event names against the existing event catalog.
- Concurrent appenders and Windows append semantics for the new ledgers.
- Classification of the `HANDOFF_CHALLENGE` engine state, currently attributed to the Approval
  Gate row until execution entry.
- Exact event-to-marker mapping for the transitions marked unverified in the lifecycle hook map.
- The refined package's `tasks.md` and `design.md` still carry the pre-gate wording
  ("additive", open questions); the authority for the answered points is this ADR and
  `wave-0-ledger-contract.md`.
