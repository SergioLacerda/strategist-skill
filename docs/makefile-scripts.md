# Makefile Scripts Inventory

The root `Makefile` sets shared variables and `include`s domain files under
`make/*.mk`; non-trivial shell logic lives in standalone scripts under
`scripts/`. This doc lists every script, its purpose, and which `make`
target calls it, so the three don't drift out of sync silently.

Source mission: `.analysis/refined/20260804-makefile-script-decomposition/`
(analysis/design); implemented as a coding task outside Strategist mode
(T2–T6, T8 are `implementation_handoff` — Strategist's default execution
provider never mutates `Makefile` or creates shell scripts itself).

## Makefile Layout

| File | Domain |
|---|---|
| `Makefile` | shared variables, `include` directives, cross-domain composites (`ci-lint`, `ci-test`, `ci`) |
| `make/go.mk` | fmt/vet/build/test family, `validate-fixtures` |
| `make/quality.mk` | lint, complexity/file-size reports, coverage, `test-report` |
| `make/governance.mk` | convergence/governance/docs/contract/analysis-structure gates |
| `make/release.mk` | goreleaser targets, install, clean |
| `make/web.mk` | `web/landing` (npm) targets |

## Scripts

| Script | Called by | Purpose |
|---|---|---|
| `scripts/check-quality-budgets.sh` | `quality-budget-gate` | Fails the build if any file/function exceeds the cognitive-complexity budgets declared in `scripts/quality-budgets.tsv` |
| `scripts/check-coverage-gate.sh` | `cover-gate` | Fails the build when a package's test coverage falls below its threshold in `scripts/coverage-packages.tsv` |
| `scripts/check-refined-structure.sh` | `analysis-structure-gate` | Validates the shape of `.analysis/refined/<mission_id>/` packages |
| `scripts/check-docs-governance.sh` | `docs-governance-gate` | Validates repository documentation governance rules |
| `scripts/check-contract-consistency.sh` | `contract-consistency-gate` | Validates Strategist contract files stay internally consistent |
| `scripts/check-release-artifacts.sh` | `check-release-artifacts` | Validates release build artifacts exist and are well-formed |
| `scripts/check-release-assets.sh` | `check-release-assets` | Validates published release assets against `dist/published.tsv` |
| `scripts/check-reproducible-build.sh` | `release-reproducible-check` | Validates that a release build is reproducible |
| `scripts/complexity-report.sh` | `complexity-report` | Lists functions over the cognitive-complexity threshold; threshold is `$(COMPLEXITY_THRESHOLD)` (default `15`), unified with `quality-budget-gate`'s source (previously hardcoded `7`) |
| `scripts/go-file-size-report.sh` | `go-file-size-report` | Lists Go source files over 200 lines |
| `scripts/coverage-per-package.sh` | `cover` | Prints per-package coverage summary (distinct from `cover-gate`'s pass/fail check) |
| `scripts/check-convergence.sh` | `convergence-check` | Runs the four repository drift assertions (provider paths, retired `strategist/` mirror, etc.) |
| `scripts/check-governance-redirectors.sh` | `governance-check` | Validates `CLAUDE.md`/`AGENTS.md`/`GEMINI.md` carry the required governance fingerprint and redirector reference |
| `scripts/test-style-report.sh` | `test-report` | Prints one status row per test style (unit, spec, integration, eval, eval-promptfoo, web) — from the separate `20260804-test-coverage-visibility-by-style` mission |

## Notes for Future Extractors

- `cmd/strategist/makefile_contract_test.go`'s `copyMakefile` helper copies
  the root `Makefile`, every `make/*.mk` file, and
  `scripts/go-file-size-report.sh` into an isolated temp root to test that
  target in a synthetic tree. Any new target added to a `make/*.mk` file that
  gets its own isolated-fixture test needs the same treatment — `make`
  fails immediately on a missing `include` target, not just a missing
  script.
- `tests/spec/cicd_residual_contract_test.go` and
  `tests/spec/path_hygiene_test.go` check Makefile content via
  `readMakefileSystem` (`tests/spec/spec_helpers_test.go`), which
  concatenates the root `Makefile` with every `make/*.mk` file — content
  contracts pass regardless of which file within the include chain defines
  a given target/variable.
