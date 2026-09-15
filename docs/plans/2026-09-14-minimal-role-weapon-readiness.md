# Minimal Role Weapon Readiness Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Remove legacy fallback/readiness behavior and make the minimum role-to-weapon contract the sole fatal gate for Wizard and `strategist check`.

**Architecture:** Keep the immutable pipeline and fixed role contracts as the authority. Validate only the minimum operational contract—role, weapon, affinity, risk, manifest, checkpoint, and persisted binding—through one shared validator. Report advanced trust, dependency, grant, connector, and observation controls as deferred advisory dimensions until a future governance profile makes them mandatory.

**Tech Stack:** Go, YAML (`gopkg.in/yaml.v3`), Cobra CLI, existing `PluginLockFile`/`SlotBinding`, table-driven tests.

---

## Task 1: Capture the new readiness contract with failing tests

**Files:**
- Modify: `internal/check/check_test.go`
- Modify: `internal/check/check_slots_test.go`
- Create: `internal/rolevalidation/role_weapon_test.go`

1. Add a valid runtime fixture with Ranger→brainstorming and Archivist→openspec-propose bindings.
2. Assert `strategist check` returns success and emits `ROLE READINESS`/`binding=valid` for the two configurable roles.
3. Assert the output contains no `fallback=` or `always_native_no_policy` fields.
4. Add missing-binding, stale-binding, duplicate-binding, wrong-role, and invalid-manifest cases.
5. Add a test proving `unknown`/`unsupported` advanced dimensions do not invalidate a valid minimum binding.
6. Run:

```bash
GOCACHE=/tmp/go-cache go test ./internal/check ./internal/rolevalidation -count=1
```

Expected: new assertions fail against the legacy fallback/readiness output.

## Task 2: Define explicit minimum and deferred readiness semantics

**Files:**
- Modify: `internal/domain/plugin_readiness.go`
- Create: `internal/domain/plugin_readiness_test.go`
- Modify: `internal/check/check_readiness.go`
- Modify: `internal/check/check_slots_test.go`

1. Add `ReadinessDeferred` or an equivalent explicit advisory state; do not map `unknown` or `unsupported` to `ready`.
2. Define the minimum gate as role validity, weapon manifest, risk, affinity, binding, lock parity, and fixed checkpoint availability.
3. Keep trust, dependency, granular grant, connector, and observation dimensions outside the minimum gate and render them as deferred/advisory.
4. Add tests proving minimum readiness is true only with a valid role/weapon contract and that deferred dimensions remain visible without causing a fatal result.
5. Run:

```bash
GOCACHE=/tmp/go-cache go test ./internal/domain ./internal/check -run 'Test.*Readiness|Test.*RoleWeapon' -count=1
```

Expected: PASS.

## Task 3: Make the shared validator the only role/weapon fatal gate

**Files:**
- Modify: `internal/rolevalidation/role_weapon.go`
- Modify: `internal/rolevalidation/role_weapon_test.go`
- Modify: `internal/domain/role_provider_contract.go` only if the minimum contract needs a domain helper

1. Ensure the validator checks both mandatory slots: discovery→ranger and refinement→archivist.
2. Require exactly one active provider and exactly one matching persisted binding for each slot.
3. Validate role files, provider manifests, risk scores, explicit `roles` affinity, and legacy `canonical_role` migration behavior.
4. Return actionable failures containing slot, role, provider, and dimension.
5. Treat absent or malformed `.strategist/plugins.lock` as fatal for mandatory roles.
6. Add tests for every failure dimension and for native execution remaining outside the external weapon requirement.
7. Run:

```bash
GOCACHE=/tmp/go-cache go test ./internal/rolevalidation -count=1
```

Expected: PASS.

## Task 4: Remove native fallback resolution from static check

**Files:**
- Modify: `internal/check/check.go`
- Modify: `internal/check/check_slots.go`
- Modify: `internal/check/check_output.go`
- Delete or isolate: `internal/check/resolveNativeFallback` and related fallback outcome helpers
- Modify: `internal/check/check_slots_test.go`

1. Remove all static calls that calculate a native fallback after successful skill-provider resolution.
2. Remove `fallbackProvider`/`fallbackPath` from static resolution state if no runtime caller remains.
3. Remove `fallback=` and `outcome=always_native_no_policy` from successful check output.
4. Invoke the shared minimum validator and propagate failures as fatal check errors.
5. Render explicit role/weapon readiness and deferred advisory sections.
6. Remove or rewrite tests that assert native fallback discovery or automatic substitution.
7. Run:

```bash
GOCACHE=/tmp/go-cache go test ./internal/check -count=1
```

Expected: PASS; no test should require fallback output.

## Task 5: Remove legacy fallback policy from runtime contracts

**Files:**
- Modify: `internal/domain/resolution_policy.go` and related policy files
- Modify: `internal/check/check_output.go`
- Modify: `internal/embed/defaults/contracts/narrative/00-routing.md`
- Modify: `internal/embed/defaults/contracts/machine/provider-fallback.yaml`
- Modify: `docs/adr/0028-native-role-resilient-baseline.md` only where it describes active fallback behavior
- Modify: fallback-related tests under `internal/domain`, `internal/check`, and `internal/plugins`

1. Search all production callers of `resolveNativeFallback`, `DecideSlotFallbackOutcome`, `fallbackProvider`, and `native_fallback`.
2. Remove behavior that substitutes a native role for a missing or failed weapon.
3. Replace legacy policy descriptions with the invariant: a role without a valid weapon binding is fatal.
4. Preserve only diagnostic/error vocabulary needed for migration, without an executable fallback path.
5. Add a regression test proving the persisted binding is never rewritten to a native provider.
6. Run:

```bash
rg -n 'resolveNativeFallback|fallbackProvider|fallbackPath|always_native_no_policy|native_fallback|fallback=' internal docs/adr
GOCACHE=/tmp/go-cache go test ./internal/domain ./internal/check ./internal/plugins -count=1
```

Expected: no legacy fallback caller remains in the static or runtime path covered by this change.

## Task 6: Enforce the minimum contract in the Wizard

**Files:**
- Modify: `internal/install/wizard.go`
- Modify: `internal/install/installer_config.go`
- Modify: `internal/install/role_provider_migration.go`
- Modify: `internal/install/installer_wizard_whitebox_test.go`
- Modify: `internal/install/installer_config_whitebox_test.go`

1. Keep role-affinity filtering and exclusion reasons in the prompts.
2. Block completion when either mandatory role has no compatible resolved weapon.
3. Block partial migrations instead of silently retaining native defaults.
4. Validate the generated `PluginLockFile` has exactly one binding for each mandatory role and matches the selected provider.
5. Treat the Wizard binding as the current operator authorization; do not require future granular grants.
6. Add tests for missing weapon, incompatible weapon, ambiguous weapon, incomplete lock, and successful two-weapon installation.
7. Run:

```bash
GOCACHE=/tmp/go-cache go test ./internal/install -run 'Test.*(Wizard|Config|RoleProvider)' -count=1
```

Expected: PASS.

## Task 7: Update output, manifests, and documentation

**Files:**
- Modify: `internal/check/check.go`
- Modify: `internal/check/check_output.go`
- Modify: `internal/check/check_readiness.go`
- Modify: `internal/embed/defaults/contracts/narrative/03-discovery.md`
- Modify: `internal/embed/defaults/contracts/narrative/04-refinement.md`
- Modify: `docs/adr/0034-role-and-skill-taxonomy.md`
- Modify: `docs/adr/0035-embedded-weapon-fallback-policy.md`
- Modify: `docs/adr/0037-wizard-role-binding-persistence.md`

1. Document that Ranger/Archivist own normalization and checkpoints while weapons remain flexible inputs.
2. Document that a missing role weapon is fatal and there is no fallback provider.
3. Define the output distinction between `ROLE READINESS` and `ADVISORY` dimensions.
4. Remove claims that discovery is always native or that a native role is automatically substituted.
5. Ensure generated skill manifests carry `roles` affinity and the Wizard binding remains inspectable.
6. Run:

```bash
rg -n 'native-only|always resolves|never consulted|always_native_no_policy|fallback=ranger|fallback=archivist' docs/adr internal/embed/defaults/contracts internal/check
```

Expected: no contradictory legacy semantics remain.

## Task 8: Regenerate and verify the complete system

**Files:**
- Modify generated files only through the embedded preparation command
- Test: `internal/install`, `internal/check`, `internal/rolevalidation`, and full repository suite

1. Regenerate embedded catalog, mirrors, and source lock:

```bash
GOCACHE=/tmp/go-cache go run ./cmd/strategist plugins prepare-embedded
```

2. Verify generated drift:

```bash
GOCACHE=/tmp/go-cache go run ./cmd/strategist plugins prepare-embedded --check
```

3. Run focused tests:

```bash
GOCACHE=/tmp/go-cache go test ./internal/rolevalidation ./internal/install ./internal/check ./internal/domain ./internal/plugins -count=1
```

4. Run the full suite with writable isolated environment:

```bash
HOME=/tmp GOMODCACHE=/home/sergio/go/pkg/mod GOCACHE=/tmp/go-cache go test ./... -count=1
```

5. Run the real check from the repository root and verify valid bindings are `ready`, advanced controls are advisory/deferred, and no fallback line appears:

```bash
GOCACHE=/tmp/go-cache go run ./cmd/strategist check
```

6. Run `git diff --check` and review `git status --short`. Preserve existing user changes; do not reset, clean, stage, commit, or bypass hooks without explicit authorization.

Expected: valid role/weapon bindings allow the pipeline to proceed, missing bindings fail fatally, and no legacy fallback behavior remains.
