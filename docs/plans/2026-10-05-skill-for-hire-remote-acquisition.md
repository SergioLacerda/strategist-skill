# Skill-for-Hire Remote Acquisition Implementation Plan

> **For Codex:** Execute this plan task-by-task in the current workspace. Preserve unrelated worktree changes and do not commit unless explicitly requested.

**Goal:** Add a fail-closed, explicitly network-enabled Git acquisition command that validates a full commit into an isolated quarantine without promoting or writing generated Strategist artifacts.

**Architecture:** Keep acquisition in a new `internal/plugins/acquisition` package. The package owns immutable-reference validation, safe Git execution, tar extraction, deterministic digesting, manifest/READY transaction state and stable errors. The Cobra adapter only validates flags, invokes the package and prints the quarantine result. Existing local ingestion and trust verification remain the promotion boundary.

**Tech Stack:** Go 1.27, Cobra, `archive/tar`, `os/exec`, YAML v3, table-driven hermetic tests and local Git fixtures only.

---

### Task 1: Acquisition domain and deterministic manifest

**Files:**
- Create: `internal/plugins/acquisition/errors.go`
- Create: `internal/plugins/acquisition/validate.go`
- Create: `internal/plugins/acquisition/manifest.go`
- Create: `internal/plugins/acquisition/digest.go`
- Test: `internal/plugins/acquisition/*_test.go`

1. Add typed request/result/manifest structures and stable error codes.
2. Add full 40-character SHA, repository URI and package ID validation.
3. Add deterministic tree hashing over sorted relative paths, modes and file bytes.
4. Add JSON manifest encoding with no trust approval claim.
5. Run `GOCACHE=/tmp/strategist-go-cache go test ./internal/plugins/acquisition` and require PASS.

### Task 2: Safe quarantine transaction and Git runner

**Files:**
- Create: `internal/plugins/acquisition/quarantine.go`
- Create: `internal/plugins/acquisition/git.go`
- Test: `internal/plugins/acquisition/*_test.go`

1. Implement temporary quarantine creation outside managed workspace roots.
2. Extract Git archives with containment checks, duplicate rejection and unsafe link rejection.
3. Run Git with argument arrays, `--no-tags`, shallow exact-commit fetch and explicit network authorization.
4. Verify `FETCH_HEAD`/resolved commit equals the requested full SHA.
5. Validate root `SKILL.md` and `strategist.yaml`, write `acquisition.json`, then write `READY` last.
6. Ensure failed or interrupted transactions do not leave a successful marker or partial promoted input.
7. Run focused package tests with network disabled by default.

### Task 3: CLI and command registration

**Files:**
- Create: `cmd/strategist/plugins/acquire_remote.go`
- Create: `cmd/strategist/plugins/acquire_remote_test.go`
- Modify: `cmd/strategist/plugins/plugins.go`
- Modify: `cmd/strategist/plugins/plugins_adapter_test.go`

1. Add `plugins acquire-remote` flags for Git source, repository, full commit, package ID, explicit network opt-in and optional quarantine root.
2. Reject missing network opt-in before any Git process starts.
3. Print quarantine path and manifest digest while never invoking promotion or `prepare-embedded`.
4. Register the command and update command-tree expectations.
5. Test no-network and no-workspace-write behavior with a fake runner or local fixture.

### Task 4: Manual-promotion integration proof

**Files:**
- Test: `internal/plugins/acquisition/*_test.go`
- Test: `internal/install/*_test.go`

1. Build a local bare Git fixture containing a valid Skill-for-Hire package.
2. Acquire it into a temporary quarantine with the fake/local runner and verify manifest plus READY.
3. Copy the reviewed package in the test to a separate `external-skills-source` fixture as the manual promotion step.
4. Run existing ingestion against that fixture and assert acquisition did not write catalog, mirror, lock or `.strategist` files.
5. Run focused tests and `make generate-embedded-check` without modifying generated defaults.

### Task 5: Final validation

1. Run `gofmt` on changed Go files.
2. Run focused acquisition, CLI and ingestion tests with isolated `GOCACHE`.
3. Run `make generate-embedded-check`.
4. Inspect `git diff --stat` and `git status --short` to confirm only scoped new/modified files changed; preserve all pre-existing changes.
