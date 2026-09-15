# Fatal Role Weapon Readiness Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make missing or invalid discovery/refinement weapon bindings fatal in both the wizard and `strategist check`, while removing static native-fallback noise from valid configurations.

**Architecture:** Add one shared validator over active slot configuration, role contracts, installed skill manifests, and `.strategist/plugins.lock`. The wizard invokes it before success and the check invokes it before printing success. Keep runtime fallback resolution separate: it is only eligible after an actual invocation failure and never replaces a valid persisted binding during static validation.

**Tech Stack:** Go, YAML (`gopkg.in/yaml.v3`), Cobra CLI, existing `domain.SlotBinding`/`PluginLockFile`, table-driven tests.

---

## Task 1: Establish the failing check contract

**Files:** `internal/check/check_test.go`, `internal/check/check_slots_test.go`, `internal/check/check_lock_parity_test.go`

1. Add fixtures with valid `active.yaml`, role files, skill manifests, and matching `.strategist/plugins.lock` bindings for `brainstorming -> ranger` and `openspec-propose -> archivist`.
2. Assert valid external bindings do not produce `fallback=ranger` or `outcome=always_native_no_policy`.
3. Add missing-binding and stale-binding cases and assert errors naming the affected role/slot.
4. Run `GOCACHE=/tmp/go-cache go test ./internal/check -run 'Test.*(Binding|Fallback|Check)' -count=1` and verify the new tests fail against the current behavior.

## Task 2: Create the shared role-weapon validator

**Files:** Create `internal/check/role_weapon_readiness.go` and `internal/check/role_weapon_readiness_test.go`; modify `internal/domain/plugin_state_types.go` only if a binding accessor is needed.

1. Define a result carrying slot, role, provider, validity, and concrete failure reasons.
2. Validate mandatory configurable slots `discovery` and `refinement` using `roles/default.yaml` and the role contract files.
3. Require a selected provider, installed valid manifest, correct slot `risk_score`, and role affinity via `roles` with legacy `canonical_role` fallback.
4. Read `.strategist/plugins.lock`; require exactly one matching binding per mandatory slot and reject missing, duplicate, or stale bindings.
5. Add table-driven tests for valid, missing, malformed, wrong-risk, wrong-role, duplicate, and stale bindings.
6. Run `GOCACHE=/tmp/go-cache go test ./internal/check -run TestRoleWeaponReadiness -count=1`; expected: PASS.

## Task 3: Make `strategist check` fail closed on role readiness

**Files:** `internal/check/check.go:110-155`, `internal/check/check_output.go:69-111`, `internal/check/check_slots.go:122-151`, `internal/check/check_test.go`

1. Invoke the shared validator after loading active slots and append every failure to the check error list.
2. Preserve probe/readiness diagnostics as a separate dimension; do not use them to invent a fallback binding.
3. Stop populating `slotResolution.fallbackProvider` during successful static skill-provider resolution.
4. Remove `fallback=` and `outcome=` from valid static bindings and print explicit `binding=valid` diagnostics.
5. Keep native fallback behind the runtime invocation-failure path only.
6. Run `GOCACHE=/tmp/go-cache go test ./internal/check -count=1`; expected: PASS, including non-zero results for missing/invalid bindings.

## Task 4: Reuse the validator in the wizard

**Files:** `internal/install/wizard.go:141-180`, `internal/install/installer_config.go:63-82`, shared validator package boundary, `internal/install/installer_config_whitebox_test.go`, `internal/install/installer_wizard_whitebox_test.go`

1. Move the validator to a package boundary usable by both `internal/install` and `internal/check`, without duplicating policy.
2. Validate resolved discovery and refinement bindings before writing final wizard state.
3. Validate again after assembling the lock file and before reporting installation success.
4. Abort with concrete role/provider/dimension errors and ensure failure cannot report success or leave an orphan lock.
5. Add wizard rejection tests for missing, mismatched, and stale bindings.
6. Run `GOCACHE=/tmp/go-cache go test ./internal/install -run 'Test.*(Wizard|Config|RoleProvider)' -count=1`; expected: PASS.

## Task 5: Preserve runtime fallback as a post-failure policy

**Files:** `internal/check/check_slots.go`, `internal/domain/role_provider_contract.go`, `internal/check/check_slots_test.go`, `internal/domain/role_provider_contract_test.go`

1. Identify the actual runtime invocation-failure consumer of native fallback policy.
2. Ensure static validation never calls it after successful provider resolution.
3. Ensure a real invocation failure still respects `ask`, `native`, or `block`.
4. Ensure fallback never rewrites `active.yaml` or `.strategist/plugins.lock`.
5. Add tests distinguishing static success from runtime failure.
6. Run `GOCACHE=/tmp/go-cache go test ./internal/check ./internal/domain -run 'Test.*(Fallback|Resolution|Compatibility)' -count=1`; expected: PASS.

## Task 6: Align diagnostics and governing documentation

**Files:** `internal/check/check.go`, `internal/check/check_output.go`, `internal/embed/defaults/contracts/narrative/00-routing.md`, `internal/embed/defaults/contracts/narrative/03-discovery.md`, `internal/embed/defaults/contracts/narrative/04-refinement.md`, `docs/adr/0034-role-and-skill-taxonomy.md`, `docs/adr/0035-embedded-weapon-fallback-policy.md`, `docs/adr/0037-wizard-role-binding-persistence.md`

1. Document that discovery and refinement require valid persisted weapon bindings.
2. Document fixed-role ownership of normalization, handoff, locks, state, and control-log checkpoints.
3. Document native fallback as runtime-only after invocation failure.
4. Make check output distinguish binding validity from probe/readiness diagnostics.
5. Remove stale static-binding claims with `rg -n 'always_native_no_policy|native-only|always resolves|never consulted|unconditionally native' docs/adr internal/embed/defaults/contracts`.

## Task 7: Regenerate embedded artifacts and add end-to-end coverage

**Files:** `internal/embed/defaults/plugins/catalog.yaml`, `internal/embed/defaults/skills/*/skill.yaml`, `external-skills-source.lock.yaml`, `internal/install/embedded_skill_ingestion_test.go`, `internal/install/installer_wizard_whitebox_test.go`

1. Run `GOCACHE=/tmp/go-cache go run ./cmd/strategist plugins prepare-embedded`.
2. Verify generated catalog role affinity and lock digests.
3. Add an end-to-end success test for wizard -> `.strategist/plugins.lock` -> check.
4. Add an end-to-end failure test removing one binding and asserting non-zero check.
5. Run `GOCACHE=/tmp/go-cache go run ./cmd/strategist plugins prepare-embedded --check` and focused install/check tests; expected: no drift and PASS.

## Task 8: Full verification and handoff

**Files:** all affected packages; no unrelated files.

1. Run `gofmt -w internal/check internal/install internal/domain`.
2. Run `GOCACHE=/tmp/go-cache go test ./internal/install ./internal/check ./internal/domain ./internal/plugins -count=1`.
3. Run `HOME=/tmp GOMODCACHE=/home/sergio/go/pkg/mod GOCACHE=/tmp/go-cache go test ./... -count=1`.
4. Run `git diff --check` and `GOCACHE=/tmp/go-cache go run ./cmd/strategist plugins prepare-embedded --check`.
5. Review `git status --short`; preserve existing user changes and do not reset, clean, stage, or commit without explicit authorization.

Expected: all tests pass and `strategist check` fails for any mandatory role without a valid weapon binding.
