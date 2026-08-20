# ADR-0028 — Native Roles as the Resilient Baseline

**Status:** Accepted
**Date:** 2026-08-19
**Context:** `20260819-strategist-improvements`

## Context

Strategist supports configurable external providers, but a valid capability descriptor does not prove that a provider is invocable in the active agent runtime. A refinement mission demonstrated this gap when `openspec-explore` was present as metadata but unavailable for invocation. The mission could resume only after the user explicitly selected the native Archivist.

ADR-0027 records the narrower, mission-scoped precedent of using Archivist as a native role for that light-client evaluation. This ADR establishes the broader architecture: compatible native roles are the resilient baseline for every Strategist phase, while external providers remain optional accelerators.

## Decision

Treat compatible native roles as the resilient baseline for Strategist phases. External providers remain optional accelerators selected by configuration.

Provider failure must never trigger silent substitution. Resolution uses an explicit policy:

- `block` — preserve strict failure behavior;
- `ask` — request explicit confirmation before using the native role; this is the recommended default;
- `native` — use the compatible native role automatically while emitting degradation evidence.

Every path must preserve slot compatibility, role contracts, write scope, local execution policy, handoff requirements, and the Strategist Approval Gate.

## Consequences

### Positive

- A missing external provider no longer has to make the whole pipeline unavailable.
- Self-contained installations can operate with native roles only.
- External-provider configuration errors remain visible and auditable.
- The role model becomes consistent across Scout, Ranger, Archivist, and Sniper.

### Negative

- `ask` adds an interaction when a provider fails.
- Supporting native and external paths increases resolution and test coverage requirements.
- Automatic `native` policy can conceal a degraded integration unless telemetry remains prominent.

## Rejected Alternatives

- **Always block:** preserves strictness but repeats the observed availability failure even when a compatible native role exists.
- **Always fall back silently:** improves continuity but hides configuration problems and weakens governance.
- **Remove external providers:** simplifies runtime behavior but discards legitimate specialization and extensibility.

## Validation Requirements

- Cover `block`, `ask`, and `native` with unit and integration tests.
- Reject fallback when slot, risk, or write contracts are incompatible.
- Emit configured provider, effective provider, reason, and degraded state.
- Prove that fallback never implies Approval Gate acceptance.

## Scope Boundary

This ADR records the architectural decision only. It does not authorize runtime, configuration, source-code, or test changes.
