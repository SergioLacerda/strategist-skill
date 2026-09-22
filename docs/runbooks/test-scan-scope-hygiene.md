# Runbook: A Test's Filesystem Walk Silently Scanning an Untracked Directory

## Symptom

`go test ./...` (or `make test`, which runs `go test -race ./...`) prints
its usual per-package `ok` / `(cached)` lines, then goes silent for far
longer than the previous package took — tens of seconds to minutes, with no
output — making the run look hung even though it eventually finishes and
passes. The effect is much more pronounced under `-race` (instrumentation
overhead multiplies the extra work) than without it, which is why it can go
unnoticed for a long time: the same test barely registers in a plain
`go test ./...`.

## Root Cause

A test walks the filesystem with `filepath.Walk`/`filepath.WalkDir` from a
broad root — often the repo root — to inspect source files (a lint-style
check, a hardcoded-list guard, a corpus scan, etc.). The walk has a
directory-skip allowlist (`.git`, `node_modules`, build output, and similar),
but that allowlist was not updated when a new **untracked, gitignored**
directory appeared at the walked root — for example a local dependency
cache, a scratch/build directory, or any other tool-generated tree that lives
on disk but was never meant to be part of the repo's own source corpus. The
walk silently descends into it and processes every file there too, exactly
as if it were project source.

This is not a deadlock and not flaky — it is deterministic extra work
proportional to the size of the untracked directory. A concrete instance:
`TestNoHardcodedRoleListsOutsideTheRegistry`
(`internal/domain/role_registry_lists_test.go`) walks the whole repository
from root looking for hardcoded role-name lists outside the domain registry.
Its `skipTree` allowlist (`.git`, `node_modules`, `defaults`, `.strategist`,
`.analysis`) did not include `.tmp-gomodcache-cli/`, a 161MB gitignored
directory holding vendored dependency source (grpc, otel, cobra, huh,
grpc-gateway, x/text — 3,686 non-test `.go` files). Reading and
regex-scanning those files line by line, under race-detector instrumentation,
took 85.7s in isolation — effectively all of the ~90s the enclosing package
took under `go test -race`, and the reason the overall suite looked hung
right after the previous (alphabetically earlier) package finished.

## Resolution Steps

1. **Isolate the slow package.** Re-run with `-v` and no cache:
   `go test -race -count=1 -v ./<suspect-package>/...`. Package-level output
   alone won't tell you which test is slow; per-test `--- PASS: Test (Ns)`
   lines will.
2. **Sort by duration** to find the outlier:
   `go test -race -count=1 -v ./<pkg>/... 2>&1 | grep -E "^--- PASS|^--- FAIL"`
   and scan for a duration far larger than its neighbors. One test taking
   nearly the whole package's time, while every other test in the same
   package finishes in milliseconds, is the signature of this bug class —
   not a sign every test in the package is slow.
3. **Check whether the slow test does a filesystem walk:**
   `grep -n "filepath.Walk" <test_file>`. If it does, find its skip/allow
   list (commonly a small `switch`/slice of directory names) and its walk
   root.
4. **Diff the skip list against what's actually on disk at that root:**
   `ls -a <walk-root>` and compare against `.gitignore` — any gitignored
   directory at or under the walk root that is *not* in the skip list is a
   candidate. Confirm size/file count with `du -sh <dir>` and
   `find <dir> -name "*.go" | wc -l` (or the relevant extension) to see if
   it's large enough to explain the slowdown.
5. **Fix by generalizing the skip rule, not just adding one name.** A
   one-off temp/cache directory is likely to reappear under a different
   exact name later (a different tool, a different local setup). Prefer a
   pattern match over the current allowlist's naming convention — for
   example, this repo's fix added
   `if strings.HasPrefix(name, ".tmp-") { return fs.SkipDir }` to the
   existing `skipTree` `switch`, so any future `.tmp-*` directory is
   excluded automatically instead of requiring another one-line patch.
6. **Verify the fix at three levels**: the isolated test's duration drops to
   in line with its neighbors; the enclosing package's total duration drops
   correspondingly; the full suite (`go test -race ./...` /
   `make test`) completes in a normal amount of time with all packages still
   passing. In the originating case: the isolated test went from 85.7s to
   2.7s, the package from 90s to 3.8s, and the full `-race` suite from
   apparently-hung to ~28.5s end-to-end.
7. **Check for other instances of the same pattern**, since one stale skip
   list is evidence the convention isn't self-maintaining:
   `grep -rln "filepath\.Walk" --include="*_test.go" .`, then for each hit,
   check what root it walks. A walk bounded to a narrow, named subtree
   (`testdata/`, a specific fixture directory, a specific package tree) is
   not at risk — only a walk from the repo root, or from any directory that
   could gain a new untracked sibling, is. As of this runbook's authoring,
   this repository has 8 other `filepath.Walk`/`WalkDir` call sites in
   `*_test.go` files, all bounded to narrow subtrees;
   `role_registry_lists_test.go` was, and remains, the only repo-root
   walker.

## Related: Other Causes of a Slow Test Package (Different Root Cause)

A test package taking unexpectedly long is not always this runbook's
filesystem-walk-scanning-an-untracked-directory pattern. A separate
diagnosis (2026-09-21, `internal/check` and `cmd/strategist`) found two
other, unrelated bug classes producing the same *symptom* — a package whose
total time is dominated by one or two outlier tests while its neighbors
finish in milliseconds — worth checking before assuming this runbook's root
cause applies:

- **Orphaned grandchild process holding stdout/stderr open**: a test kills a
  shell-wrapped subprocess (`sh -c ...`) via `context.WithTimeout` +
  `exec.CommandContext`. The context deadline fires and the *direct* child
  (the shell) is killed correctly, but a *grandchild* it forked (e.g. a
  `sleep`) is not — Go's `exec.CommandContext` only signals its direct
  child. The orphaned grandchild keeps the inherited pipe write-ends open,
  so `cmd.CombinedOutput()`/`cmd.Wait()` blocks until it exits on its own.
  The test's own configured timeout may be correctly short (e.g. 300ms);
  the wall-clock cost comes entirely from waiting out the orphan's real
  runtime. See `internal/check/check_ranked_readiness_test.go:539` and
  `internal/check/check_ranked_host_node.go:30-40` for a concrete instance.
- **Repeated real, full-fidelity I/O or subprocess work by design**: several
  tests in `cmd/strategist` each pay a real ~1.5s cost (full embedded-tree
  extraction, install/upgrade/rollback, or a real `go test` subprocess
  invocation) because that is what they are built to exercise — not because
  of a bug. Diffing against this runbook's `filepath.Walk` signature (grep
  the slow test for `filepath.Walk`/`WalkDir`) quickly rules this pattern in
  or out: if the outlier test's slowness traces to `exec.Command*` or real
  install/build steps instead, this runbook does not apply — see
  `internal/install/installer.go:73` and `cmd/strategist/upgrade_test.go:67`
  for examples.

Full record: `.analysis/refined/20260921-slow-test-packages-diagnosis/analysis.md`.

This runbook's own `applies_when`/`analysis` steps in
`test-scan-scope-hygiene.runbook.yaml` are intentionally left unchanged —
they are machine-matched by `select_runbook` against mission signals, and
folding in an unrelated bug signature would make this runbook match (and
mis-suggest its `filepath.Walk` fix for) cases it does not actually explain.

## Reference

- Origin: this conversation's own diagnosis and fix (2026-09-21) — see
  `.analysis/refined/20260921-test-maintenance-runbook/analysis.md` for the
  full mission record, including the side-quest audit of other walk sites.
- Fix applied in: `internal/domain/role_registry_lists_test.go` (`skipTree`).
- Related: `docs/runbooks/verifying-test-failures.md` — a different problem
  with a similar diagnostic shape: that runbook is for triaging a test that
  is *failing*; this one is for a test that *passes* but silently takes far
  longer than expected because of what it scanned.
