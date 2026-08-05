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
| unit (gated subset) | `cover` / `cover-gate` | line coverage for the 6 packages listed in `scripts/coverage-packages.tsv` | line coverage %, 90% minimum, enforced by `cover-gate` |
| spec (Gherkin) | `spec` | governance/contract behavior, driven by 15 `.feature` files under `tests/spec/specs/` (Given/When/Then scenarios consumed by Go test helpers in `tests/spec/*_test.go` — not a Cucumber/Godog runner) | none |
| integration | `integration` | cross-component Go behavior (`go test -race -tags=integration ./tests/integration/...`) | none as its own view (it is folded in as a coverage *source* for `cover-html`, but not reported as its own number) |
| eval | `eval` | prompt/artifact scenario correctness (`go test -race -tags=eval ./tests/evals/...`, 14 files) | none |
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
- `ci-web: install-web lint-web test-web build-site` — note `cover-web` is
  standalone, not part of `ci-web`
- `.github/workflows/test.yml` runs every style above except
  `eval-promptfoo` (intentionally excluded) as separate CI jobs, reported as
  job pass/fail — not as a per-style coverage breakdown.

## Known gaps (not covered by this document alone)

- A unified per-style report/table (proposed design:
  `.analysis/refined/20260804-test-coverage-visibility-by-style/design.md`)
  does not exist yet — implementing it is tracked as separate,
  not-yet-authorized work.
- `cover`'s 90%-gated scope is 6 packages out of the full `./internal/...`
  tree.
- `cover-web` is not measured in CI, only locally.
