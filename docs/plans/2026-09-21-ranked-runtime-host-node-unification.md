# Ranked Runtime Host-Node Unification Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make every Ranked OpenSpec installation use the embedded OpenSpec bundle with a validated host Node `>=20.19.0`, independent of whether Strategist came from a release archive or `go install`.

**Architecture:** Remove the target-specific private-Node payload and retain one OpenSpec-only bundle materializer. The installer resolves and validates one absolute host Node path, records it with the digest-verified private bundle, and `strategist check` consumes that exact state through the same healthcheck semantics.

**Tech Stack:** Go, Cobra CLI, Go `embed`, `io/fs`, Node/OpenSpec bundle, GoReleaser, GitHub Actions, Git Bash Windows smoke tests.

---

## Guardrails

- Preserve unrelated staged and unstaged work; do not reset, clean, add, commit, tag, or publish it.
- This plan intentionally removes support for a client without Node. The supported host prerequisite is Node `>=20.19.0`.
- No runtime command may locate `openspec`, npm, or a Node executable through a provider `PATH`. The installer may resolve host `node` once, then stores an absolute path.
- Treat `ranked-runtimes.yaml` records with `mode: payload` as incompatible and require explicit upgrade/reinstall. Do not reinterpret them.
- Run each Go test with an isolated `GOCACHE` such as `/tmp/strategist-gocache`.

### Task 1: Define the normalized host-Node runtime state

**Files:**
- Modify: `internal/install/ranked_runtime_bootstrap.go`
- Modify: `internal/install/ranked_runtime_private.go`
- Modify: `internal/install/ranked_runtime_state.go`
- Modify: `internal/install/ranked_runtime_private_test.go`
- Modify: `internal/check/check_ranked_readiness.go`
- Test: `internal/check/check_ranked_readiness_test.go`

**Step 1: Write failing state-validation tests.**

Add cases that parse `ranked-runtimes.yaml` with a Ranked OpenSpec entry and require:

- `runtime.node` is absolute;
- `runtime.script` is clean, relative, and under `weapon-runtime/`;
- no runtime mode field is required for a new record;
- legacy `mode: payload` is rejected with the existing actionable reinstall/upgrade path.

Keep the fixture role, slot, provider, and certification digest equal to the catalog contract so failures isolate the runtime shape.

**Step 2: Run the focused tests and verify failure.**

Run:

```bash
GOCACHE=/tmp/strategist-gocache go test ./internal/check ./internal/install -run 'Test.*RankedRuntime.*(Host|Payload|State)'
```

Expected: the new assertions fail because the current state supports both `payload` and `host_node` modes and records payload-relative Node paths.

**Step 3: Simplify the state model.**

Replace the mode-dependent `rankedRuntimeStateRuntime` / `rankedRuntimeStatePrivate` behavior with the normalized record:

```go
type rankedRuntimeStateRuntime struct {
    Node       string // absolute host executable
    Script     string // weapon-runtime/... relative to .strategist
    Components []rankedRuntimeStateComponent
}
```

At the install boundary, canonicalize the result of `exec.LookPath("node")` with `filepath.Abs` before it can be persisted. Preserve the OpenSpec component evidence (version and digest). Reject legacy `payload` explicitly during state interpretation rather than silently accepting it.

**Step 4: Re-run focused tests.**

Run the command from Step 2.

Expected: PASS; each invalid state produces a cataloged blocked reason and the normalized state accepts an absolute Node plus private bundle script.

**Step 5: Checkpoint.**

Review only the files listed above with `git diff --check`. Create a commit only after explicit user authorization.

### Task 2: Make OpenSpec-only materialization the sole runtime path

**Files:**
- Modify: `internal/install/ranked_runtime_private.go`
- Modify: `internal/runtimepayload/embedded_build.go`
- Modify: `internal/runtimepayload/manifest.go`
- Modify: `internal/runtimepayload/embedded_build_test.go`
- Modify: `internal/runtimepayload/runtimepayload_test.go`
- Delete: `internal/runtimepayload/bundled/embed_payload_darwin_amd64.go`
- Delete: `internal/runtimepayload/bundled/embed_payload_darwin_arm64.go`
- Delete: `internal/runtimepayload/bundled/embed_payload_linux_amd64.go`
- Delete: `internal/runtimepayload/bundled/embed_payload_linux_arm64.go`
- Delete: `internal/runtimepayload/bundled/embed_payload_windows_amd64.go`
- Delete: `internal/runtimepayload/bundled/embed_payload_windows_arm64.go`
- Delete: `internal/runtimepayload/bundled/nodepayload/`
- Test: `internal/runtimepayload/embedded_runtime_test.go`

**Step 1: Write failing OpenSpec-only materialization tests.**

Add tests proving the materializer copies and verifies only the OpenSpec tree, returns the private `openspec.mjs` launcher, and never expects a Node component or target-specific archive. Test malformed/missing/tampered OpenSpec tree behavior remains fail-closed.

**Step 2: Run the focused tests and verify failure.**

Run:

```bash
GOCACHE=/tmp/strategist-gocache go test ./internal/runtimepayload -run 'Test(BuildEmbedded|MaterializeOpenSpec|EmbeddedRuntime)'
```

Expected: FAIL for tests that assert no Node component and no target selection.

**Step 3: Remove target payload composition.**

Delete `BuildEmbedded`, `RegisterEmbedded`, `nodeInfo`, `runtimeLock`, layered payload filesystem support, and Node launcher fields that exist only for the private-Node branch. Retain a narrowly named OpenSpec bundle manifest/verification API that reads `runtime.build.yaml`, verifies the tree digest, stages extraction, and returns the launcher path.

`resolveRankedExecutable` must always call the host-Node resolver and `MaterializeOpenSpec`; remove `runtimepayload.Default()` branching and payload pin checks.

**Step 4: Re-run focused tests.**

Run the command from Step 2, then:

```bash
GOCACHE=/tmp/strategist-gocache go test ./internal/install -run 'Test.*Ranked.*Runtime'
```

Expected: PASS; no test fixture needs Node archives or a target payload.

**Step 5: Checkpoint.**

Run `git diff --check`. Do not commit without authorization.

### Task 3: Unify install and check around the same real provider invocation

**Files:**
- Modify: `internal/install/ranked_runtime_command.go`
- Modify: `internal/install/ranked_runtime_bootstrap.go`
- Modify: `internal/install/ranked_runtime_private.go`
- Modify: `internal/check/check_ranked_readiness.go`
- Modify: `internal/check/check_ranked_host_node.go`
- Delete: `internal/check/check_ranked_private.go`
- Modify: `internal/check/check_ranked_readiness_test.go`
- Test: `internal/install/ranked_runtime_bootstrap_test.go`
- Test: `internal/install/ranked_runtime_private_test.go`

**Step 1: Write parity tests first.**

Create shared fixtures for a valid absolute fake Node and private `openspec.mjs` script. For each condition below, assert install preparation and `strategist check` reach the same reason class:

- missing Node;
- Node exits while reading `--version`;
- Node version below `20.19.0`;
- missing/tampered private OpenSpec script;
- `context --json` error;
- semantic OpenSpec root mismatch.

**Step 2: Run the parity tests and verify failure.**

Run:

```bash
GOCACHE=/tmp/strategist-gocache go test ./internal/install ./internal/check -run 'Test.*Ranked.*(Parity|Healthcheck|HostNode)'
```

Expected: FAIL where installer-only and check-only branches report different behavior or use distinct fixtures.

**Step 3: Extract one internal validation seam.**

Make installer bootstrap validation and checker validation call the same host-Node/private-script invocation helper. The helper must receive an absolute Node path and a validated private script path; it must never use a bare executable name or provider `PATH` lookup. Keep bootstrap (`init`) installer-only, but use the shared `context --json` validation immediately after bootstrap and during check.

Delete the private-payload healthcheck branch and its tests after replacement coverage exists. Preserve the existing detailed reason-code distinction for missing Node, unsupported version, command execution failure, and semantic root mismatch.

**Step 4: Re-run parity and package tests.**

Run:

```bash
GOCACHE=/tmp/strategist-gocache go test ./internal/install ./internal/check
```

Expected: PASS; the valid fixture proves a private OpenSpec bundle runs under host Node with no `openspec` lookup.

**Step 5: Checkpoint.**

Run `git diff --check`. Do not commit without authorization.

### Task 4: Make Wizard activation transactional and explain the prerequisite

**Files:**
- Modify: `internal/install/installer.go`
- Modify: `internal/install/wizard.go`
- Modify: `internal/install/wizard_prompts_ranked.go`
- Modify: `internal/install/wizard_prompts_slots.go`
- Modify: `internal/install/ranked_binding_activation.go`
- Modify: `internal/install/installer_wizard_whitebox_test.go`
- Modify: `internal/install/wizard_prompts_test.go`
- Test: `tests/integration/hermetic_install_test.go`
- Modify: `internal/embed/defaults/contracts/machine/errors.yaml`
- Modify: `.strategist/contracts/machine/errors.yaml`

**Step 1: Write failing Wizard/install tests.**

Add a Ranked refinement selection fixture that has a certified `archivist -> openspec-propose` binding but an absent or outdated Node. Assert failure leaves no active Ranked record in `plugins.lock` or `ranked-runtimes.yaml`. Add a successful fixture proving the prompt labels Ranked as using host Node `>=20.19.0` and persists the normalized state only after `context --json` succeeds.

**Step 2: Run the focused tests and verify failure.**

Run:

```bash
GOCACHE=/tmp/strategist-gocache go test ./internal/install -run 'Test(Install_Wizard|Wizard.*Ranked|PrepareRanked)'
```

Expected: FAIL where `plugins.lock` becomes active before runtime preparation has passed.

**Step 3: Reorder activation atomically.**

Keep the Wizard's explicit `ROLE -> WEAPON` and `Ranked|Custom` choice. Remove any payload/runtime choice from UI wording. Stage the candidate lock and runtime state, complete Node probe/materialization/bootstrap/healthcheck, then atomically expose both records. Ensure the existing install transaction rolls back all staged runtime/binding artifacts on any failure.

Add/adjust cataloged messages for host Node missing, below minimum, private bundle missing/tampered, and real invocation failure. Mirror normative changes from `internal/embed/defaults/` into `.strategist/` with the repository's generation/parity workflow; do not hand-edit one side only.

**Step 4: Re-run focused and hermetic integration tests.**

Run:

```bash
GOCACHE=/tmp/strategist-gocache go test ./internal/install
GOCACHE=/tmp/strategist-gocache go test -tags=integration ./tests/integration -run 'WizardRanked|HermeticInstall'
```

Expected: PASS; failed prerequisites leave no activation state, successful Ranked installation stores the checked state.

**Step 5: Checkpoint.**

Run `git diff --check`. Do not commit without authorization.

### Task 5: Simplify build, version, release, and smoke paths

**Files:**
- Delete: `cmd/strategist/payload_bundled.go`
- Modify: `cmd/strategist/version.go`
- Modify: `cmd/strategist/version_test.go`
- Modify: `Makefile`
- Modify: `make/release.mk`
- Modify: `.goreleaser.yaml`
- Modify: `scripts/smoke-standalone-install.sh`
- Delete: `scripts/fetch-node-runtime.py`
- Modify: `scripts/check-reproducible-build.sh`
- Modify: `.github/workflows/test.yml`
- Modify: `.github/workflows/release.yml`

**Step 1: Write failing build-contract tests.**

Change version tests to require a single build line such as:

```text
runtime payload: embedded OpenSpec 1.13.0; host Node >=20.19.0 required
```

Add a repository-level assertion (shell or Go test, following current conventions) that release flags and Make targets do not contain `strategist_payload`, `fetch-node-runtime.py`, or `nodepayload`.

**Step 2: Run the focused tests and verify failure.**

Run:

```bash
GOCACHE=/tmp/strategist-gocache go test ./cmd/strategist ./internal/runtimepayload
rg -n 'strategist_payload|fetch-node-runtime|nodepayload' Makefile make .goreleaser.yaml .github scripts cmd internal/runtimepayload/bundled
```

Expected: the source search still finds the retired branch before cleanup.

**Step 3: Remove build variants.**

Make `make build`, `make install`, release archives, and `go install` compile the same binary without build tags or Node fetching. Update GoReleaser to remove `-tags=strategist_payload` and its pre-build Node fetch. Replace the empty-PATH standalone smoke with a hermetic environment containing only a fake/validated host Node and no OpenSpec/npm; assert install and check both use the private bundle.

Keep the OpenSpec bundle digest/`plugins prepare-embedded --check` release gate and reproducibility verification, but make them platform-independent.

**Step 4: Run build and smoke validation.**

Run:

```bash
GOCACHE=/tmp/strategist-gocache go build -trimpath ./cmd/strategist
GOCACHE=/tmp/strategist-gocache go test ./cmd/strategist ./internal/runtimepayload
bash scripts/smoke-standalone-install.sh
```

Expected: PASS; the smoke has Node available, has no global OpenSpec/npm, and reaches `check --json` status `ready`.

**Step 5: Checkpoint.**

Run `git diff --check`. Do not commit without authorization.

### Task 6: Validate cross-platform behavior and documentation

**Files:**
- Modify: `README.md`
- Modify: `docs/runbooks/standalone-runtime-hermeticity.md`
- Modify: `docs/onboarding/readme-en.md`
- Modify: `docs/test-styles.md` only if the test-style guidance names the retired payload flow
- Modify: `docs/releases/v1.0.21-preparation.md` only if this work is included in that release scope
- Test: `tests/integration/hermetic_install_test.go`

**Step 1: Add Linux and Windows acceptance cases.**

Extend hermetic installation coverage so both platforms assert: Node `>=20.19.0` is accepted; an older/absent Node blocks activation; `openspec` is absent from PATH; and a successful check invokes the private bundle. On Windows, preserve required startup variables while proving `USERPROFILE`, npm, and OpenSpec configuration remain private.

**Step 2: Run tests and verify baseline behavior.**

Run:

```bash
GOCACHE=/tmp/strategist-gocache go test -tags=integration ./tests/integration
GOCACHE=/tmp/strategist-gocache GOOS=windows go vet ./...
```

Expected: the new tests initially expose any platform-specific state/launcher assumption.

**Step 3: Update user-facing contracts.**

Document one installation prerequisite and one runtime diagnosis path: Node `>=20.19.0` is required; Strategist owns the OpenSpec bundle; users do not install OpenSpec with npm; `strategist version --build` and `strategist check --json` show the exact remediation. Remove all claims that releases contain a private Node or work with an empty Node PATH.

**Step 4: Run the full quality and parity suite.**

Run:

```bash
GOCACHE=/tmp/strategist-gocache go test ./...
GOCACHE=/tmp/strategist-gocache go test -tags=integration ./tests/integration/...
make embed-skills-check
make fmt-check
make lint
make quality-budget-gate
make docs-generated-gate
make docs-links-gate
git diff --check
```

Expected: all gates pass; source/default/runtime contract mirrors remain synchronized.

**Step 5: Final checkpoint.**

Confirm the change set contains no retired Node payload assets or build tags and that documentation matches the install/check behavior. Commit, tag, and publish only with explicit authorization.

## Rollout and rollback

- Ship this as a compatibility-breaking installation change with a clear release note: existing Ranked installations must run the explicit upgrade/reinstall flow and have host Node `>=20.19.0`.
- Rollback means restoring the prior release binary; do not mutate an existing normalized runtime record into a legacy payload record.
- Capture Windows and `go install` evidence separately in the release checklist before publication.
