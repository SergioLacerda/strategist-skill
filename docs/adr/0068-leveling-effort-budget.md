# ADR-0068 — LEVELING effort budget

**Status:** Proposed
**Date:** 2026-10-10
**Mission:** `20261010-leveling-integration-gaps-evaluation` (refined package `.analysis/refined/20261010-leveling-integration-gaps-evaluation/`; original demand `.analysis/refined/20261005-leveling-integration-gaps-review/`)
**Related:** [ADR-0064](0064-canonical-seven-family-taxonomy.md), [ADR-0065](0065-provider-integration-mechanism.md), [ADR-0050](0050-confidence-governance-contract.md)

## Context

LEVELING resolves a model, a capability and an effort tier per Role. It had no bounded representation
of how much orchestration a Role may mobilize, so delegation depth, evidence depth, escalation and
exhaustion were advisory ideas rather than governed behavior. The demand
`20261005-leveling-integration-gaps-review` added an additive effort budget. An evaluation of the
working tree (2026-10-10) found integration gaps that are now closed by the refined implementation
demand; the original package remains an uncommitted working-tree change.

What exists:

- a versioned envelope `strategist-effort-budget/v1` with closed vocabularies and validation
  (`internal/domain/effort_budget.go`), policy defaults and per-role budgets in
  `internal/embed/defaults/leveling.yaml`;
- issuance only after an authoritative provider mapping (`internal/tools/leveling/suggest.go`),
  correlated with mission, run and role (`internal/application/leveling/leveling.go`);
- a synchronous, fail-closed delegation boundary (`BudgetState.Begin/End` in
  `internal/tools/leveling/budget_runtime.go`) with a closed set of denial reasons;
- separate telemetry for recommendation, grant, consumption, denial, exhaustion, escalation and
  provider usage (`internal/telemetry/leveling_budget.go`), and an isolated append-only ledger with
  rotation (`internal/tools/leveling/budget_ledger*.go`);
- one consumer: the Ranger-to-Archivist provider pre-check (`cmd/strategist/ranger_budget.go`,
  `handoff_delegation_boundaries.go`);
- INITIATIVE observes the envelope through an immutable snapshot
  (`internal/feats/initiative/leveling_snapshot.go`).

## Decision

- **DEC-001. Availability contract.** A budget is issued only when a ranked provider mapping is
  authoritative. A host-only run, a manual run and an unknown provider issue none, and the consumer
  reports `leveling_budget_unavailable`, which is audited. An unresolved budget is never fabricated; a
  generic-fallback budget for host-only runs is rejected because it would claim an authority the
  provider mapping does not give.
- **DEC-002. Timing.** Mission start resolves only Scout. A Role's budget exists after that Role's own
  `leveling label` runs with a provider; the Ranger's budget is therefore available only when its
  label ran, not at mission start.
- **DEC-003. First consumer.** The Ranger-to-Archivist pre-check is the first authorized consumer
  (answering the original open question about the delegation owner). The archivist and gate
  pre-checks are not behind the budget and need their own demand.
- **DEC-004. Ledger placement.** Budget events live in a separate, isolated, append-only ledger
  (default `memory-effort/budget-events.jsonl`), not in the role-level ledger, so the role ledger and
  its readers are unaffected (answering the second open question).
- **DEC-005. Escalation semantics.** Escalation creates a new linked envelope (parent resolution id,
  level, reason) and never mutates the parent. It raises delegations and depth only; it does not
  re-resolve the model or effort tier through policy. It is bounded by `max_escalations` and an
  explicit allow flag, and a denied or exhausted escalation is a distinct, explicit result.
- **DEC-006. Integration closure.** `leveling label --reason escalated` calls `EscalateBudget`,
  carries a deterministic base resolution id, records the linked child in the isolated budget ledger,
  and exposes budget availability in delegation output. The directed verification target is
  provider-free and uses an isolated cache.

## Consequences

- A missing budget is visible and audited instead of silently permissive; operators can tell a
  missing label from an unreachable provider once the delegation line shows the budget state.
- The envelope, ledger and telemetry are additive: legacy label output, role ledger records and
  effort-resolution artifacts keep their shape.
- Escalation is reachable from the label command and remains bounded by the envelope's policy; the
  parent record is unchanged and the child/event linkage is auditable.
- The verification target listed in the implementation status passed on 2026-10-10: 908 directed
  tests across seven package invocations, with `GOPROXY=off` and no live provider call.
- Provider-side enforcement cannot be proved locally; `provider_usage` stays `unverified`.

## Non-goals

- Concurrent delegation, subprocess pools, automatic provider switching or model replacement.
- Mutation of a prior resolution by INITIATIVE, a confidence check or a delegated Role.
- Any change to the Approval Gate or the handoff lifecycle outcomes.

## Open Questions

- Whether the archivist and gate pre-checks should consume budget (separate demand).
- Whether escalation should later re-resolve the effort tier through policy (needs a policy decision).
- The implementation is staged in the working tree and not yet committed; recorded in the status note
  of the original package.
