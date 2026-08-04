# ADR-0017 — internal/eval Phase 2: Fixture-Based Content Assertions, Not FakeProvider

**Status:** Accepted  
**Date:** 2026-08-04  
**Context:** `20260804-eval-fake-provider`

---

## Context

`SQ-005`, deferred from `20260804-test-framework-v2`'s Phase 1, asked for a
`FakeProvider`/`FixtureProvider`/`GoldenAssert` layer implementing a
`domain.SkillProvider`-shaped interface, to unblock 16 scenarios (Groups
E/F/G, B1–B3, D1/D4) requiring scripted LLM-response content. ADR-0016
(`20260804-test-framework-v2`, DEC-1) established that no such interface
currently exists in `internal/domain` and routed Phase 1 around it, but left
open whether one *should* be built to unblock Phase 2.

Refinement of this mission (`20260804-eval-fake-provider`) re-examined that
open question directly. A repo-wide search for LLM-call code
(`http.Client`, `openai`, `anthropic`, `llm`, case-insensitive) across
`internal/` and `cmd/` found none. `internal/eval/harness.go`'s own doc
comment states explicitly: "No provider is invoked — the function under
test is called directly, in-process." Strategist's actual invocation model
has the LLM agent calling this repository's Go/CLI surface as a tool — never
the reverse. There is therefore no in-process Go call site, now or
foreseeably, that a `domain.SkillProvider`-style interface could ever
substitute for.

Separately, `internal/eval/assertion.go`'s `AssertionType` doc comment
already names seven content-inspection assertion kinds deferred alongside
`FakeProvider` (`contains`, `not-contains`, `regex`, `max-tokens`,
`forbidden-tool-call`, `required-sections`, `source-citations`) — none of
which require a call boundary, only *some* content to inspect. The existing
`TargetArtifactCheck` (`internal/eval/artifact_check.go`) already reads real
files from disk for this purpose.

## Decision

**DEC-2:** `internal/eval` Phase 2 does not introduce `FakeProvider`,
`FixtureProvider`, or any new `internal/domain` provider-invocation
interface. Instead, it extends `TargetArtifactCheck` to implement the seven
already-declared content-assertion types, evaluated against real fixture
content — hand-authored golden files, or artifacts harvested from completed
missions (`SQ-002`) — rather than a scripted mock response.

This unblocks D1/D4 fully, and the sub-cases of B1–B3 that validate a
*recorded* `route_decision` against policy. It leaves Group E (adversarial),
Group F (token-economy), Group G (failure-modes), and the live-judgment
portion of B1–B3 blocked — not for lack of an interface, but because
`internal/eval`'s in-process Go model has no mechanism to observe or replay
real agent behavior. That work belongs to external evaluation tooling
(`SQ-004`'s Promptfoo direction) or a live-harness extension, never to this
package.

**DEC-3:** `SQ-003` (LM Studio local-LLM integration) is not folded into
this mission. It shares `SQ-005`'s original premise (a provider-invocation
dependency), but DEC-2 redirects that premise: a local LLM backend is a real
thing to invoke, not something to fake behind a new Go interface. `SQ-003`'s
actual dependency is now the external-tooling direction, not this mission's
output. It remains a separate, future mission.

### Alternatives Considered and Rejected

- **Add a minimal `internal/domain` interface purely for test purposes**
  (the option ADR-0016 already rejected for Phase 1). Rejected again, on
  stronger grounds: this mission's evidence shows no present or foreseeable
  in-process caller exists for such an interface in this architecture — it
  would be dead abstraction from day one, not merely premature.
- **Leave `SQ-005` fully blocked with only a re-check condition.** Rejected
  as too conservative: the `TargetArtifactCheck`/`assertion.go` evidence
  shows a real, buildable, Go-native path exists for a meaningful subset of
  the deferred scenarios.
- **Fold `SQ-003` into this mission now.** Rejected: its real dependency
  (external live-invocation tooling) has not itself been scoped yet;
  bundling would speculatively couple two unscoped missions.

## Consequences

- `internal/eval` Phase 2 (T2–T4 in this mission's `tasks.md`) stays
  entirely within `internal/eval/**` and `tests/evals/**` — zero risk to
  `internal/domain/**`'s forbidden-scope boundary, and zero new abstraction
  to maintain.
- D1/D4 and part of B1–B3 become achievable; Groups E/F/G and live B1–B3
  judgment are now understood to be permanently out of `internal/eval`'s
  reach by construction, not merely unscheduled — future missions should
  route that work to external tooling (`SQ-004`) rather than revisit adding
  a domain interface for it.
- `SQ-002` (mission-harvesting) becomes an explicit prerequisite for the
  B1–B3 portion of this decision's benefit, and is recommended as the next
  mission to scope.
- `SQ-003` stays blocked, but on a corrected dependency: it now depends on
  the external live-invocation tooling direction, not on any interface this
  or a future `internal/eval` mission would produce.
- Future missions extending `internal/eval` should re-read this ADR and
  ADR-0016 together before re-litigating either decision, and should check
  whether Strategist's own execution model has changed (which would be the
  only thing that could revive the original `FakeProvider` premise).
