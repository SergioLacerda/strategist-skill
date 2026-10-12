# Runbook: Splitting a Go Package into Affinity Sub-packages

## Trigger

Use this runbook when a request asks to organise one large, flat Go package into sub-packages or sub-directories by
affinity, usually together with consolidating shared constants, paths and variables. Signals: the package has
dozens of files that already cluster by file-name prefix; the same string literal is spelled in several files; the
user asks for "common constants" and "subdirectories by affinity".

Not a trigger:

- a narrow lint or complexity finding, or "reduce complexity below N": use
  `refactoring-for-agent-operations.md`;
- moving whole packages between parents under `internal/`: see `docs/architecture/internal-layout.md`;
- any change of behaviour, serialised output or public contract.

Scope of what was verified: Go packages of this repository, one mission
(`20261012-internal-install-refactor-evaluation`, `internal/install`, 206 files into 10 sub-packages). Steps that
were not re-run are marked *unverified*.

## Steps

1. **Capture the baseline.** Run `go build ./...`, `go vet ./...`, `golangci-lint run ./...`, `go test ./...`,
   `go test -tags spec ./tests/spec/...`, `bash scripts/check-coverage-manifest.sh` and `make quality-budget-gate`.
   Record each result, the package's own coverage
   (`go test -coverprofile=/tmp/c.out -coverpkg=./<pkg> ./<pkg>`, non-recursive like the gate) and the files that
   import it (`grep -rl 'internal/<pkg>"' --include='*.go' cmd internal tests`). A failure that is not yours stays
   recorded as a pre-existing exception; do not fix files another session owns.
2. **Read the rules keyed on the path** before designing: depguard globs and file-path exclusions in
   `.golangci.yaml`, `TestLateralIsolation` in `internal/domain/architecture_test.go`, the
   `scripts/coverage-packages.tsv` rows, `scripts/quality-budgets.tsv`, and any code that derives a path from its own
   source location (`runtime.Caller`) or uses `..` literals in tests.
3. **Consolidate constants first, in place.** Replace repeated literals with named constants in the same package
   (file names, directories, identities, slot and role names; use `domain.Slot*` instead of string slot keys).
   No file moves. Run the package tests: this step alone answers "which constants are shared".
4. **Measure the dependency graph.** For every production file, collect the identifiers it declares and the files
   that reference them. Exclude selector names (`x.Name`) and the blank identifier `_`; a `var _ I = ...` once
   inflated a hub from 14 references to 208. Compute strongly connected components. The graph ignores method calls
   through selectors: it is a lower bound on coupling, and the compiler plus the tests are the oracle.
5. **Design on an acyclic graph.** Group by affinity only after step 4: a split by file-name prefix produced
   bidirectional edges in 7 of 10 clusters. Resolve each cycle by moving the few identifiers that cause it
   (types used by two groups, generic helpers declared inside feature files). Leaves first (constants, helpers),
   then the hub that others use (the catalog), then features, then UI; the root keeps the orchestration. Check the
   mapping at AST level before moving: zero edges into the root and no package-level cycle.
6. **Choose package names against collisions.** Count identifiers in the package named like each candidate
   (`catalog` was used 426 times in 65 files and had to become `plugincatalog`).
7. **Move one package per wave, in dependency order.** Move the files with their tests. Export every identifier that
   crosses a boundary and qualify its uses. For an unexported struct field or method that crosses, rename the
   declaration with a type-aware rename (`gopls rename`), not only the call sites. A newly exported field that was
   not serialised needs `yaml:"-" json:"-"`. After each wave run build, vet and the package tests; re-run the full
   vet at the end of a batch, because a per-file check once passed for a move that left an undefined identifier in
   another package's test.
8. **Relocate tests deliberately.** Shared test helpers go to a fixtures package imported by tests only (give it the
   same lint exclusions as `_test.go`); when an internal test package would create an import cycle with it, make the
   test an external `_test` package. Replace `..` literals with a path derived from the fixtures package, and fix
   the depth of production code that uses `runtime.Caller`, with a test that the resolved root holds `go.mod`.
   Tests that drive code only through the root orchestrator stay in the root.
9. **Meet the per-package gates.** Every discovered production package needs a row or an exemption in
   `scripts/coverage-packages.tsv`; the gate measures each package non-recursively, so a package whose tests stay in
   the root measures low. Set a new row at the measured floor only with the operator's agreement, never lower an
   existing threshold, and regenerate `docs/generated/coverage-policy.md` with
   `bash scripts/generate-coverage-policy.sh`.
10. **Keep lint and budgets green.** New exported symbols need doc comments and names that do not stutter with the
    package name; errors returned across a package boundary trip `wrapcheck` (a `//nolint:wrapcheck` with the reason
    "sibling package of the same module" avoids adding uncovered error branches); each non-test file stays within
    200 lines and cognitive complexity 7.
11. **Re-point the outside importers** (decision below) and update documentation that names moved file paths
    (`grep -rn 'internal/<pkg>/<file>.go' docs .github`); do not rewrite ADRs.
12. **Verify with the same commands as step 1** and report: files per package, importers, coverage per package,
    gates that failed for reasons outside the change, and the open items.

## Decision Point

| Question | Options | Used in the verified mission |
|---|---|---|
| Public surface after the split | root aliases for one release, or re-point importers directly | re-pointed directly (operator decision at the gate) |
| Where shared constants live | a package-local `names` package, or `internal/domain` | package-local; cross-package duplicates left as a separate side quest |
| Coverage of the new packages | move tests until each meets the old threshold, or set ratcheted floors | ratcheted floors at the measured values, rows added; operator to confirm |
| Package names | short domain words, or distinctive names | distinctive names where the short word collided with locals |

## Stop Conditions

- The AST graph shows a cycle that cannot be broken by relocating generic helpers or shared types: stop and ask.
- The split needs a new behaviour, a changed serialised output or a changed public contract: this is a separate
  demand.
- A gate must be lowered or an exemption added to pass: stop and ask; a lowered threshold is a policy change.
- A gate fails for a file you did not touch (another session's staged change): record it and report; do not fix it.
- The git index holds changes you did not make: do not run `git add`, `git commit`, `git mv` or `git rm`; deliver
  the result in the working tree.
- A step cannot be verified with real output: mark it *unverified* and report it.

## Reference

- `.analysis/refined/20261012-internal-install-refactor-evaluation/design.md` (D1-D8, "Corrections to the discovery
  handoff") and `implementation-status.md` (deviations, gates, coverage).
- `docs/runbooks/refactoring-for-agent-operations.md`: local, signal-driven cleanup.
- `docs/architecture/internal-layout.md`: whole-package moves and the `internal/install` sub-package table.
- `scripts/coverage-packages.tsv`, `.golangci.yaml`, `internal/domain/architecture_test.go`.
