# Test Centralization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reorganize the test directory structure so that `tests/` contains two explicit subfolders (`integration/` and `spec/`) with dedicated build tags, eliminating the out-of-place `strategist/tests/` package.

**Architecture:** Co-located unit tests in `internal/*/` and `cmd/strategist/` stay untouched (whitebox access requirement). The existing `tests/*.go` files move to `tests/integration/`. The `strategist/tests/` package migrates to `tests/spec/` with its fixtures, specs, and run script. Each layer has its own build tag: none for unit, `spec` for contract tests, `integration` for E2E.

**Tech Stack:** Go (build tags, package naming), Makefile, GitHub Actions YAML

---

### Task 1: Create `tests/integration/` and move E2E files

**Files:**
- Create: `tests/integration/` (directory)
- Move: `tests/*.go` → `tests/integration/`
- Move: `tests/fixtures/` → `tests/integration/fixtures/`
- Delete: `tests/specs/` (empty directory)

**Step 1: Create the integration subdirectory**

```bash
mkdir -p tests/integration
```

**Step 2: Move all Go test files**

```bash
mv tests/bench_e2e_test.go     tests/integration/
mv tests/compile_test.go       tests/integration/
mv tests/e2e_cli_happy_path_test.go tests/integration/
mv tests/e2e_harness_test.go   tests/integration/
mv tests/e2e_skill_pipeline_test.go tests/integration/
mv tests/fixtures_test.go      tests/integration/
mv tests/install_test.go       tests/integration/
mv tests/stale_test.go         tests/integration/
mv tests/yaml_roundtrip_test.go tests/integration/
```

**Step 3: Move fixtures directory**

```bash
mv tests/fixtures tests/integration/fixtures
```

**Step 4: Remove empty specs directory**

```bash
rmdir tests/specs 2>/dev/null || true
```

**Step 5: Change package name in all moved files**

In every file under `tests/integration/`, change:
```go
package tests_test
```
to:
```go
package integration_test
```

Files to update (all 9 — search with `grep -l "package tests_test" tests/integration/`):
- `tests/integration/bench_e2e_test.go`
- `tests/integration/compile_test.go`
- `tests/integration/e2e_cli_happy_path_test.go`
- `tests/integration/e2e_harness_test.go`
- `tests/integration/e2e_skill_pipeline_test.go`
- `tests/integration/fixtures_test.go`
- `tests/integration/install_test.go`
- `tests/integration/stale_test.go`
- `tests/integration/yaml_roundtrip_test.go`

The `//go:build integration` tag is already present on each file — do NOT remove it.

**Step 6: Verify build compiles**

```bash
go build -tags=integration ./tests/integration/...
```

Expected: no errors (some may have undefined symbols until the binary is built, but compilation must pass).

**Step 7: Verify relative path in `fixtures_test.go` still resolves**

`fixtures_test.go` uses `filepath.Glob(filepath.Join("fixtures", "*.yaml"))`. In Go, tests run with working directory set to the package directory. After the move this resolves to `tests/integration/fixtures/*.yaml` — which is where we moved the fixtures. No code change needed.

**Step 8: Run integration tests to confirm**

```bash
go test -tags=integration -race ./tests/integration/...
```

Expected: same results as before the move (all pass or same failures as baseline).

**Step 9: Commit**

```bash
git add tests/integration/ tests/fixtures tests/specs
git add tests/bench_e2e_test.go tests/compile_test.go tests/e2e_cli_happy_path_test.go
git add tests/e2e_harness_test.go tests/e2e_skill_pipeline_test.go tests/fixtures_test.go
git add tests/install_test.go tests/stale_test.go tests/yaml_roundtrip_test.go
git commit -m "refactor: move E2E tests to tests/integration/ with integration build tag"
```

---

### Task 2: Create `tests/spec/` and migrate `strategist/tests/`

**Files:**
- Create: `tests/spec/` (directory)
- Move: `strategist/tests/spec_alignment_test.go` → `tests/spec/spec_alignment_test.go`
- Move: `strategist/tests/specs/` → `tests/spec/specs/`
- Move: `strategist/tests/fixtures/` → `tests/spec/fixtures/`
- Move: `strategist/tests/run-tests.sh` → `tests/spec/run-tests.sh`
- Delete: `strategist/tests/` (directory, now empty)

**Step 1: Create the spec subdirectory**

```bash
mkdir -p tests/spec
```

**Step 2: Move supporting directories**

```bash
mv strategist/tests/specs    tests/spec/specs
mv strategist/tests/fixtures tests/spec/fixtures
```

**Step 3: Move and update `spec_alignment_test.go`**

```bash
mv strategist/tests/spec_alignment_test.go tests/spec/spec_alignment_test.go
```

Edit `tests/spec/spec_alignment_test.go` — two changes at the top of the file:

Change the package declaration:
```go
// BEFORE
package tests

// AFTER
//go:build spec

package spec_test
```

> **Path check:** `testDir(t)` uses `runtime.Caller(0)` which returns the source file path at compile time. After the move, `testDir(t)` → `tests/spec/`. Then:
> - `filepath.Join(testDir(t), "specs", ...)` → `tests/spec/specs/` ✓
> - `filepath.Join(testDir(t), "fixtures", ...)` → `tests/spec/fixtures/` ✓
> - `repoRoot(t)` = `filepath.Clean(filepath.Join("tests/spec", "../.."))` = repo root ✓
>
> No other path changes needed in the test file.

**Step 4: Move `run-tests.sh`**

```bash
mv strategist/tests/run-tests.sh tests/spec/run-tests.sh
chmod +x tests/spec/run-tests.sh
```

> The script uses `SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"` — self-resolving, works from any location. `FIXTURES_DIR="$SCRIPT_DIR/fixtures"` → `tests/spec/fixtures/` ✓. `REPO_ROOT` → repo root ✓. No edits needed.

**Step 5: Remove `strategist/tests/`**

```bash
rmdir strategist/tests
```

Expected: succeeds (directory is now empty).

**Step 6: Run spec tests to confirm**

```bash
go test -tags=spec ./tests/spec/...
```

Expected: all tests pass (same as running `go test ./strategist/tests/...` before).

**Step 7: Confirm unit tests still pass**

```bash
go test -race $(go list ./... | grep -v '/testutil')
```

Expected: all pass, `strategist/tests` no longer appears in the list.

**Step 8: Commit**

```bash
git add tests/spec/
git add strategist/tests/
git commit -m "refactor: migrate strategist/tests/ to tests/spec/ with spec build tag"
```

---

### Task 3: Update Makefile

**Files:**
- Modify: `Makefile`

**Step 1: Update the `integration` target**

```makefile
# BEFORE
integration:
	go test -race -tags=integration ./tests/...

# AFTER
integration:
	go test -race -tags=integration ./tests/integration/...
```

**Step 2: Update the `cover-html` target**

```makefile
# BEFORE
cover-html:
	go test -race -coverprofile=coverage.out -coverpkg=./internal/... ./internal/... ./tests/...

# AFTER
cover-html:
	go test -race -coverprofile=coverage.out -coverpkg=./internal/... ./internal/... ./tests/integration/...
```

**Step 3: Add `spec` and update `test-all` targets**

```makefile
# ADD after the integration target:
spec:
	go test -race -tags=spec ./tests/spec/...

# UPDATE test-all:
test-all: test integration spec
```

Update the `.PHONY` line at the top to include `spec`:
```makefile
.PHONY: build test test-all integration spec test-lite ...
```

**Step 4: Verify Makefile targets work**

```bash
make test
make spec
make integration
```

Expected: all pass.

**Step 5: Commit**

```bash
git add Makefile
git commit -m "build: add spec make target and update integration path to tests/integration/"
```

---

### Task 4: Update CI workflow

**Files:**
- Modify: `.github/workflows/test.yml`

**Step 1: Update the Integration tests step**

```yaml
# BEFORE
- name: Integration tests
  run: go test -tags=integration -race ./tests/...

# AFTER
- name: Integration tests
  run: go test -tags=integration -race ./tests/integration/...
```

**Step 2: Update the Fixture and schema validation step**

```yaml
# BEFORE
- name: Fixture and schema validation
  run: |
    pip install pyyaml --quiet
    bash strategist/tests/run-tests.sh

# AFTER
- name: Fixture and schema validation
  run: |
    pip install pyyaml --quiet
    bash tests/spec/run-tests.sh
```

**Step 3: Commit**

```bash
git add .github/workflows/test.yml
git commit -m "ci: update test paths after tests/integration/ and tests/spec/ reorganization"
```

---

### Task 5: Final verification

**Step 1: Full test suite**

```bash
go test -race $(go list ./... | grep -v '/testutil')
```

Expected: all packages pass, `strategist/tests` absent from output.

**Step 2: Spec layer**

```bash
go test -tags=spec -race ./tests/spec/...
```

Expected: all spec alignment tests pass.

**Step 3: Integration layer**

```bash
go test -tags=integration -race ./tests/integration/...
```

Expected: all E2E tests pass.

**Step 4: Fixture script**

```bash
bash tests/spec/run-tests.sh
```

Expected: `Results: N passed, 0 failed`.

**Step 5: Confirm `strategist/tests/` is gone**

```bash
ls strategist/tests 2>&1
```

Expected: `No such file or directory`.

**Step 6: Final commit (if any cleanups needed)**

```bash
git status  # should be clean
```
