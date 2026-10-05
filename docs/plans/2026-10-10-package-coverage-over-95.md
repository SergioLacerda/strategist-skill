# Package Coverage Above 95% Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Raise the independently measured coverage of `internal/telemetry`, `internal/install`, and `cmd/strategist` strictly above 95% with hermetic behavioral tests.

**Architecture:** Tests stay in the owning packages and exercise public behavior plus package-local helpers where the existing white-box suites already establish that convention. Filesystem scenarios use `t.TempDir`, isolated HOME/configuration, and deterministic fixtures; production behavior and coverage thresholds remain unchanged.

**Tech Stack:** Go testing, atomic cover profiles, `go tool cover`, testify, temporary filesystem fixtures.

---

### Task 1: Close telemetry coverage gaps

**Files:**
- Modify: `internal/telemetry/coverage_gaps_test.go`

**Steps:**

1. Add table-driven tests for LEVELING budget event kinds, confidence validation status/ground-truth branches, and append-only history duplicate/malformed-line behavior.
2. Run the package profile and inspect `go tool cover -func`.
3. Add only the remaining real-path cases until the package is strictly above 95%.

### Task 2: Close installer coverage gaps

**Files:**
- Modify: `internal/install/coverage_gaps_test.go`

**Steps:**

1. Add hermetic tests for the uncovered embedded-prepare or silent-binding error/success branch identified by the profile.
2. Run the package profile and confirm it is strictly above 95%.

### Task 3: Close CLI coverage gaps

**Files:**
- Create: `cmd/strategist/coverage_gaps_test.go`

**Steps:**

1. Add table-driven tests for uncovered command adapters, budget/config helpers, delegation output, integration calibration/doctor branches, and runbook preparation using existing package fixtures.
2. Keep external-provider and live-host paths hermetic by using deterministic failures, temp roots, and injected dependencies.
3. Run the package profile and add focused cases until it is strictly above 95%.

### Task 4: Validate all three package gates

**Files:**
- Verify: `scripts/coverage-packages.tsv`
- Verify: `scripts/coverage-per-package.sh`

**Steps:**

1. Re-measure each named package with its own atomic profile and `GOMAXPROCS=2`.
2. Run the focused package tests and `git diff --check`.
3. Confirm no production code or coverage threshold changed.

