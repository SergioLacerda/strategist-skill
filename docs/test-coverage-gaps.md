# Test Coverage Gaps — Implementation Handoff

**Status:** T2–T6 applied (verified 2026-08-06); T7 remains a future-mission candidate
**Source mission:** `.analysis/refined/20260805-test-coverage-mapping-and-offline-eval/`
**Related:** `docs/test-styles.md`, `docs/adr/0016-test-framework-v2.md`,
`docs/adr/0017-eval-fake-provider.md`, `.analysis/todo/riposte-backlog.md`

This doc is a single starting point for implementing the test-coverage/offline
trait-verification gaps identified in the source mission above. It intentionally
does not repeat `docs/test-styles.md`'s taxonomy or the two prior missions'
own analysis — read those first for background. Every item below was
`implementation_handoff`: source, CI-config, or test-file mutation that
Strategist's default Sniper contract does not perform. T2–T6 were applied
manually outside Strategist; each item's "How" section below still documents
the applied approach for reference.

Do not introduce a `FakeProvider`/mock-LLM/`domain.SkillProvider`-shaped
interface while implementing any item below — that path was evaluated and
rejected twice on record (ADR-0016 DEC-1, ADR-0017 DEC-2/DEC-3). If an item
below seems to need one, stop and re-read those ADRs before proceeding.

| ID | Status | One-line objective |
|---|---|---|
| T2 | done — `.github/workflows/test.yml` publishes `make test-report` to `$GITHUB_STEP_SUMMARY` (non-blocking, `if: always()`) | Publish `make test-report`'s table to `$GITHUB_STEP_SUMMARY` |
| T3 | done — `scripts/coverage-packages.tsv` covers all of `./internal/...` (except `internal/testutil`, which has no test files); `make cover-gate` passes | Widen `cover-gate`'s 90% scope from 6 packages to `./internal/...` |
| T4 | done — `make/web.mk`'s `ci-web` target depends on `cover-web` | Wire `cover-web` into `ci-web` |
| T5 | done — `tests/evals/scenarios/treasure_chest_grading_test.go` + `internal/eval` dispatch Target; `go test -race -tags=eval ./tests/evals/...` passes (15/15 scenarios) | New `internal/eval` Target for treasure-chest grading functions |
| T6 | done — `tests/spec/specs/e2e-critical-hit-closure.feature` + Go helper; `go test -race -tags=spec ./tests/spec/...` passes (159/159 scenarios) | New Gherkin feature for Critical Hit plain-move vs closure-move |
| T7 | not_started (future mission) | Extract a pure Critical Hit trigger/closure function |

---

## T2 — CI job summary for `make test-report`

**Why:** `.github/workflows/test.yml` runs each test style as a separate
pass/fail job; nobody sees the unified per-style table
(`scripts/test-style-report.sh` / `make test-report`) unless they run it
locally. `riposte-backlog.md` SQ-001.

**Where:** `.github/workflows/test.yml`.

**How:**
1. Add a job (or a final step on an existing aggregating job) that runs
   `make test-report` after the individual style jobs.
2. Redirect its stdout into `$GITHUB_STEP_SUMMARY` (GitHub Actions renders
   Markdown written there directly into the run summary UI) — e.g. wrap the
   script's row output in a Markdown table/code block before appending.
3. Keep this job non-blocking with respect to the existing per-style
   pass/fail jobs — it is a visibility addition, not a new gate.

**Validation:** a CI run's "Summary" tab shows the same 6 rows
`scripts/test-style-report.sh` prints locally (unit/spec/integration/eval/
eval-promptfoo/web), with matching STATUS values.

**Stop condition:** if adding this step would require installing new
dependencies beyond what `test-style-report.sh` already assumes (Go
toolchain, `web/landing` node_modules), stop and report — that's scope
creep beyond "publish what already runs."

---

## T3 — Widen `cover-gate`'s package scope

**Why:** Only 6 packages (`scripts/coverage-packages.tsv`) are gated at 90%
line coverage; the rest of `./internal/...` has coverage measured (via
`make test`) but never enforced. `riposte-backlog.md` SQ-002.

**Where:** `scripts/coverage-packages.tsv`, `Makefile` (`cover`, `cover-gate`
targets).

**How:**
1. Run `go list ./internal/...` to get the full package list.
2. For each package not already in `coverage-packages.tsv`, run
   `go tool cover -func` against it (same mechanism `test-style-report.sh`
   already uses for the `unit` row) to get its current line coverage.
3. Any package already ≥90%: add it to `coverage-packages.tsv` with
   `minimum=90` and a short `reason` (matching the existing file's
   3-column format: package, minimum, reason).
4. Any package below 90%: **do not silently lower the bar for the whole
   file.** Either raise its coverage first in the same change, or add it
   with an explicitly lower, reasoned minimum and flag it for a follow-up —
   never omit it from the file to dodge the question.

**Validation:** `make cover-gate` runs against every `./internal/...`
package (not just the original 6) and still exits 0.

**Stop condition (ambiguity):** if a package is meaningfully below 90% and
raising it is a nontrivial effort on its own, stop and report rather than
picking an arbitrary lowered threshold — that decision belongs to whoever
owns that package, not to this change.

---

## T4 — Wire `cover-web` into `ci-web`

**Why:** `web/landing`'s Vitest coverage (100% today) is only ever measured
locally (`make cover-web`); `ci-web: install-web lint-web test-web
build-site` doesn't include it. `riposte-backlog.md` SQ-003.

**Where:** `Makefile` (`ci-web` target).

**How:** add `cover-web` as a step in the `ci-web` target's dependency
chain (or as an explicit step in the CI job that calls `ci-web`), positioned
after `install-web` (needs `node_modules`) and before/alongside `test-web`.

**Validation:** `make ci-web` includes a `cover-web` run; a regression below
the current 100% baseline fails the build (match whatever failure behavior
`cover-web` already has locally — do not introduce a new coverage-gating
mechanism for this one style only).

---

## T5 — Go-native eval scenarios for treasure-chest grading

**Why:** `internal/domain/chest_grade.go`, `jewel_grade.go`, and
`potion_grade.go` already export pure validation functions
(`ValidateChestGrade`, `ValidateJewelKind`/`Score`/`Status`/`Trust`,
`ValidatePotionStatus`/`Trust`) with zero `internal/eval` scenario coverage
today. The only existing trait scenario file,
`tests/evals/scenarios/treasure_chest_scope_filter_test.go`, covers chest
*consultation scope filtering* (the `scope_filter` Target), not grading.

**Where:** `internal/eval/scenario.go` (new `Target` constant(s)),
`internal/eval/harness.go` (new `run*Scenario` dispatch case, following the
existing `runSlotWriteScopeScenario`/`runRouteDecisionScenario` shape),
`tests/evals/scenarios/` (new scenario file(s), same package/build-tag
pattern as `treasure_chest_scope_filter_test.go`).

**How (mirror the existing 5-Target pattern exactly):**
1. Add e.g. `TargetChestGrade Target = "chest_grade"` (and, if warranted,
   a separate `TargetJewelGrade`/`TargetPotionGrade` — decide based on
   whether one `Input.Params` shape can reasonably cover all three
   validators, or whether that overloads `Expected`'s existing fields).
2. Add `runChestGradeScenario(s Scenario, res *ScenarioResult)` in
   `harness.go`, calling `domain.ValidateChestGrade(...)` (or the Jewel/
   Potion equivalents) and translating the returned `error` into the
   existing `Status`/`Reason` `Expected` fields via `checkStatusAndReason`
   (already generic — reuse it, don't duplicate it).
3. Add scenario case(s) in `tests/evals/scenarios/` covering at least: a
   valid grade, an invalid trust-tier mismatch (`ValidateJewelTrust`), and
   an invalid status transition (`ValidateJewelStatus`/`ValidatePotionStatus`).

**Validation:** `go test -race -tags=eval ./tests/evals/...` includes and
passes the new file(s).

**Stop condition:** do not modify `internal/domain/*_grade.go` themselves —
this task only adds an `internal/eval`-side dispatch Target to functions
that already exist and are already exported.

---

## T6 — Gherkin feature for Critical Hit plain-move vs closure-move

**Why:** `tests/spec/critical_hit_closure_test.go` verifies that specific
narrative/machine contract files *contain* required phrases about closure
moves — useful, but it's a contract-text check, not a Given/When/Then
scenario at the Gherkin granularity the other 15 `.feature` files use.
`tests/spec/specs/e2e-treasure-chests.feature` is the closest sibling
pattern to follow, including its "Scope note" convention that explicitly
states what's in vs out of scope for the feature.

**Where:** new `tests/spec/specs/e2e-critical-hit-closure.feature`, plus a
Go test helper in `tests/spec/` (build tag `spec`) that parses and asserts
against it, following the existing helpers' style (not a Cucumber/Godog
runner — see `docs/test-styles.md` KF-04 in the source mission's
`analysis.md`).

**How:**
1. Write scenarios distinguishing: (a) plain move (no evaluation, no
   evidence required) vs (b) closure move (requires an explicit
   completion/validation claim + supplied evidence summary) — mirror the
   distinction already documented in `00-routing.md` § Routes and
   `11-critical-hit.md`.
2. Add a "Scope note" at the top of the feature file stating it covers the
   plain-move/closure-move *distinction*, and explicitly does not
   duplicate `critical_hit_closure_test.go`'s contract-text assertions or
   `tests/evals/contracts/critical_hit_closure_report_shape_valid_test.go`'s
   shape checks.
3. Wire the new feature file into whatever mechanism the existing 15
   `.feature` files use to reach their Go helpers (check an existing
   helper, e.g. the one backing `e2e-treasure-chests.feature`, for the
   loading convention before writing a new one from scratch).

**Validation:** `go test -race -tags=spec ./tests/spec/...` includes and
passes the new feature/helper pair; `make spec`'s scenario count in
`make test-report`'s output increases accordingly (was 156/156).

---

## T7 — (future mission, not started here) Critical Hit pure-function extraction

**Why:** Unlike treasure-chest grading (T5), Critical Hit's trigger/closure
logic has no equivalent pure Go function an `internal/eval` Target could
dispatch to — it's currently expressed only as contract prose
(`critical-hit.yaml` trigger_conditions), read and verified via string
matching. Building a Go-native *scenario* Target for Critical Hit the way
T5 does for treasure chests requires first extracting a pure function
(shaped like `domain.ValidateRouteDecision`) that encapsulates "is this a
valid plain-move / closure-move" — a change to `internal/domain`, which is
under stricter scope discipline than `internal/eval`.

**Why not scoped here:** this mission's approved Focus Area 2 was
"scenario/spec additions" (T5/T6), not "refactor `internal/domain`."
Designing the extraction is real work — deciding the function's signature,
what state it needs (mission evidence pack? tasks.md contents?), and how it
relates to `EvaluatePipelineBypass`'s existing style in
`internal/domain/pipeline_bypass.go` — and deserves its own discovery pass,
not a bullet point folded into this doc.

**Suggested next step:** scope a dedicated mission (tracked as `SQ-004` in
this mission's `tasks.md`) whose Ranger discovery reads
`internal/domain/pipeline_bypass.go` and `11-critical-hit.md` together to
propose that function's shape, before any code is written.
