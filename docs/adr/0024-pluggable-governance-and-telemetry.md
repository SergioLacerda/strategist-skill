# ADR-0024 — Pluggable Governance and AI-First Telemetry

**Title:** Pluggable Governance and AI-First Telemetry
**Status:** Accepted
**Date:** 2026-08-11
**Mission ID:** 20260811-governanca-plugavel

## Context

Strategist must be governable without becoming a second organizational governance system. Its role is to declare contracts, emit evidence, expose state, record local violations, expose hooks, produce structured telemetry, and accept decisions from an external authority — leaving compliance, cross-project auditing, and policy enforcement to an external system (e.g. `sdd-harness`).

Repository inspection found that Strategist already emits structured telemetry — OpenTelemetry tracing (`internal/telemetry/setup.go`), a `PolicyEvent` envelope with `correlation_id` (`internal/telemetry/policy_event.go`), and JSONL outcome persistence (`internal/telemetry/outcome*.go`) — but has no unifying `EventSink` interface, no `internal/telemetry/sink/{noop,slog,jsonl,otel,external}` layout, and no `contract_id`/`authority` fields in any emitted event. It also found `internal/governance/sync.go`, which solves a narrower, adjacent problem (reconciling `skill.yaml` against `.sdd/` mandates at install/compile time) and is itself named-coupled to `.sdd/` — the opposite of the "no mandatory dependency on a named governance system" principle already declared in Strategist's own routing contracts (`.strategist/contracts/narrative/00-routing.md`).

Two alternatives were considered for closing this gap:

- **Do nothing / defer indefinitely** — leave telemetry and governance as separate, purpose-built implementations. Rejected: leaves `contract_id`/`authority` unavailable for drift analysis, and leaves `internal/governance` coupled to one named external system, contradicting an already-published invariant.
- **Big-bang replacement** of `PolicyEvent` and related event types with a single new `Event` envelope. Rejected for this decision: `legacy_compatibility: required` is the default mission constraint, and no explicit user instruction authorized a breaking migration; an additive/phased approach is the safer default (open question UNC-01 in `design.md`, not resolved by this ADR — it remains an implementation-time decision).

## Decision

Adopt the following design:

1. Introduce an `EventSink` interface and a standardized event envelope (`event`, `event_version`, `mission_id`, `correlation_id`, `contract_id`, `phase`, `expected`, `observed`, `severity`, `decision`, `authority`, `timestamp`).
2. Introduce `internal/telemetry/sink/{noop,slog,jsonl,otel,external}` as the target layout, migrating existing OTel and JSONL logic into it without regressing current test coverage.
3. Introduce an optional `GovernanceBridge` interface (`Evaluate(ctx, GovernanceRequest) (GovernanceDecision, error)`), kept ignorant of any concrete governance system's internal logic.
4. Treat the `internal/governance` package's `.sdd/`-specific coupling as an accepted side quest (SQ-002 / task 7) for this mission — to be resolved by decoupling the `.sdd/` read path into a concrete adapter, keeping `RunSync`/`SyncReport` stable.
5. Defer the exact package placement of `GovernanceBridge` (new package vs. generalized `internal/governance`) and the additive-vs-substitutive shape of `EventSink` relative to `PolicyEvent` to implementation time — recorded as open questions (UNC-01, UNC-02) in the refined package, not settled by this ADR.

This is an analysis/documentation decision only. No code was written or modified by Strategist as part of this mission; implementation of items 1–4 above is out of Strategist's execution scope (`task_type: implementation_handoff`) and requires a separate coding task.

## Consequences

**Positive:**
- Strategist telemetry gains a pluggable, testable abstraction (`EventSink`) instead of parallel, purpose-built sink implementations.
- Governance enforcement can be delegated to an external authority through `GovernanceBridge` without Strategist knowing that authority's internal logic.
- `internal/governance` stops being implicitly and irreversibly tied to `.sdd/` by name, aligning the codebase with the invariant already published in `00-routing.md`.
- `contract_id` and `authority` become available on critical events, closing a traceability gap identified during discovery (KF-09).

**Negative / follow-up required:**
- Two open design questions (UNC-01: additive vs. substitutive `EventSink`; UNC-02: package placement of `GovernanceBridge`) are not resolved by this ADR and must be closed before implementation tasks can be written precisely.
- Migrating existing sinks (`setup.go`, `outcome*.go`) into `internal/telemetry/sink/` risks regressing existing tests if not done incrementally; `legacy_compatibility: required` constrains this to a phased migration.
- Decoupling `internal/governance` from `.sdd/` (task 7 / SQ-002) touches `cmd/strategist/sync_governance*.go` as well as `internal/governance/sync_test.go`, widening the blast radius slightly beyond the original `internal/governance` package boundary.
- No strict-mode (blocking telemetry failure) parameter exists today; whether it belongs in this mission's MVP remains open (UNC-03).
