# cmd-strategist-refactor Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Keep `cmd/strategist/mission` as the Cobra adapter and leave the root executable with only composition and process-owned runtime wiring.

**Architecture:** Move command output encoding and completion-input parsing into the mission adapter, where Cobra and transport concerns already live. Keep persistence, host execution, telemetry, and provider/runtime assembly in the root composition bridge or existing internal owners; move their tests only when they no longer require root-private helpers.

**Tech Stack:** Go, Cobra, internal application/mission packages, hermetic Go tests.

---

### Task 1: Make mission construction composition-based

**Files:**
- Modify: `cmd/strategist/mission/mission.go`
- Modify: `cmd/strategist/commands.go`
- Modify: `cmd/strategist/mission_wiring_test.go`
- Modify: `cmd/strategist/mission/lifecycle_test.go`

Change `mission.New` to consume one `Composition`, update all callers, and preserve the registered command tree and dependency injection behavior.

### Task 2: Move CLI result and completion transport into the adapter

**Files:**
- Create: `cmd/strategist/mission/output.go`
- Create: `cmd/strategist/mission/completion_input.go`
- Delete: `cmd/strategist/mission_completion_json.go`
- Modify: `cmd/strategist/mission_persistence.go`
- Modify: `cmd/strategist/mission_composition.go`

Keep JSON/text envelopes and error wording stable while removing the root-package implementations.

### Task 3: Redistribute transport and output coverage

**Files:**
- Create: `cmd/strategist/mission/output_test.go`
- Create: `cmd/strategist/mission/completion_input_test.go`
- Modify: root mission and command regression tests to use public adapter boundaries.

Run focused mission tests, then the full Go suite and architecture/quality gates using isolated caches. Do not stage, commit, reset, or update an unrelated command-tree golden.
