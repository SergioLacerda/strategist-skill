# Go 1.27.2 Toolchain Stabilization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make Go 1.27.2 the explicit and consistent toolchain for module metadata, local quality gates, CI, contributor documentation, ADR policy, and contract tests.

**Architecture:** `go.mod` remains the sole version authority. CI continues to consume that authority through `go-version-file`, while local Make targets derive an exact `GOTOOLCHAIN` from its `toolchain` directive or, for Go 1.27.2, its exact `go` directive. Documentation and contract tests mirror the same canonical version without introducing independent configuration.

**Tech Stack:** Go modules, GNU Make, GitHub Actions, Bash, Go spec tests.

---

### Task 1: Establish the canonical module toolchain

**Files:**
- Verify: `go.mod:3`

**Steps:**

1. Keep `go 1.27.2` as the canonical module and exact patch declaration.
2. Do not add a redundant matching `toolchain` directive because Go 1.27.2's `go mod tidy` removes it.
3. Verify the declaration with `go env GOVERSION` and a direct `awk` check.

### Task 2: Align contributor and policy documentation

**Files:**
- Modify: `CONTRIBUTING.md:5-11`
- Modify: `docs/adr/0014-monorepo-and-toolchain-policy.md:25-52`

**Steps:**

1. Replace previous patch-version references with `1.27.2`.
2. Preserve the statement that `go.mod` is authoritative and CI/local verification use the exact patch toolchain.

### Task 3: Align executable contract tests

**Files:**
- Modify: `tests/spec/cicd_residual_contract_test.go:192`
- Modify: `tests/spec/cicd_policy_eval_residuals_test.go:51-52`

**Steps:**

1. Update exact normative expectations to `1.27.2`.
2. Run the focused spec tests with `-tags spec`.

### Task 4: Validate CI and local paths

**Files:**
- Verify: `.github/workflows/test.yml`
- Verify: `.github/workflows/release.yml`
- Verify: `.github/workflows/codeql.yml`
- Verify: `Makefile`
- Verify: `make/quality.mk`

**Steps:**

1. Confirm every Go workflow uses `go-version-file: go.mod`.
2. Confirm local quality and vulnerability targets resolve `PINNED_GOTOOLCHAIN=go1.27.2` from the `go` directive fallback.
3. Run focused tests, module verification, and local doctor checks with isolated writable caches.
