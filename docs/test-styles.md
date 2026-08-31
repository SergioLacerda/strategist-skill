# Test Styles

**Status:** Accepted
**Last Updated:** 2026-08-30

This repository runs six distinct test styles, each behind its own `make`
target and (for the Go ones) its own build tag. Coverage — a *measured,
gated* metric — currently exists for only two of them. This doc names the
taxonomy explicitly so the gap is visible instead of implicit.

| Style | `make` target | Validates | Coverage today |
|---|---|---|---|
| unit | `test` | package-level Go logic (`go test -race ./...`, excludes `/testutil`) | none gated at this target; see `cover`/`cover-gate` below |
| unit (gated subset) | `cover` / `cover-gate` | line coverage for the packages listed in `scripts/coverage-packages.tsv` (widened to the full `./internal/...` tree + `cmd/strategist`) | line coverage %, 90% minimum baseline; `internal/eval`, `internal/integrity`, `internal/runtimefs` are gated at the 90% baseline and currently measure 94.6%/98.2%/100.0%, and `internal/treasure` is gated at a stricter 95% threshold and measures 95.5% — see `scripts/coverage-packages.tsv` for per-package provenance |
| spec (Gherkin) | `spec` | governance/contract behavior, driven by 16 `.feature` files under `tests/spec/specs/` (Given/When/Then scenarios consumed by Go test helpers in `tests/spec/*_test.go` — not a Cucumber/Godog runner) | none |
| integration | `integration` | cross-component Go behavior (`go test -race -tags=integration ./tests/integration/...`) | none as its own view (it is folded in as a coverage *source* for `cover-html`, but not reported as its own number) |
| eval | `eval` | prompt/artifact scenario correctness (`go test -race -tags=eval ./tests/evals/...`, 15 files across `contracts/` and `scenarios/`) | none |
| eval-promptfoo | `eval-promptfoo` | LLM-judged artifact quality (`npx promptfoo eval`) | none — deliberately standalone, not wired into `eval`/`test`/`test-all`/`ci-test`/`ci`; requires a local LM Studio endpoint. See `.analysis/archived/20260804-promptfoo-ci-adapter-adr.md`. |
| web | `test-web` / `cover-web` | Vitest suite in `web/landing` | line coverage % via `cover-web` (`npm run cover`), not gated, and not wired into `ci-web` |

## Why coverage isn't uniform across styles

Line-coverage percentage is the right metric for `unit`, `integration`, and
`web` — they exercise code paths directly. It is the wrong metric for
`spec` and `eval`: those suites validate scenarios/contracts (Given/When/Then
behavior, prompt-quality scenarios), where "% of scenarios passing" is the
meaningful signal, not "% of lines executed." Forcing a single blended
coverage number across all six styles would misrepresent the scenario-driven
suites rather than clarify them.

## Aggregators

- `test-all: test spec integration` — omits `eval` and `web`
- `ci-test: test-all golden convergence-check contract-consistency-gate cover-gate`
- `ci-web: install-web lint-web test-web cover-web build-site`
- `.github/workflows/test.yml`'s `test` job runs `make ci-test` and then
  publishes `make test-report`'s unified per-style table to
  `$GITHUB_STEP_SUMMARY` (non-blocking, `if: always()`) — in addition to
  each style's own pass/fail job result.

## Known gaps (not covered by this document alone)

- The two gaps previously tracked here (`internal/eval`/`internal/integrity`/
  `internal/runtimefs`/`internal/treasure` gated below the 90% baseline, and
  no pure Go `Target` modeling Critical Hit's trigger/closure logic) are
  closed: `scripts/coverage-packages.tsv` now gates all four packages at or
  above baseline (see the `cover` row above), and
  `internal/domain/critical_hit_trigger.go`'s `EvaluateCriticalHit` is that
  pure decision function (`riposte-backlog.md`'s `SQ-004`/`SQ-005` entries
  are gone — closed, not silently dropped).
- The golden-test suite (`tests/evals/golden/`, `//go:build golden`) is now
  wired into CI: `make golden` (`go test -race -tags=golden
  ./tests/evals/golden/...`) is a prerequisite of `ci-test`, which
  `.github/workflows/test.yml`'s `test` job already runs via `make ci-test` —
  no separate workflow step was needed. Closes ADR-0026's CI-wiring
  follow-up and `.analysis/refined/20260830-skill-gaps-triage/tasks.md`
  Task 6.

See `.analysis/refined/20260805-test-coverage-mapping-and-offline-eval/`
and `docs/test-coverage-gaps.md` for the mission that closed the previous
round of gaps here (unified report publication, gated-scope widening,
`cover-web` wired into `ci-web`, and new eval/spec trait scenarios).
