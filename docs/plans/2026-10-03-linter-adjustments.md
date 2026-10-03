# Linter Adjustments Implementation Plan

> **For Codex:** Apply the tasks in order and preserve unrelated worktree changes.

**Goal:** Resolve the exact linter diagnostics in `ajuste_linter.txt` with behavior-preserving Go changes.

**Architecture:** Keep filesystem and handoff responsibilities in their existing packages. Extract only cohesive cleanup, recovery, and commit helpers needed for error handling and cognitive-complexity reduction.

**Tech Stack:** Go, golangci-lint, gocognit, repository quality-budget scripts.

---

### Task 1: Harden execution-entry error handling

**Files:**
- Modify: `cmd/strategist/mission/execution_entry.go`

1. Add contextual wrapping for all reported external errors.
2. Handle temporary-file cleanup errors through a small helper and preserve primary failures.
3. Add the scoped `gosec` justification for the validated execution-entry path.
4. Extract recovery helpers until `recoverExecutionEntry` is below complexity 15.

### Task 2: Reduce submit-flow complexity

**Files:**
- Modify: `cmd/strategist/mission/submit_flow.go`

1. Extract outcome consumption and claim projection helpers.
2. Preserve idempotency and journal updates.
3. Confirm `commitExecutionEntry` is below complexity 15.

### Task 3: Fix refinement documentation and wrapping

**Files:**
- Modify: `internal/refinement/documentation_targets.go`

1. Correct the exported GoDoc prefix.
2. Wrap validation errors from `internal/handoff`.

### Task 4: Verify

1. Run `gofmt` on modified Go files.
2. Run focused package tests.
3. Run `gocognit`, `golangci-lint`, and the quality-budget script.
4. Inspect `git diff` and report only task-related changes; do not stage or commit.
