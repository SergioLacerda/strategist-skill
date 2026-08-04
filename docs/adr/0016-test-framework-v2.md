# ADR-0016 — internal/eval Phase 1 Scope: Deterministic Domain Surface, Not FakeProvider

**Status:** Accepted  
**Date:** 2026-08-04  
**Context:** `20260804-test-framework-v2`

---

## Context

The pre-existing draft for a behavioral test harness (`.analysis/todo/v2/tests/`,
2026-07-28) specified `internal/eval`'s `FakeProvider` as implementing a
`domain.SkillProvider` interface, scripting deterministic responses for a
provider-invocation call boundary of the shape `Complete(ctx, req)
(ProviderResponse, error)`.

Refinement of this draft (mission `20260804-test-framework-v2`) found two
things true simultaneously:

1. No `domain.SkillProvider` interface, or any interface by that name,
   exists anywhere in `internal/domain` — `internal/domain/ports.go` defines
   only `Compiler`, `StaleChecker`, `Installer`, and `FileExtractor`.
2. Strategist's actual slot-invocation model is agent-embodied or
   skill-delegated over natural language (the parent agent embodies a
   native role such as Ranger/Sniper, or invokes an external skill by name
   such as `openspec-explore` for Archivist) — there is no in-process Go
   call boundary of the `Complete(ctx, req)` shape to fake in the first
   place.

The original draft's own `tasks.md` had already anticipated this as a named
stop condition ("`ambiguity` — if `internal/domain/ports.go` lacks a
suitable provider interface to attach `FakeProvider` to") but had not
resolved it. Separately, `internal/domain/**` is explicitly forbidden scope
for this mission, which rules out simply adding the missing interface as
part of this work.

Meanwhile, `internal/domain/state_machine.go` (FSM/gate/execution
transitions), `internal/domain/route_decision.go`
(`ValidateRouteDecision`), and `internal/treasure/status.go`
(`FilterRowsByScope`) already exist as plain, exported, already-tested Go
functions that cover a meaningful subset of the originally proposed 25
scenarios (approval/governance, route-decision policy, treasure-chest scope
filtering) without needing any new abstraction.

## Decision

Phase 1 of `internal/eval` targets only the existing deterministic Go
surface (`state_machine.go`, `route_decision.go`,
`treasure.FilterRowsByScope`) via a scenario/harness format that calls these
functions directly, in-process. No `FakeProvider`, `FixtureProvider`, or new
`internal/domain` interface is introduced in Phase 1.

This reduces Phase 1's scenario coverage from the originally proposed 25 to
9 (Groups A, C, part of B, part of D). The remaining 16 scenarios — the ones
that require scripting actual LLM-authored response content (adversarial
injection, token-budget, malformed-output, golden prompt/artifact
comparison, and Scout's own routing *judgment* as opposed to validating a
route decision already produced) — are deferred to a new backlog side
quest, `SQ-005`, gated on a real provider-invocation interface existing in
`internal/domain` for reasons independent of this test harness.

Alternatives considered and rejected:

- **Fake at a different, already-existing boundary** (e.g. a captured
  agent transcript/tool-call log). Rejected: no such boundary currently
  exists either; introducing one would still require new `internal/domain`
  surface, which is forbidden scope for this mission.
- **Explicitly re-open `internal/domain/**` scope** to add a minimal
  provider interface now. Rejected: unnecessary given the narrower path
  above already delivers a genuinely useful Phase 1, and re-opening
  forbidden scope should not be the default move when a narrower path
  exists.

## Consequences

- Phase 1 ships real, running scenario coverage (9 cases) against
  already-tested domain/treasure functions, with zero new abstractions and
  zero risk to `internal/domain/**`'s forbidden-scope boundary.
- The LLM-facing evaluation layer (`FakeProvider`, golden prompt tests,
  adversarial injection tests) does not exist yet and is not scheduled —
  `SQ-005` has no target date and is blocked on an interface this mission
  does not create. Anyone picking up `SQ-005` later must first decide where
  a provider-invocation interface would live and why, independent of this
  test harness's needs.
- Group B's scenarios split in spirit: the sub-case that validates a route
  decision already produced (`ValidateRouteDecision` policy checks) ships in
  Phase 1; the sub-cases that would simulate Scout's own classification
  *judgment* (B1–B3) move to `SQ-005` since judgment cannot be exercised
  without something standing in for the agent's response.
- Future missions extending `internal/eval` should re-check whether a
  provider interface has since appeared in `internal/domain` for unrelated
  reasons before re-litigating this decision.
