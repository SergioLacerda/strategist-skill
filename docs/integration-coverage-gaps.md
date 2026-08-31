# Integration-Style Coverage Gaps — Implementation Handoff

**Status:** Partially implemented outside Strategist (see Implementation Log)
**Last Updated:** 2026-08-06
**Source mission:** `.analysis/refined/20260805-integration-coverage-mapping/`
**Related:** `docs/test-coverage-gaps.md` (sibling doc, different metric axis —
see Metric Note below), `docs/test-styles.md`

## Implementation Log

C2 and C3 were written as planned below (`tests/integration/e2e_telemetry_commands_test.go`,
`tests/integration/e2e_treasure_chest_test.go`) and pass, but **on their own
they moved the `integration` line-coverage number by 0.0 points.** Root
cause, found while verifying: every scenario in `tests/integration/**` that
uses `runStrategistCLI` (including C2/C3, and the pre-existing install/
compile/check/validate/check-stale scenarios) runs the compiled `strategist`
binary as a **subprocess**. `go test -coverpkg=./internal/...` only
instruments code running inside the `go test` process itself — a subprocess
is invisible to it, regardless of what the subprocess executes.

The actual fix was infrastructure, not more test files: build the
`strategist` binary with `-cover -covermode=atomic`
(`tests/integration/e2e_harness_test.go`'s `buildStrategistBinary`), route
every subprocess run's coverage into a shared `GOCOVERDIR`
(`runStrategistCLI` now sets it; the directory is
`STRATEGIST_E2E_GOCOVERDIR`-driven), and have
`scripts/test-style-report.sh` merge that directory with
`go tool covdata textfmt`, filtered back down to `internal/...` lines
(the compiled binary is instrumented for the whole main module, not just
`internal/...`, since `go build -coverpkg=./internal/...` — excluding
package main — was found by direct experiment to silently write zero
coverage files on exit).

This retroactively lit up every subcommand already exercised via
`runStrategistCLI`, not just the two new files. **Result: 25.5% → 31.8%**,
verified via a clean `make test-report` run, `go test -race -tags=integration
./tests/integration/...` (full suite, all passing), and `go vet ./...`
(clean). C4 remains not recommended, per the original evaluation below.

**Open question (unresolved, do not pick unilaterally):** the source mission
asked whether the goal is "raise the `integration` row specifically" or
"raise overall `internal/...` coverage regardless of style." The answer
received was **"whichever infers better quality"** — i.e. prefer the option
that adds real end-to-end verification value over the option that only
moves the number. That answer is what ranks C2/C3 above C4 below; it does
not by itself resolve every future prioritization call in this space.

## Metric Note

`make test-report`'s `integration` row runs `go test -tags=integration
-coverpkg=./internal/... ./tests/integration/...` — the denominator is all
of `./internal/...`, the numerator is only what `tests/integration/**`
(build tag `integration`) exercises. This is a different measurement axis
from `docs/test-coverage-gaps.md`'s `unit`/`cover-gate` items (T2-T7),
which measure `go test ./...` without that tag. Raising `unit` coverage for
a package does **not** move this doc's number; only new tests under
`tests/integration/**` do. Conversely, the packages this doc targets
(`internal/telemetry`, `internal/domain`) are already at 95-96% under
`unit` — the gap here is specifically "integration style never calls the
commands that exercise this code," not "this code lacks tests."

Baseline at time of writing: **25.5%**. After C2/C3 + the GOCOVERDIR fix
(see Implementation Log above): **31.8%**.

| ID | Status | One-line objective |
|---|---|---|
| C2 | done | New `tests/integration/e2e_telemetry_commands_test.go`: `metrics`, `metrics-scout`, `handoff-verify`, `runbook select` happy-path scenarios |
| C3 | done | New `tests/integration/e2e_treasure_chest_test.go`: scan → doctor → items list/show happy-path scenario |
| C5 | done | Infra fix (not originally scoped): route `runStrategistCLI` subprocess coverage into the `integration` metric via GOCOVERDIR — see Implementation Log |
| C4 | not recommended | Integration-style duplicates of `internal/domain` FSM/grading logic already covered by `eval` style |

---

## C2 — Telemetry-heavy CLI integration scenarios

**Why:** `internal/telemetry` is 102/103 functions at 0.0% under the
`integration` coverprofile (356-line `go tool cover -func` breakdown,
mission `analysis.md`). Its mission-pipeline emission paths
(`mission_run.go`, `route_decision.go`, `route_metrics.go`,
`scout_metrics_history.go`, `handoff_metrics.go`,
`handoff_challenge_record.go`) only fire on commands that run a
mission-shaped flow — `metrics`, `metrics-scout`, `handoff-verify`,
`runbook select` — none of which any current `tests/integration/*.go` file
invokes (confirmed via `runStrategistCLI` call-site grep: only `install`,
`compile`, `check`, `validate`, `check-stale` appear today).

**Where:** new `tests/integration/e2e_telemetry_commands_test.go`
(`//go:build integration`), following `e2e_cli_happy_path_test.go`'s
`runStrategistCLI(t, workspace, "<subcommand>", ...)` pattern.

**How:**
1. Seed a fixture workspace (`t.TempDir()` + `install`/`compile` as
   preconditions, matching existing tests' setup) with whatever mission
   checkpoint/handoff artifact each subcommand expects. Check
   `cmd/strategist/metrics.go`, `metrics_scout.go`, `handoff_verify.go`,
   `handoff_verify_io.go`, `runbook_select.go` for the exact fixture shape
   each reads before writing new fixtures from scratch.
2. Add one happy-path scenario per subcommand: `metrics`, `metrics scout`,
   `handoff verify`, `runbook select`. Assert exit code 0 and at least one
   telemetry side effect specific to that command (e.g. a
   `.strategist/memory/*.jsonl` line).
3. Assert through the CLI's own observable output/artifacts, not
   `internal/telemetry` internals directly — same style
   `e2e_cli_happy_path_test.go` already uses for `compile`'s `.compiled/`
   outputs.

**Validation:** `go test -race -tags=integration ./tests/integration/...`
passes with the new file; a follow-up `make test-report` run shows the
`integration` row's line-coverage value increased above the 25.5%
baseline.

**Stop condition:** if a subcommand requires an interactive/manual
precondition (e.g. an LLM-backed step) that can't be fixture-seeded
deterministically, skip that subcommand and note it in the new test file's
comments rather than mocking a provider — do not introduce a
`FakeProvider`/mock-LLM-shaped interface (rejected on record, see
`docs/test-coverage-gaps.md` header note, ADR-0016/0017).

---

## C3 — Treasure-chest end-to-end scenario

**Why:** `internal/integrity` (`warning.go`) is 14/14 functions at 0.0%
under the `integration` coverprofile. Its only callers
(`cmd/strategist/root.go`, `treasure_chest_config_lock.go`) are otherwise
dark for this metric, and a treasure-chest CLI flow also reaches a slice of
`internal/domain` not already covered by the `eval` style.

**Where:** new `tests/integration/e2e_treasure_chest_test.go`.

**How (as implemented):**
1. `treasure-chest scan --dry-run` against a workspace with zero missions.
2. `treasure-chest doctor` against a workspace whose `active.yaml` declares
   a chest with no matching `treasure-chests.yaml`/`knowledge.index.yaml`
   entry — deliberately, to exercise the real divergence-detection path
   (`LoadActiveChests`/`LoadGoverned`/`LoadIndexed`/`MergeChestRows`) rather
   than a synthetic always-green fixture.
3. `treasure-chest items list` / `items show`, seeding `jewels.yaml`
   directly (same fixture shape as
   `internal/treasurecli/treasure_chest_items_test.go`'s `oneProposedJewelYAML`).

**Result:** `internal/integrity` went from 14/14 functions at 0.0% to 2/14
(`IsModified`, `pathMismatchResult` still unreached) — via `root.go`'s
own use of the package during every subcommand run, once the GOCOVERDIR fix
made subprocess coverage visible at all, not from a dedicated
deliberately-invalid-chest scenario (that idea from the original plan
wasn't needed in the end — worth knowing if `IsModified`/
`pathMismatchResult` become a target later).

---

## C5 — GOCOVERDIR subprocess coverage merging (infra fix, not originally scoped)

**Why:** see Implementation Log at the top of this doc — `runStrategistCLI`
subprocess runs were invisible to `go test -coverpkg=./internal/...`
regardless of what CLI surface they exercised, so C2/C3 alone measured
0.0 points of lift despite passing.

**Where:** `tests/integration/e2e_harness_test.go` (`buildStrategistBinary`,
`runStrategistCLI`, new `strategistGOCOVERDIR` helper),
`scripts/test-style-report.sh` (integration row).

**What changed:**
1. `buildStrategistBinary` now builds with `go build -cover
   -covermode=atomic` (mode must match what `go test -race` forces on the
   test binary's own instrumentation, or `go tool covdata` refuses to merge
   with a "counter mode clash" error).
2. `runStrategistCLI` sets `GOCOVERDIR` in the subprocess env to a shared
   directory (`strategistGOCOVERDIR`, driven by the
   `STRATEGIST_E2E_GOCOVERDIR` env var, falling back to a scratch temp dir
   for plain `go test` runs that don't set it).
3. `scripts/test-style-report.sh` sets `STRATEGIST_E2E_GOCOVERDIR`, passes
   `-args -test.gocoverdir=<dir>` to `go test` so the test binary's own
   in-process coverage lands in the same directory, then merges everything
   with `go tool covdata textfmt`, filtered to `internal/...` lines only
   (`go build -coverpkg=./internal/...` — excluding package main — was
   confirmed by direct experiment to silently write zero coverage files on
   exit; instrumenting the whole main module and filtering the merged
   profile afterward was the only path that actually worked).

**Both `STRATEGIST_E2E_GOCOVERDIR` and `-test.gocoverdir` must be absolute
paths** — the Go runtime resolves a relative `GOCOVERDIR` against the
subprocess's own working directory (`workspace`, a per-test tempdir), and
`-test.gocoverdir` against the test binary's own working directory (the
package directory, `tests/integration/`), neither of which match the
shell's cwd. `scripts/test-style-report.sh` normalizes `coverage_dir` to an
absolute path at the top of the script for this reason.

**Validation:** `make test-report` (`scripts/test-style-report.sh`) shows
`integration` at 31.8% (was 25.5%); `go test -race -tags=integration
./tests/integration/...` passes in full; `go vet ./...` clean.

---

## C4 — (not recommended) Integration-style duplicates of domain FSM/grading logic

**Why not:** `internal/domain`'s FSM (`state_machine.go`) and grading
functions (`chest_grade.go`/`jewel_grade.go`/`potion_grade.go`) are large
and dark under this specific metric, which makes them look like "free"
coverage lift. But they're already exercised by the `eval` style
(`tests/evals/scenarios/treasure_chest_grading_test.go`) and by `unit`
(95.1% for the whole package). Adding integration-tagged duplicates would
move the `integration` row's number without adding distinct verification
value — coverage-for-its-own-sake, not the "better quality" standard this
doc's open question was resolved against. If a genuine integration-level
gap in FSM/grading behavior is later identified (e.g. an end-to-end CLI
flow that exercises the FSM in a way `eval`/`unit` don't), scope that as
its own mission with its own evidence — don't fold it into C2/C3.
