# CODEX Bootstrap Drift Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the CODEX project bootstrap survive clean installs and make `strategist check` expose CODEX bootstrap drift without claiming that the external provider was invoked.

**Architecture:** Keep the existing Claude and Copilot seed behavior unchanged. Extend the compile/awareness source so an existing project `.codex/` directory receives a deterministic `commands.md` seed even when the file is absent; keep the global Codex shim opt-in to an existing Codex home and validate its `skill_root`. Add non-blocking CODEX bootstrap advisories to the existing preflight warnings, while leaving live provider certification exclusively to the existing `strategist-live-evidence/v1` path.

**Tech Stack:** Go, Cobra CLI, `testify`, YAML/JSON contract fixtures, embedded runtime defaults, and the existing conformance/live-evidence packages.

---

## Scope and non-goals

- Fix the source writer and installer tests; do not edit generated or ignored CODEX files as the durable fix.
- Preserve the current catalog/provider pin; this demand does not change the provider package digest.
- Do not create `$HOME/.codex` for users who do not have a CODEX installation. An existing CODEX home remains the opt-in boundary for the global skill shim.
- Do not invoke a live provider from `strategist check`; a missing, unavailable, or failed live probe remains non-certified evidence.
- Do not alter unrelated staged or unstaged work, and do not run `git add`, `git commit`, or other Git state-mutating commands.

## Implementation sequence

### Task 1: Add the failing clean-install CODEX seed test

**Files:**
- Modify: `internal/compile/agent_awareness_test.go`
- Reference: `internal/compile/agent_awareness.go:44-72`

**Step 1: Write the failing test**

Add a subtest under `TestAgentAwareness` that:

1. Creates only `project/.codex/` in a temporary project.
2. Calls `agentAwareness(project)`.
3. Asserts `project/.codex/commands.md` is created.
4. Asserts the generated file contains `## Strategist Runtime Discovery`, `strategist check --json`, and the explicit-invocation/fail-closed instructions.
5. Runs `agentAwareness` a second time and asserts byte-for-byte idempotence.

Keep the existing `no-op when no agent files present` test so the implementation cannot create `.codex/` in repositories without a CODEX surface.

**Step 2: Run the focused test to verify it fails**

Run:

```bash
GOCACHE=/tmp/strategist-go-cache go test ./internal/compile -run 'TestAgentAwareness/(creates codex commands seed|no-op when no agent files present)' -count=1
```

Expected: the new clean-install subtest fails because `agentAwareness` currently skips absent seed files.

### Task 2: Implement source-level CODEX seed materialization

**Files:**
- Modify: `internal/compile/agent_awareness.go:44-72`
- Test: `internal/compile/agent_awareness_test.go`

**Step 1: Add an explicit seed policy**

Replace the anonymous seed-target shape with a named policy that distinguishes:

- Claude and Copilot: update only when their seed file already exists.
- CODEX: update an existing file, or create `commands.md` only when the project `.codex/` directory already exists.

Do not make the loop create all supported-agent files. The CODEX parent-directory check is the opt-in signal.

**Step 2: Add the minimal helper**

Implement a helper with this behavior:

```go
func upsertSeed(path string, createWhenParentExists bool, update func(string) error, label string) {
    info, err := os.Stat(path)
    switch {
    case err == nil && !info.IsDir():
        // continue to update the existing file
    case errors.Is(err, os.ErrNotExist) && createWhenParentExists:
        if parentInfo, parentErr := os.Stat(filepath.Dir(path)); parentErr != nil || !parentInfo.IsDir() {
            return
        }
        if err := writeFile(path, nil, label); err != nil {
            slog.Warn("[Strategist] agent awareness: "+label+" create failed", "error", err)
            return
        }
    case errors.Is(err, os.ErrNotExist):
        return
    default:
        slog.Warn("[Strategist] agent awareness: "+label+" stat failed", "error", err)
        return
    }
    if err := update(path); err != nil {
        slog.Warn("[Strategist] agent awareness: "+label+" update failed", "error", err)
    }
}
```

Use the repository's existing wrapped `writeFile` helper and preserve the non-blocking awareness contract. Add the required `errors` import and update the function comment to describe the CODEX exception precisely.

**Step 3: Run the focused tests**

Run:

```bash
GOCACHE=/tmp/strategist-go-cache go test ./internal/compile -run 'TestAgentAwareness|TestRefreshAgentAwareness' -count=1
```

Expected: PASS, including the existing Claude, Copilot, idempotence, and failure-isolation cases.

### Task 3: Lock the global CODEX shim contract at the installer boundary

**Files:**
- Modify: `internal/install/installer_shim_whitebox_test.go`
- Inspect/modify only if required by the test: `internal/install/shim.go:107-147`, `internal/install/installer_shim_step.go:107-132`

**Step 1: Add a regression test for the effective `skill_root`**

Create an existing temporary `$HOME/.codex` directory, call `installOptionalShims` with a temporary project skill root, read `.codex/skills/strategist/SKILL.md`, and assert:

- the file exists;
- the frontmatter contains the absolute temporary `.strategist` path;
- the body contains the current skill content;
- a second install produces identical bytes.

This protects the source-generated shim from pointing at a stale workspace after reinstall.

**Step 2: Preserve the optional-home boundary**

Keep the existing test proving that absent `.codex` and `.gemini` directories are not created. If implementation changes are necessary, make them limited to structured, actionable warning context for a failed existing Codex shim; do not make a missing Codex home fatal and do not create it implicitly.

**Step 3: Run installer tests**

Run:

```bash
GOCACHE=/tmp/strategist-go-cache go test ./internal/install -run 'TestInstallOptionalShims|TestInstallShim' -count=1
```

Expected: PASS, with the current best-effort semantics preserved for optional shims.

### Task 4: Surface CODEX project-bootstrap drift as non-blocking preflight evidence

**Files:**
- Create or modify: `internal/check/check_codex_bootstrap.go`
- Modify: `internal/check/check_preflight_advisories.go`
- Modify: `internal/check/check_preflight_json.go` only to include the new advisory source if it is not centralized in `preflightAdvisories`
- Test: `internal/check/check_preflight_advisories_test.go`
- Reference: `internal/domain/preflight_result.go`

**Step 1: Define the advisory cases**

Add a pure, read-only diagnostic for the project containing the selected `.strategist` root:

- no `.codex/` directory: no CODEX advisory;
- `.codex/` exists but `commands.md` is absent: `codex_bootstrap_missing`;
- `commands.md` exists but lacks the generated Strategist Runtime Discovery marker: `codex_bootstrap_stale`;
- the file cannot be read: `codex_bootstrap_unreadable`.

The diagnostic must never execute `codex`, inspect provider payloads, or change `PreflightResult.Status`. It is static/bootstrap evidence only.

**Step 2: Wire it into existing advisory aggregation**

Append the diagnostic to `preflightAdvisories(root)` so `strategist check --json` and the human-readable check share the same reason. Keep existing advisory semantics: warnings are visible, but non-blocking advisories do not turn a valid runtime into a failed live-provider certification.

**Step 3: Write the failing/readiness tests**

Extend `internal/check/check_preflight_advisories_test.go` with hermetic temporary-root cases for missing, stale, unreadable, and absent `.codex`. Assert the reason code and assert that the resulting status remains `ready` when no independent blocking warning exists.

**Step 4: Run the focused check tests**

Run:

```bash
GOCACHE=/tmp/strategist-go-cache go test ./internal/check -run 'TestCheckCmd_JSON_Advisories|Codex|Preflight' -count=1
```

Expected: PASS with no provider invocation and no workspace mutation.

### Task 5: Document the CODEX recovery and evidence boundary

**Files:**
- Modify: `docs/cli-reference.md` near the `check` section
- Modify: `docs/testing/client-conformance.md`
- Modify: `docs/runbooks/role-invocation-failed.md`

**Step 1: Document the preflight output**

Explain the new `codex_bootstrap_missing`, `codex_bootstrap_stale`, and `codex_bootstrap_unreadable` advisories, their repair path (`strategist compile` or reinstall), and that they describe project bootstrap only.

**Step 2: Document the live boundary**

State that successful CLAUDE invocation does not certify CODEX live invocation, `strategist check` does not invoke external agents, and the CODEX live matrix row remains pending until an authorized bounded runner emits valid `strategist-live-evidence/v1`.

**Step 3: Review documentation for stale claims**

Remove or correct any wording that equates static readiness, bootstrap presence, and live provider certification.

### Task 6: Validate reinstall, generated assets, and full regression

**Files:**
- No additional source files expected; inspect the final diff for generated-file drift.

**Step 1: Run package-level tests**

```bash
GOCACHE=/tmp/strategist-go-cache go test ./internal/compile ./internal/install ./internal/check ./internal/conformance ./tests/conformance -count=1
```

Expected: PASS.

**Step 2: Verify embedded assets remain synchronized**

```bash
GOCACHE=/tmp/strategist-go-cache go run ./cmd/strategist plugins prepare-embedded --check
```

Expected: `prepare-embedded --check: no drift`.

**Step 3: Run the full Go regression suite**

```bash
GOCACHE=/tmp/strategist-go-cache go test ./... -count=1
```

Expected: PASS. Any failure in pre-existing modified tests must be reported separately and not fixed opportunistically.

**Step 4: Run lint on changed Go packages**

```bash
GOCACHE=/tmp/strategist-go-cache GOLANGCI_LINT_CACHE=/tmp/golangci-lint-cache $(go env GOPATH)/bin/golangci-lint run ./internal/compile ./internal/install ./internal/check
```

Expected: zero new lint findings.

**Step 5: Perform a clean-install smoke check**

Use temporary project and home directories. Create only the project `.codex/` directory and a separate existing home `.codex/` directory, run the source-built install, and verify:

- project `.codex/commands.md` is materialized from the source writer;
- project `.codex/commands.md` contains the generated section;
- home `.codex/skills/strategist/SKILL.md` points to the temporary project `.strategist` root;
- rerunning install is idempotent;
- no provider invocation is attempted by `strategist check --json`.

Record live CODEX evidence as unavailable/pending if the environment cannot provide an authorized runner; do not mark `codex-live-probe` certified from static output.

## Completion criteria

- A clean install with an existing project `.codex/` creates and refreshes `commands.md` from source.
- A project without `.codex/` is not mutated merely because Strategist was installed.
- Reinstall preserves the correct CODEX shim `skill_root` and is idempotent.
- `strategist check` reports CODEX bootstrap drift with actionable reason codes without invoking a provider.
- CLAUDE behavior remains green.
- Existing catalog digest/pin remains unchanged.
- Focused tests, full tests, embedded drift check, and lint pass.
- Live CODEX certification remains explicitly separate and is not claimed without authorized evidence.

## Delivery note

The plan document is saved in the workspace. Per local M010 governance, Git state changes are not performed automatically; review and commit remain a separate human-authorized action.
