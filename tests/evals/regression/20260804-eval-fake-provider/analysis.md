---
mission_id: 20260804-eval-fake-provider
mission_status: documentation_applied
claimed_by: claude-code-session
claimed_by: openspec-explore-session
date: 2026-08-04
discovery_subtype: creative
analysis_artifact_path: .analysis/refined/20260804-eval-fake-provider/analysis.md
confidence_score: 0.8
---

# Analysis — `internal/eval` FakeProvider/FixtureProvider/GoldenAssert Layer (SQ-005)

## Mission Objective

Resolve the open architectural question left by `20260804-test-framework-v2-adr.md`
(DEC-1) and `SQ-005`'s own backlog card: whether, and why, a provider-invocation
interface should exist in `internal/domain` to unblock the 16 scenarios deferred
from Phase 1 of the `internal/eval` behavioral test harness (Groups E/F/G,
B1–B3, D1/D4).

## Known Facts (from Ranger discovery, unchanged)

- `internal/domain/ports.go` declares exactly four interfaces (`Compiler`,
  `StaleChecker`, `Installer`, `FileExtractor`) — none resembling a
  provider/LLM-invocation boundary.
- `internal/domain/slots.go` defines only `SlotName` configuration constants,
  not an invocation interface.
- `domain.SkillProvider`, which the 2026-07-28 draft specified `FakeProvider`
  to implement, does not exist anywhere in the codebase — confirmed
  independently twice (mission `20260804-test-framework-v2` and this one).
- Strategist's actual slot-invocation model is agent-embodied (native roles)
  or skill-delegated by name (external providers) — not an in-process Go
  call boundary.
- `SQ-003` (LM Studio integration) self-flags the same blocking dependency as
  `SQ-005`. `SQ-004` (Promptfoo) does not share it.
- 16 scenarios are gated on this decision: Groups E, F, G, B1–B3, D1/D4.

## Additional Findings (this refinement pass)

- `internal/eval/harness.go`'s `RunScenario` doc comment states directly:
  *"No provider is invoked — the function under test is called directly,
  in-process."* A repo-wide search (`http.Client`, `openai`, `anthropic`,
  `llm`, case-insensitive) across `internal/` and `cmd/` found no LLM-call
  code anywhere in this Go codebase. This is not an omission — Strategist's
  Go binary is a CLI invoked *by* an LLM-driven agent (via shell/tool calls
  reading `.strategist/` contracts); the agent is the caller, never the
  callee, of this repository's Go code.
- This inverts the usual "app calls provider" shape a `FakeProvider`
  interface assumes. There is no in-process call site anywhere in this
  codebase's control flow that a `domain.SkillProvider`-style interface
  could ever substitute for — this is a structural property of the
  architecture, not a temporary gap that "hasn't been built yet."
  `20260804-test-framework-v2-adr.md`'s DEC-1 established the interface
  doesn't currently exist; this finding establishes that an in-process
  version of it, as originally specified, has no caller to attach to,
  regardless of when it might be built.
- `internal/eval/assertion.go` already anticipates this split. Its
  `AssertionType` doc comment names seven content-inspection assertion
  kinds explicitly deferred alongside `FakeProvider` — `contains`,
  `not-contains`, `regex`, `max-tokens`, `forbidden-tool-call`,
  `required-sections`, `source-citations` — but nothing about these types
  requires a live or scripted call boundary: each just inspects *some*
  string content against a rule. That content can come from a saved
  fixture file exactly as `TargetArtifactCheck`
  (`internal/eval/artifact_check.go`) already reads real files from disk —
  no mock, no interface, no `internal/domain` change.
- `SQ-002`'s own framing ("harvest regression cases from real missions")
  is the natural fixture source for this: real `route_decision` records,
  analysis artifacts, or ADRs captured from completed missions, replayed
  through the existing deterministic validators
  (`domain.ValidateRouteDecision`, `TargetArtifactCheck`) plus the new
  content-assertion types above. This requires zero new `internal/domain`
  interface and zero `FakeProvider`.

## Decisions

### DEC-2: No provider-invocation interface — extend artifact/content assertion instead

**Status:** accepted (this mission)

**Decision:** Do not introduce `FakeProvider`, `FixtureProvider`, or any new
`internal/domain` provider-invocation interface, in Phase 2 or later. Instead,
extend the existing `TargetArtifactCheck` pattern and activate
`assertion.go`'s seven already-declared, currently-unused content-assertion
types to evaluate **real fixture content** — hand-authored golden files or
artifacts harvested from completed missions (`SQ-002`) — rather than
scripted mock responses from an interface.

**Evidence:** known facts above + additional findings (no LLM call site
exists anywhere in this Go codebase for a provider interface to attach to;
the existing `TargetArtifactCheck`/`assertion.go` scaffolding already points
at a content-based, not call-based, evaluation shape).

**Consequences / scope split of the 16 deferred scenarios:**

- **Unblocked by this decision** (Go-native, no `internal/domain` change,
  fixture-driven): D1/D4 (real-artifact-output comparison) fully; the
  sub-cases of B1–B3 that validate a *recorded* Scout `route_decision`
  against policy (as opposed to live judgment) — via harvested fixtures per
  `SQ-002`.
- **Still genuinely blocked, and for a different reason than DEC-1
  identified**: Group E (adversarial-content probing), Group F
  (token-economy), Group G (failure-modes tied to live agent recovery
  behavior), and the live-judgment portion of B1–B3 (does the agent
  *classify correctly when it hasn't yet*, not does a past classification
  match policy). These require actually invoking or replaying a real agent
  session — structurally outside what `internal/eval`'s Go-native,
  in-process model can ever do. This is not a resourcing gap; it is a
  category boundary. This work belongs to external evaluation tooling
  (`SQ-004`'s Promptfoo direction, which already invokes real
  providers/models) or a live-harness extension — never to this package.

**Alternatives rejected:**

- *Add a minimal `internal/domain` interface purely for test purposes*
  (the option DEC-1 already rejected for Phase 1). Rejected again here with
  stronger grounds: the additional-findings evidence shows there is no
  present or foreseeable in-process caller for such an interface in this
  architecture, so it would be dead abstraction from day one, not merely
  premature.
- *Leave `SQ-005` fully blocked with only a re-check condition* (the
  Ranger artifact's Option 3). Rejected as too conservative given the
  `TargetArtifactCheck`/`assertion.go` evidence shows a real, buildable,
  Go-native path exists for a meaningful subset of the deferred scenarios —
  declaring total blockage would understate what is actually achievable.

### DEC-3: `SQ-003` (LM Studio) stays a separate mission

**Status:** accepted (this mission)

**Decision:** `SQ-003` is not folded into this mission's scope. It shares
`SQ-005`'s original blocking premise (dependency on *some* provider
invocation interface) but DEC-2 redirects that premise: a local LLM backend
is a real thing to invoke, not something to fake behind a new Go interface.
`SQ-003`'s actual dependency, after this decision, is on the external
evaluation-tooling direction (`SQ-004`/Promptfoo-style harness capable of
invoking a real backend), not on anything this mission produces. Its scope
(local-LLM operational integration specifics) is independent enough from
`internal/eval`'s Go-native harness to remain its own mission.

**Alternatives rejected:** folding `SQ-003` in now — rejected because its
real dependency (external live-invocation tooling) has not itself been
scoped yet; bundling would speculatively couple two unscoped missions.

## Uncertainties Carried Forward

- Whether `SQ-002` (mission-harvesting) should be scoped before or after the
  `TargetArtifactCheck` content-assertion extension — both are
  prerequisites for unblocking D1/D4/B-partial, and neither is scoped by
  this mission. Recommend a future mission scope `SQ-002` first (fixture
  supply) since the assertion-type extension is meaningless without fixture
  content to check.
- Whether Groups E/F/G should ever be attempted at all, or permanently
  descoped from `internal/eval` in favor of a wholly external tool. This
  mission does not decide that — it only establishes that they cannot live
  in `internal/eval` as currently shaped.

## Affected Scope

Unchanged from Ranger's discovery pass — `internal/domain/**` (read-only;
DEC-2 explicitly avoids touching it), `internal/eval/**`, `tests/evals/**`,
plus the two related pending cards (`SQ-003`, and implicitly `SQ-002`/`SQ-004`
context) and the existing ADR.

## Side Quests

See `tasks.md` § Side Quests for disposition of `SQ-002`/`SQ-003`/`SQ-004`
cross-references surfaced during this refinement — none are new; all were
already open backlog cards from the parent mission.
