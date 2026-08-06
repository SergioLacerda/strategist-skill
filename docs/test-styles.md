# Test Styles

**Status:** Accepted
**Last Updated:** 2026-08-04

This repository runs six distinct test styles, each behind its own `make`
target and (for the Go ones) its own build tag. Coverage — a *measured,
gated* metric — currently exists for only two of them. This doc names the
taxonomy explicitly so the gap is visible instead of implicit.

| Style | `make` target | Validates | Coverage today |
|---|---|---|---|
| unit | `test` | package-level Go logic (`go test -race ./...`, excludes `/testutil`) | none gated at this target; see `cover`/`cover-gate` below |
| unit (gated subset) | `cover` / `cover-gate` | line coverage for the packages listed in `scripts/coverage-packages.tsv` (widened to the full `./internal/...` tree + `cmd/strategist`) | line coverage %, 90% minimum for packages already at/above baseline; 4 packages (`internal/eval`, `internal/integrity`, `internal/runtimefs`, `internal/treasure`) carry an explicit, lower interim minimum with a `riposte-backlog.md` follow-up reference (see `SQ-005`) instead of silently omitting them |
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
- `ci-test: test-all convergence-check contract-consistency-gate cover-gate`
- `ci-web: install-web lint-web test-web cover-web build-site`
- `.github/workflows/test.yml`'s `test` job runs `make ci-test` and then
  publishes `make test-report`'s unified per-style table to
  `$GITHUB_STEP_SUMMARY` (non-blocking, `if: always()`) — in addition to
  each style's own pass/fail job result.

## Known gaps (not covered by this document alone)

- `internal/eval`, `internal/integrity`, `internal/runtimefs`, and
  `internal/treasure` are gated below the 90% baseline (see the `cover`
  row above) — raising them is tracked as `riposte-backlog.md` `SQ-005`,
  not done here.
- No internal/eval `Target` yet models Critical Hit's own trigger/closure
  logic the way `chest_grade`/`jewel_trust` now model treasure-chest
  grading — no pure Go function currently encapsulates that decision to
  dispatch to. Tracked as `riposte-backlog.md` `SQ-004`.

See `.analysis/refined/20260805-test-coverage-mapping-and-offline-eval/`
and `docs/test-coverage-gaps.md` for the mission that closed the previous
round of gaps here (unified report publication, gated-scope widening,
`cover-web` wired into `ci-web`, and new eval/spec trait scenarios).
