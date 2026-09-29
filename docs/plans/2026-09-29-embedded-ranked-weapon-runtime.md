# Embedded Ranked Weapon Runtime Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the embedded `brainstorming` Weapon use Strategist-owned runtime semantics in both Codex and Claude, and reject external runtime kinds for Ranked bindings.

**Architecture:** The canonical `external-skills-source` adapter will declare `runtime.kind: embedded`; generation will propagate that declaration to the embedded catalog, lock, and installed `.strategist` mirror. The connector boundary will carry the runtime kind so embedded invocation evidence is accepted without a host-issued receipt, while host invocation remains receipt-authenticated. Ranked readiness will fail closed for host and executable runtime kinds.

**Tech Stack:** Go, YAML manifests/catalogs, embedded defaults generator, `go test`, Strategist CLI checks.

---

### Task 1: Change the canonical brainstorming runtime contract

**Files:**
- Modify: `external-skills-source/brainstorming/strategist.yaml`
- Generate: `internal/embed/defaults/skills/brainstorming/strategist.yaml`, `internal/embed/defaults/plugins/catalog.yaml`, `external-skills-source.lock.yaml`
- Generate: `.strategist/skills/brainstorming/strategist.yaml`, `.strategist/plugins/catalog.yaml`, `.strategist/plugins.lock`

**Step 1: Write the failing contract assertion**

Extend the existing embedded catalog/ingestion coverage to assert that the
Ranked `brainstorming` entry has `runtime.kind: embedded`, no `host_api`, and
retains `origin: embedded` and its Ranger affinity.

**Step 2: Run the focused test to verify it fails**

Run: `rtk go test ./internal/install ./internal/embed`

Expected: failure showing the current `brainstorming` runtime is `host`.

**Step 3: Write the minimal source change**

Change only the canonical adapter runtime from `host` plus `host_api` to
`embedded`; keep the existing weapon contract and provenance unchanged.

**Step 4: Regenerate the governed artifacts**

Run: `rtk make embed-skills`

Expected: catalog, embedded lock, and generated defaults update consistently;
no hand-edits to `.strategist` are made.

**Step 5: Run the focused tests**

Run: `rtk go test ./internal/install ./internal/embed`

Expected: PASS, including source/runtime parity and baseline roster checks.

### Task 2: Carry runtime kind through the invocation boundary

**Files:**
- Modify: `internal/plugins/connectors/runtime_connector.go`
- Modify: `internal/plugins/connectors/embedded_weapon_connector.go`
- Modify: `internal/plugins/connectors/host_weapon_connector.go`
- Modify: `internal/provider/discovery.go`
- Modify: `internal/provider/discovery_connector.go`
- Test: `internal/provider/discovery_connector_test.go`, `internal/provider/discovery_test.go`

**Step 1: Write failing embedded-path tests**

Add a connector test where `EmbeddedWeaponConnector` returns internal
invocation evidence without a host receipt; assert Ranger normalization
succeeds. Add a host-path regression asserting that a host result without a
valid receipt still fails with `role_invocation_failed`.

**Step 2: Run the focused tests to verify the embedded case fails**

Run: `rtk go test ./internal/provider ./internal/plugins/connectors`

Expected: the embedded case fails because normalization currently validates a
host receipt unconditionally.

**Step 3: Implement the runtime-kind signal**

Add a runtime-kind field to connector capabilities and the discovery request.
Set it to `embedded` for `EmbeddedWeaponConnector` and `host` for
`HostWeaponConnector`; pass it into Ranger normalization.

**Step 4: Make receipt validation runtime-specific**

For `embedded`, require non-empty internal invocation evidence and record an
internal/non-host verification state without accepting or fabricating a host
receipt. For `host` and other external invocation kinds, retain mandatory
receipt validation, identity binding, freshness, nonce replay protection, and
catalog pin checks.

**Step 5: Run the focused tests**

Run: `rtk go test ./internal/provider ./internal/plugins/connectors`

Expected: embedded invocation passes; host invocation without a valid receipt
remains fail-closed.

### Task 3: Reject external runtime kinds for Ranked bindings

**Files:**
- Modify: `internal/check/check_ranked_readiness.go`
- Modify: `internal/embed/defaults/contracts/machine/errors.yaml`
- Generate: `.strategist/contracts/machine/errors.yaml`
- Test: `internal/check/check_ranked_readiness_test.go`

**Step 1: Write the failing Ranked guard test**

Add table cases proving a certified Ranked provider with `runtime.kind: host`
or `runtime.kind: executable` is blocked with a dedicated external-runtime
reason, rather than reported as an advisory unknown state.

**Step 2: Run the focused test to verify it fails**

Run: `rtk go test ./internal/check -run RankedRuntime`

Expected: current code returns `ranked_runtime_requires_host_invocation` with
an unknown status.

**Step 3: Implement the fail-closed guard**

Return a blocked readiness result for host/executable Ranked runtimes with a
cataloged reason such as `ranked_external_runtime_forbidden`; keep embedded
and `openspec_root` runtime handling unchanged.

**Step 4: Regenerate the runtime contract mirror**

Run the sanctioned runtime generation/upgrade path so the machine error
contract matches `internal/embed/defaults/contracts/machine/errors.yaml`.

**Step 5: Run the focused tests**

Run: `rtk go test ./internal/check -run RankedRuntime`

Expected: host and executable Ranked candidates are blocked; embedded and
valid private OpenSpec runtimes retain their existing behavior.

### Task 4: Validate clean installation and both host-facing shims

**Files:**
- Test: `internal/embed/embedded_weapon_roster_test.go`
- Test: `internal/install/embedded_weapon_baseline_roster_test.go`
- Test: relevant provider/connector tests from Tasks 2–3

**Step 1: Run source/runtime drift checks**

Run: `rtk make embed-skills-check`

Expected: no generated drift.

**Step 2: Run the package validation suite**

Run: `rtk go test ./internal/domain ./internal/plugins/connectors ./internal/provider ./internal/check ./internal/install ./internal/embed`

Expected: PASS.

**Step 3: Run Strategist static validation**

Run: `rtk strategist check --json`

Expected: `status: ready`, with `brainstorming` resolved as an embedded
Ranked runtime and no host invocation requirement.

**Step 4: Verify worktree scope**

Run: `rtk git diff --check` and `rtk git status --short`

Expected: only the requested source, generated artifacts, plan, and tests are
changed; existing unrelated modifications remain untouched.

