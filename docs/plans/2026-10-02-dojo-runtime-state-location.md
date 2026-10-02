# Dojo Runtime-State Location Implementation Plan

> **For Codex:** Execute the tasks in this plan against the accepted dojo runtime-state ownership handoff.

**Goal:** Make the current `<base_path>/dojo` ownership contract explicit and robust for path resolution, persistence integrity, compatibility, and operator recovery without relocating existing state.

**Architecture:** Keep scenario inputs and all existing dojo artifacts under the configured workspace dojo domain. Add typed storage/path contracts and atomic, recovery-aware persistence at the storage boundary; preserve checker verdicts separately from persistence warnings. Document the compatibility window and future migration boundary, without moving or dual-writing user state.

**Tech Stack:** Go, standard-library filesystem primitives, existing dojo domain and CLI tests, Markdown documentation.

---

### Task 1: Establish the storage ownership contract

**Files:**
- Modify: `internal/dojo/result_store.go`
- Modify: `internal/dojo/learning.go`
- Modify: `cmd/strategist/dojo/dojo.go`
- Modify: `internal/domain/dojo_types.go` only if a typed storage policy is required
- Test: `internal/dojo/result_store_test.go`, `internal/dojo/learning_test.go`, `cmd/strategist/dojo/dojo_test.go`

1. Add focused path-resolution and artifact-classification tests for configured base-path overrides and scenario isolation.
2. Run the focused tests and confirm any missing contract fails.
3. Implement the smallest package-owned storage contract that preserves `<base_path>/dojo`, stable scenario identity, and existing public paths.
4. Run the focused tests and format changed Go files.

### Task 2: Harden persistence integrity

**Files:**
- Modify: `internal/dojo/result_store.go`
- Test: `internal/dojo/result_store_test.go`

1. Add tests for concurrent same-scenario and cross-scenario writes, atomic latest-result replacement, ordered history append, interrupted/failed writes, and corrupt/orphaned state reporting.
2. Run those tests to establish the failing behavior.
3. Implement atomic replacement and synchronization/recovery semantics while retaining best-effort persistence and distinguishing check verdict from persistence failure.
4. Run the storage package tests and format the implementation.

### Task 3: Define compatibility and lifecycle behavior

**Files:**
- Modify: `cmd/strategist/dojo/dojo.go`
- Modify: `internal/dojo/result_store.go`
- Modify: `internal/dojo/learning.go`
- Test: `cmd/strategist/dojo/dojo_test.go`, `internal/dojo/result_store_test.go`, `internal/dojo/learning_test.go`
- Create if needed: focused dojo compatibility tests

1. Add contract coverage for upgrade/reinstall, workspace move, base-path-only backup/restore, legacy-state discovery, stale/orphaned state, and partial migration failure.
2. Implement read-preserving discovery/reporting and explicit migration boundaries without automatic moves, cleanup, symlinks, or dual writes.
3. Run targeted command and storage tests.

### Task 4: Align public documentation

**Files:**
- Modify: `docs/architecture/strategist-concepts.md`
- Modify: `docs/configuration.md`
- Create if needed: `docs/runbooks/dojo-runtime-state-location.md`

1. Document the typed ownership matrix, stable public paths, retention expectations, persistence warnings, and explicit future migration policy.
2. Check links and wording against the implementation and accepted ADR.

### Task 5: Validate the implementation

1. Run `go test ./internal/dojo ./cmd/strategist/dojo`.
2. Run `go test ./...` with an isolated writable Go cache.
3. Run the repository quality/lint and generated-drift checks relevant to changed files.
4. Recheck that no `.strategist` artifact or real user workspace state was changed and report any pre-existing or environmental failures separately.

