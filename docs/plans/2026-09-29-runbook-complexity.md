# Runbook Complexity Reduction Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reduce the cognitive complexity of `matchDeclaredSignals` and `ValidateRunbook` below the repository threshold without changing behavior.

**Architecture:** Extract the per-declared-signal predicate and the independent validation groups into small helpers. Preserve matching order, error order, `errors.Join`, and the existing public API.

**Tech Stack:** Go, `gocognit`, package tests.

---

### Task 1: Refactor the two reported functions

**Files:**
- Modify: `internal/runbook/select_runbook_match.go`
- Modify: `internal/runbook/runbook.go`
- Test: `internal/runbook/select_runbook_test.go`
- Test: `internal/runbook/runbook_test.go`

**Step 1: Extract cohesive helpers**

Move the nested declared-signal match predicate and the basic/signal/nested validation loops behind helpers, retaining their current iteration and append order.

**Step 2: Run focused tests**

Run: `go test ./internal/runbook`

Expected: PASS.

**Step 3: Verify the named complexity findings**

Run: `/home/sergio/go/bin/gocognit -over 6 internal/runbook/select_runbook_match.go internal/runbook/runbook.go`

Expected: no `matchDeclaredSignals` or `ValidateRunbook` finding.

**Step 4: Check formatting and diff hygiene**

Run: `gofmt -w internal/runbook/select_runbook_match.go internal/runbook/runbook.go && git diff --check`

Expected: no formatting or whitespace errors.
