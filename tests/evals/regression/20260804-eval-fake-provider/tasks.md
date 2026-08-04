---
mission_id: 20260804-eval-fake-provider
date: 2026-08-04
analysis_artifact_path: .analysis/refined/20260804-eval-fake-provider/analysis.md
proposal_path: .analysis/refined/20260804-eval-fake-provider/proposal.md
design_path: .analysis/refined/20260804-eval-fake-provider/design.md
tasks_path: .analysis/refined/20260804-eval-fake-provider/tasks.md
---

# Tasks — `internal/eval` Phase 2 (Fixture-Based Content Assertions)

## Approved Scope

```yaml
approved_scope:
  allowed:
    - internal/eval/**
    - tests/evals/**
    - tests/evals/fixtures/**
    - .analysis/archived/20260804-eval-fake-provider-adr.md   # only if approved as OA-ADR side quest
  forbidden:
    - internal/domain/**
    - internal/embed/**
    - cmd/**
    - tests/spec/**
    - tests/integration/**
    - "**"
```

DEC-2 (see `analysis.md`) makes this scope achievable without touching
`internal/domain/**` — same shape as `20260804-test-framework-v2` Phase 1's
own scope decision, for the same reason (no domain interface needed).

## Implementation Plan

All items mutate Go source or test files — none are documentation
materialization. Per the Scope Invariant, this mission analyzes and refines
this work; it does not execute it. Every numbered item below is
`implementation_handoff` except T1, which records this analysis itself and
is `analysis_artifact`. **None are `documentation_target`** in the base
plan — implementation proceeds through a separately authorized execution
provider outside this mission, exactly as `20260804-test-framework-v2`'s
T1–T11 did.

| ID | Objective | Scope | Validation | task_type | Status |
|----|-----------|-------|------------|-----------|--------|
| T1 | Record this refined package as the mission reference | `.analysis/refined/20260804-eval-fake-provider/` | files exist and are non-empty | analysis_artifact | done (this package) |
| T2 | Implement the seven content-assertion types (`contains`, `not-contains`, `regex`, `required-sections`, `source-citations`, `max-tokens`, `forbidden-tool-call`) as an extension of `TargetArtifactCheck`'s evaluation path | `internal/eval/content_assert.go` (new file, split out per the 200-line budget precedent), `internal/eval/artifact_check.go` (wired in), `internal/eval/assertion.go` (new constants) | `go build ./internal/eval/...` | implementation_handoff | ✅ done (2026-08-04, separately authorized implementation request) |
| T3 | Add hand-authored golden fixtures under `tests/evals/fixtures/` covering D1/D4's real-artifact-output comparison cases | `tests/evals/fixtures/ranger-analysis-example.md`, `tests/evals/fixtures/closure-completion-report-example.md` | fixtures are valid, non-empty, and match the format `contains`/`required-sections` checks expect | implementation_handoff | ✅ done |
| T4 | Add D1/D4 scenario test files exercising T2's new assertion types against T3's fixtures | `tests/evals/contracts/ranger_artifact_shape_valid_test.go`, `tests/evals/contracts/critical_hit_closure_report_shape_valid_test.go` | `go test -race -tags=eval ./tests/evals/...` passes the new cases | implementation_handoff | ✅ done |

**Implementation note (2026-08-04):** T2–T4 were implemented in a separate,
explicitly-authorized turn outside this Strategist mission — per the Mission
Boundary Clause, this mission's own gate acceptance did not authorize it;
the user made a new, direct implementation request afterward ("implementar
demanda: .analysis/refined/20260804-eval-fake-provider"). D1 and D4 are
fixture-shape validations, not live path-placement/move checks — see the new
test files' doc comments for why (Ranger and Critical Hit are agent-embodied,
not Go-callable). `go build ./...`, `go vet -tags=eval ./tests/evals/...`,
and `go test -race -tags=eval ./tests/evals/...` all pass; the full
`go test ./internal/...` suite is unaffected.
| T5 | Draft ADR recording DEC-2 (no provider interface; extend TargetArtifactCheck instead) and DEC-3 (SQ-003 stays separate), with rejected alternatives | `.analysis/archived/20260804-eval-fake-provider-adr.md` | file exists, non-empty, contains Context/Decision/Consequences sections and required frontmatter fields | documentation_target | accepted at gate (2026-08-04) — materializing |

Explicitly **not** in this plan: anything requiring `SQ-002` (harvested
mission fixtures) — those B1–B3 sub-cases stay blocked until `SQ-002` is
separately scoped and produces fixture output T2's assertions can read. Also
not in this plan: Groups E/F/G and live B1–B3 judgment — see `proposal.md`
"What Remains Blocked."

## Acceptance Checks

- `go build ./internal/eval/...` passes (no new `internal/domain` import)
- `go test -tags=eval ./tests/evals/...` passes all existing Phase 1
  scenarios plus the new D1/D4 cases from T4
- No file under `internal/domain/**` is modified
- Existing `tests/spec/` and `tests/integration/` suites unaffected

## Side Quests

No new side quests were opened by this refinement — the three related
backlog cards below already exist from the parent mission and are cited for
context, not modified.

| ID | Description | Relation | Status |
|----|-------------|----------|--------|
| SQ-002 | Harvest regression fixtures from real missions | Prerequisite for the B1–B3 recorded-decision sub-cases this mission's T2 assertions would consume — not scoped by this mission | sq_backlog (existing card, unchanged) |
| SQ-003 | LM Studio local-LLM integration | Confirmed (DEC-3) to depend on external live-invocation tooling, not on this mission's output — stays a separate future mission | sq_backlog (existing card, unchanged) |
| SQ-004 | Promptfoo CI adapter | Confirmed as the likely home for Groups E/F/G and live B1–B3 judgment, per `proposal.md` — not scoped by this mission | sq_backlog (existing card, unchanged) |
| OA-ADR-20260804-eval-fake-provider | Opportunity Attack candidate: `analysis.md` DEC-2 and DEC-3 are documented architectural tradeoffs with explicitly rejected alternatives | Accepted at gate (2026-08-04) — promoted to T5 above | accepted |

## Stop Conditions

- `scope_drift` — if any change touches `tests/spec/`, `tests/integration/`,
  or `internal/domain/` without explicit approval
- `validation_failure` — if `go build ./...` or `make test-all` fails after
  T2–T4
- `ambiguity` — if `internal/eval/artifact_check.go`'s current shape has
  changed since 2026-08-04 in a way that invalidates T2's extension point
  (re-verify at implementation time)
