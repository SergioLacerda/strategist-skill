# Internal Package Layout

**Status:** waves 0-7 applied in the working tree on 2026-10-12. The per-wave status table at the end records what was done.
**Date:** 2026-10-12
**Provenance:** refined demand package `20261012-internal-affinity-move-proposal-r2` (`.analysis/refined/20261012-internal-affinity-move-proposal-r2/`), built on the preliminary inventory of 2026-10-12 and the earlier layout decisions in `20261003-internal-layout-refactor-reassessment`.

## Purpose

This page records how the Go packages under `internal/` are grouped by affinity, which packages are deliberately left where they are, and the rules that keep a move from changing behavior. It describes whole-package moves only. It does not describe splitting a package, renaming a symbol or redistributing responsibilities.

## Principles

- A move renames a package directory as a whole. The Go package clause and the directory basename stay unchanged.
- Import paths, path literals that name a moved package, manifests, CI lines, scripts, documentation and embedded contract text that cite the old path are updated in the same change as the move.
- No compatibility package, alias or forwarding shim is left behind.
- A directory is a group because its members share an owner, an import direction and a vocabulary, not because of a taxonomy label. Taxonomy classifies responsibility; it does not define a filesystem move boundary.
- A parent directory that holds a package of its own (for example `internal/telemetry`) may be imported by its children. A directory used as a pure namespace may not hold implementation files.

## Groups

Groups that already exist and are unchanged:

| Group | Kind | Members today |
|---|---|---|
| `internal/domain` | parent package with children | `domain`, `domain/installplan`, `domain/roster` |
| `internal/application` | parent package with children | `application`, `application/{authorization,digest,installplan,leveling,metrics,provider}` |
| `internal/feats` | pure namespace | `critical_hit`, `initiative`, `opportunity_attack`, `select_runbook` |
| `internal/tools` | pure namespace | `leveling`, `prepare_runbook`, `resolver` |
| `internal/roles` | pure namespace | `jeweler`, `registry`, `validation` |
| `internal/plugins` | parent package with children | `plugins`, `plugins/{acquisition,conformance,connectors,governance,lifecycle,policy,trust}` |
| `internal/integration` | parent package with children | `integration`, `integration/{config,credential,handoffconsumer,jev,live,mechanism,policy,probe,runbookconsumer,setup,transport}` |
| `internal/runtime` | namespace of adapters | `catalog`, `env`, `integrity`, `payload`, `validate` |
| `internal/governance` | parent package with children | `governance`, `governance/providence` |
| `internal/weapon` | parent package with children | `weapon`, `weapon/treasuretmp` |

Groups that change, by wave:

| Wave | Package | Target path | Reason for the affinity |
|---|---|---|---|
| 1 | `hostusage` | `internal/telemetry/hostusage` | observes host usage and normalizes it into telemetry; no internal imports |
| 1 | `usagesummary` | `internal/telemetry/usagesummary` | aggregates declared token observations; imports only `domain` and `telemetry` |
| 2 | `hardening` | `internal/runtime/hardening` | runtime hardening contracts; only `runtime/integrity` imports it |
| 3 | `bindingdrift` | `internal/mission/bindingdrift` | records the slot bindings a mission started with |
| 3 | `gatecriteria` | `internal/mission/gatecriteria` | approval and quality gate criteria; consumed by `mission/view` |
| 3 | `refinement` | `internal/mission/refinement` | OpenSpec normalization and publication for the mission lifecycle |
| 4 | `calibration` | `internal/dojo/calibration` | the DOJO judge calibration core; `dojo` already imports it |
| 5 (optional) | `cliutil`, `check` | `internal/cli/cliutil`, `internal/cli/check` | CLI command clusters extracted from `cmd/strategist` |
| 6 (optional) | `handoff` | `internal/mission/handoff` | role-to-role handoff records; the largest rewrite (85 importing files) |
| 7 (optional) | `roster` | `internal/application/roster` | small read-only planning facade over `domain/roster` |

## Frozen anchors

These are not moved by this plan. Each one is named by a rule, a manifest or a large number of importers, so a move would be a policy change and not a layout tidy.

| Package | Why it stays |
|---|---|
| `internal/domain` | 698 importing files; named by depguard, `TestDomainIsolation`, `TestDomainDoesNotDependOnRuntimeBoundaries`; tests locate the repository root with `../..` |
| `internal/embed` | `internal/embed/defaults` is hard-coded in `go:embed`, CLI flag defaults, scripts and many spec tests |
| `internal/telemetry` (parent) | 185 importing files; `make/go.mk` file lists and `scripts/generate-event-catalog.sh` name its files |
| `internal/install`, `internal/compile`, `internal/stale` | depguard globs, `TestLateralIsolation`, `runtime.Caller` and path constants in production code (`internal/install` keeps its path; its contents are split, see "Inside `internal/install`") |
| `internal/filesystem` | neutral leaf primitive with 26 importing files |
| `internal/testutil` | shared fixtures with 63 importing files |
| `internal/mission` (parent) | lifecycle and orchestration boundary; children are added, the parent is not moved |

## Inside `internal/install`

Mission `20261012-internal-install-refactor-evaluation` split the flat package into sub-packages by affinity. The root keeps the `Service`/`Installer` orchestration; every sub-package is a leaf of it, and the package graph is acyclic (nothing imports the root).

| Package | Owns |
|---|---|
| `names` | file, path, connector-identity and vocabulary names shared by the packages; imports nothing from `internal/install` |
| `support` | atomic writes, `.gitignore` entries, mission ids, small slice helpers |
| `plugincatalog` | plugin catalog model, lock file, option sets, weapon pin comparison, role/provider catalog mapping |
| `embeddedweapon` | embedded Weapon ingestion, certification, mirrors, lock and drift |
| `sidecar` | `strategist.yaml` sidecar scaffold |
| `customprovider` | Custom provider resolution, declaration, materialization |
| `rankedbinding` | Ranked binding activation and private runtime preparation |
| `onboarding` | plugin onboarding plan and role/provider migration |
| `activeyaml` | `active.yaml` and its knowledge and treasure-chest sections |
| `wizard` | interactive install Wizard, prompters and defaults |
| `installtest` | test fixtures shared by these packages; imported by tests only |
| root `install` | `Service`, `NewInstaller`, installer steps, upgrade, runtime defaults, shim, audit, manifest |

Dependency order: `names`, `support` -> `plugincatalog` -> `embeddedweapon`, `sidecar`, `customprovider` -> `rankedbinding`, `onboarding` -> `wizard` -> root; `activeyaml` depends only on `names` and `support`. Rules that govern the subtree: the `install-lateral` depguard rule (glob `**/internal/install/**`), `TestLateralIsolation`, and one `scripts/coverage-packages.tsv` row per package. Code that locates the repository from its own source path must account for the extra directory level (`embeddedweapon.RepoRoot`).

## Deferred

Not moved because they need a rename or a separate decision, not because of effort:

- `internal/provider` imports `install`; nesting it under `plugins/` would invert the direction.
- `internal/authorization`, `internal/conformance` and `internal/integrity` have basename collisions with `application/authorization`, `plugins/conformance` and `runtime/integrity`.
- `internal/governancebridge` is an intentionally narrow cycle-breaker; nesting it under `governance/` would imply the dependency it exists to avoid.
- `internal/i18n`, `internal/weapon`, `internal/mechanisms`, `internal/runbook` and `internal/eval` keep their current paths. `runbook` is the shared model that the namespace sibling rule forces outside `feats` and `tools`; `eval` appears in CLI help text and a golden file.

## Allowed dependency direction

The rules below are enforced by `internal/domain/architecture_test.go` and `internal/domain/layout_architecture_test.go` (integration build tag) and by depguard in `.golangci.yaml`.

- `internal/domain` is a pure contract layer: it imports no other internal package and no I/O or UI dependency.
- Business packages depend on `internal/domain`; `compile`, `install` and `stale` do not import each other or `embed`.
- `internal/application` and its children do not import `telemetry`, `provider`, `plugins`, `cmd/strategist` or Cobra.
- Children of `feats`, `tools` and `roles` do not import their siblings; shared contracts live in `domain` or a deliberate port package.
- `internal/pipelines/` is reserved in `layoutNamespaces` but has no owner. Mission packages are not placed there because they legitimately import each other.
- A rule that names a package by path passes without protecting anything once that package moves. Update the rule in the same change.

## Adding or moving a package

1. Pick the group whose members share the new package's owner and import direction; if none fits, keep it at the top level and say why.
2. Add or move the directory with its tests, `testdata` and the package clause unchanged.
3. Update, in the same change: the row in `scripts/coverage-packages.tsv` (or `scripts/coverage-exemptions.tsv`), any entry in `scripts/quality-budgets.tsv`, depguard globs in `.golangci.yaml`, the architecture test lists, the CI package list in `.github/workflows/test.yml`, and the package scan list in `internal/integration/hermetic_test.go`.
4. Check depth-relative literals in the package's tests: a package one level deeper needs one more `..` to reach `internal/embed` or the repository root.
5. Edit package paths cited in `internal/embed/defaults` and the `owner_package` values in its `mechanisms.yaml` only in the authoring source, then regenerate projections.
6. Regenerate `docs/generated/` with `make docs-generate`; do not hand-edit it. Leave `docs/adr/` and `CHANGELOG.md` as historical records.

## Verification per wave

| Gate | Check |
|---|---|
| G1 | `go build ./...` and `go vet ./...` |
| G2 | `gofmt -l` over touched files is empty |
| G3 | `go test ./...` and `go test -tags integration ./internal/domain/...` |
| G4 | `make coverage-manifest-check` |
| G5 | `make generate-embedded-check` when embedded text changed |
| G6 | `make docs-generate`, then `make docs-generated-gate` |
| G7 | `bash scripts/check-convergence.sh` and the quality-budget check |
| G8 | the `go list` import graph is equal before and after once old paths are mapped to new ones |
| G9 | the old path no longer appears outside `docs/adr/`, `CHANGELOG.md` and `.analysis/` |
| G10 | `git diff --check` |

A failing gate blocks the next wave and the wave is reverted as a whole. No runtime data migration is involved.

## Known limits

- The baseline measured on 2026-10-12 before the first move was green, and the same gates were green after the last wave: `go test ./...`, the integration-tagged suite, `make coverage-manifest-check`, `make generate-embedded-check`, `scripts/check-convergence.sh`, `scripts/check-quality-budgets.sh` and `golangci-lint run ./...`.
- `owner_package` values in `mechanisms.yaml` are parsed but not validated by any test, so they go stale silently after a move unless edited by hand.
- The local `.strategist/` runtime is a gitignored projection of `internal/embed/defaults`. After the embedded text edits of the waves it must be refreshed (`make install`, or `strategist upgrade` with a rebuilt binary). Until then two local-only spec tests report mirror drift (`TestConfidenceProducerContractsRuntimeParityWhenInstalled`, `TestLocalRuntimeMirrorsCanonicalNormativeFilesWhenPresent`); CI has no `.strategist/` and skips them.
- Moving the three role-conformance test files of `handoff` changed the `test_suite_digest` of three entries in `internal/embed/defaults/plugins/catalog.yaml`; it was regenerated with `make generate-embedded`.
- The help text of `strategist handoff verify` names the package path, so its golden file and `docs/generated/command-tree.md` changed with wave 6.
- Some dated notes and test sample paths still mention the old locations on purpose: `docs/adr/`, `CHANGELOG.md`, `docs/releases/`, `tests/evals/regression/`, and sample strings in `internal/plugins/policy/write_enforcement_test.go` and `internal/dojo/calibration/tasks_test.go`.
- The group names `internal/cli/` and `internal/dojo/calibration` were applied as recommended; they remain the maintainer's to change.

## Wave status

| Wave | Scope | Status |
|---|---|---|
| 0 | baseline recorded, CI path `./internal/runtime/env` corrected, empty `internal/initiative/` removed | done |
| 1 | `hostusage`, `usagesummary` under `telemetry/` | done |
| 2 | `hardening` under `runtime/` | done |
| 3 | `bindingdrift`, `gatecriteria`, `refinement` under `mission/` | done |
| 4 | `calibration` under `dojo/` | done |
| 5 | `cliutil`, `check` under `cli/` | done |
| 6 | `handoff` under `mission/` | done |
| 7 | `roster` under `application/` | done |
