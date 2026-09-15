# Wizard Role-Skill Pipeline Normalization Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make embedded external skills selectable by explicit role affinity while fixed Strategist roles normalize their outputs through immutable pipeline checkpoints, with hard validation in the wizard and strategist check.

**Architecture:** Build ingestion publishes role-affinity metadata into the embedded catalog. The wizard persists only valid Role -> Skill bindings in .strategist. Fixed roles remain the normalization boundary and enforce existing artifact paths, schemas, locks, logs, state transitions, and approval gates.

**Tech Stack:** Go, YAML, go:embed defaults, Cobra CLI, OpenTelemetry, Testify, integration/spec tests.

---

## Constraints

- Preserve all existing staged and unstaged work; do not reset, clean, or overwrite unrelated files.
- external-skills-source/ is the package source, internal/embed/defaults/ is the embedded source, and .strategist/ is the installed runtime.
- Reuse existing pipeline schemas, templates, locks, logs, and binding authorities.
- Git commits are suggested checkpoints only. Do not execute Git state mutation without explicit user authorization.
- The current wizard's supported_handoff_schemas filter is the bug for listing; handoff validation remains mandatory at the fixed role checkpoint.

### Task 1: Establish a focused failing baseline

Files:
- Test: internal/install/wizard_prompts_test.go
- Test: internal/install/embedded_skill_ingestion_test.go
- Test: internal/install/role_provider_migration_test.go
- Test: internal/check/check_role_compatibility.go

Step 1: Run:
    GOCACHE=/tmp/go-cache go test ./internal/install ./internal/check ./internal/domain ./internal/plugins -count=1

Expected: record current pass/fail state and any failures caused by the user's existing worktree.

Step 2: Add a characterization test with brainstorming affiliated to ranger and openspec-propose affiliated to archivist, without native handoff declarations. Assert that role-affiliated entries are eligible for listing while native handoff validation remains a role checkpoint concern.

Step 3: Run:
    GOCACHE=/tmp/go-cache go test ./internal/install -run 'Test.*(RoleAffinity|CompatibleProviderOptions)' -count=1

Expected: FAIL against the current handoff-driven filtering.

### Task 2: Make role affinity explicit and canonical

Files:
- Modify: internal/install/plugin_catalog.go
- Modify: internal/install/embedded_skill_ingestion.go
- Modify: internal/install/embedded_skill_catalog_entry.go
- Modify: internal/install/role_provider_catalog_mapping.go
- Modify: internal/domain/role_provider_contract.go
- Test: internal/install/embedded_skill_ingestion_test.go
- Test: internal/install/role_provider_catalog_mapping_test.go
- Test: internal/domain/role_provider_contract_test.go

Step 1: Add failing tests for roles: [ranger, archivist], missing roles, unknown roles, legacy canonical_role with one role, and disagreement between roles and canonical_role.

Step 2: Add list-valued Roles/roles to the Strategist adapter and catalog provider. Normalize, sort, and deduplicate at ingestion.

Step 3: Validate every declared role against canonical role definitions. Return structured dimension/code/detail diagnostics.

Step 4: Make roles the sole affinity resolver. Retain canonical_role only as a derived compatibility alias when exactly one role is present; disagreement is an error.

Step 5: Add role-affinity compatibility to ProviderContract as a separate dimension. Keep handoff-schema compatibility for checkpoint validation, not wizard listing.

Step 6: Run:
    GOCACHE=/tmp/go-cache go test ./internal/install ./internal/domain -run 'Test.*(RoleAffinity|CatalogProvider|ExternalSkill)' -count=1

Expected: PASS.

### Task 3: Update and regenerate embedded skill metadata

Files:
- Modify: external-skills-source/brainstorming/strategist.yaml
- Modify: external-skills-source/openspec-explore/strategist.yaml
- Modify: external-skills-source/openspec-propose/strategist.yaml
- Modify: internal/embed/defaults/plugins/catalog.yaml
- Modify: external-skills-source.lock.yaml
- Test: internal/install/embedded_skill_ingestion_test.go
- Test: internal/embed/embedded_weapon_roster_test.go

Step 1: Declare roles: [ranger] for brainstorming and openspec-explore, and roles: [archivist] for openspec-propose. Keep portable SKILL.md files unchanged.

Step 2: Assert ingestion copies role affinity and produces deterministic catalog and lock output.

Step 3: Run:
    go run ./cmd/strategist plugins prepare-embedded
    go run ./cmd/strategist plugins prepare-embedded --check

Expected: generated catalog/mirrors/lock are current and check reports no drift.

### Task 4: Make wizard options affinity-driven with explainable exclusions

Files:
- Modify: internal/install/wizard_prompts.go
- Modify: internal/install/wizard.go
- Modify: internal/install/role_provider_catalog_mapping.go
- Test: internal/install/wizard_prompts_test.go
- Test: internal/install/wizard_test.go

Step 1: Add failing tests asserting Ranger lists brainstorming and openspec-explore, Archivist lists openspec-propose, another-role entries are excluded, and missing handoff declarations are not mislabeled as affinity failures.

Step 2: Replace supported_handoff_schemas as the listing predicate with canonical role affinity plus catalog availability, trust, dependency, and provider-contract validation.

Step 3: Extend excludedProviderOption diagnostics with missing affinity, unknown role, malformed metadata, unavailable package, and invalid binding codes. Include role and provider identifiers.

Step 4: Validate the fixed role checkpoint/normalization contract before wizard success. A flexible skill need not declare the native handoff itself.

Step 5: Run:
    GOCACHE=/tmp/go-cache go test ./internal/install -run 'Test(CompatibleProviderOptions|PromptSlots|RunWizard)' -count=1

Expected: PASS and explanatory exclusion output.

### Task 5: Persist and reload the selected Role -> Skill binding

Files:
- Inspect/modify: internal/install/role_provider_migration.go
- Inspect/modify: internal/install/active_yaml.go
- Inspect/modify: internal/install/plugin_lock_file.go
- Inspect/modify: internal/domain/plugin_state_types.go
- Test: internal/install/role_provider_migration_test.go
- Test: internal/install/plugin_lock_file_test.go
- Test: internal/install/active_yaml_test.go

Step 1: Add failing tests selecting brainstorming for Ranger and openspec-propose for Archivist, writing the binding, reloading it, and detecting role/provider, digest, and state mismatches.

Step 2: Reuse activateRoleProviderMigration, PluginLockFile, and SlotBinding. Write only after static validation succeeds and preserve atomic config-lock behavior.

Step 3: Run:
    GOCACHE=/tmp/go-cache go test ./internal/install -run 'Test.*(Migration|PluginLock|ActiveYAML|Binding)' -count=1

Expected: PASS with durable immutable-instance references and hard errors for invalid bindings.

### Task 6: Put output normalization in the fixed roles and checkpoints

Files:
- Modify: internal/embed/defaults/internal_skills/ranger/SKILL.md
- Modify: internal/embed/defaults/internal_skills/archivist/SKILL.md
- Modify: internal/embed/defaults/contracts/machine/handoff-contract.yaml
- Modify: internal/embed/defaults/contracts/machine/mission-status.yaml
- Inspect/modify: internal/handoff/verify.go
- Inspect/modify: internal/handoff/policy.go
- Test: internal/handoff/verify_test.go
- Test: internal/handoff/role_provider_conformance_test.go
- Test: tests/spec/discovery_contract_test.go
- Test: tests/evals/contracts/archivist_handoff_schema_valid_test.go

Step 1: Add failing fixtures for pure weapon output. Accept only results normalized into canonical pending/refined artifacts and the corresponding handoff. Cover missing fields, wrong path, stale lock, invalid state, and concurrent transition.

Step 2: Extend fixed Ranger and Archivist contract surfaces to state that weapon output is input to normalization, never a handoff. Require existing templates, evidence, locks, logs, state, and schemas.

Step 3: Wire the existing internal/handoff verifier and policy/state authorities to the provider-result checkpoint boundary. Do not create a second state store or handoff protocol.

Step 4: Run:
    GOCACHE=/tmp/go-cache go test ./internal/handoff ./tests/spec ./tests/evals/contracts -run 'Test.*(Handoff|Discovery|Archivist|RoleProvider)' -count=1

Expected: PASS for valid normalization and hard failure for malformed output or state/concurrency violations.

### Task 7: Reuse the same validation in strategist check

Files:
- Modify: internal/check/check_role_compatibility.go
- Modify: internal/check/check_slots.go
- Modify: internal/check/check_runtime.go
- Modify: internal/check/check_lock_parity.go
- Test: internal/check/check_slots_test.go
- Test: internal/check/check_runtime_test.go
- Test: internal/check/check_lock_parity_test.go

Step 1: Add failing tests for missing/unknown affinity, role/binding mismatch, catalog/runtime drift, invalid checkpoint, missing lock, bad control-log state, and stale provenance. Assert dimension, code, role, and provider in diagnostics.

Step 2: Extract or reuse a non-mutating shared validator for wizard and check. Keep affinity, fixed checkpoint, handoff, and runtime-state dimensions distinct.

Step 3: Make invalid selected bindings non-zero check results. Do not let native fallback metadata conceal a selected binding error.

Step 4: Run:
    GOCACHE=/tmp/go-cache go test ./internal/check -run 'Test.*(Role|Slot|Runtime|Lock)' -count=1

Expected: PASS with equivalent wizard/check diagnostics.

### Task 8: Add end-to-end regression coverage

Files:
- Modify: tests/integration/install_test.go
- Modify: tests/integration/e2e_harness_test.go
- Modify: tests/spec/specs/slot-contracts.feature
- Modify: tests/spec/specs/e2e-install-compile.feature

Step 1: Exercise the real embedded catalog and wizard with brainstorming -> ranger and openspec-propose -> archivist. Assert both options appear, bindings persist under .strategist, and fixed pipeline contracts remain selected.

Step 2: Exercise unknown role, conflicting legacy canonical_role, invalid checkpoint, and corrupted lock/control log. Assert wizard/check fail with reasons.

Step 3: Run:
    GOCACHE=/tmp/go-cache go test ./tests/integration ./tests/spec -count=1

Expected: PASS.

### Task 9: Update ADRs and embedded contract documentation

Files:
- Modify: docs/adr/0035-embedded-weapon-fallback-policy.md
- Modify: docs/adr/0037-wizard-role-binding-persistence.md
- Modify: docs/adr/0034-role-and-skill-taxonomy.md
- Modify: internal/embed/defaults/contracts/narrative/00-routing.md
- Modify: internal/embed/defaults/contracts/narrative/03-discovery.md
- Modify: internal/embed/defaults/contracts/narrative/04-refinement.md
- Modify: internal/embed/defaults/contracts/tests/ranger.test.yaml
- Modify: internal/embed/defaults/contracts/tests/archivist.test.yaml

Step 1: Document pipeline > role > skill, explicit multi-role affinity, role-owned normalization, hard wizard/check validation, and no silent fallback.

Step 2: Remove claims that external weapons must declare native handoff schemas. Keep handoff requirements at fixed role checkpoints.

Step 3: Run:
    GOCACHE=/tmp/go-cache go test ./tests/spec ./internal/embed -count=1

Expected: PASS.

### Task 10: Full verification and handoff

Files:
- No source changes expected; inspect all files changed by Tasks 1-9.

Step 1: Run formatting only on changed Go files, then:
    GOCACHE=/tmp/go-cache go test ./internal/domain ./internal/handoff ./internal/install ./internal/check ./internal/plugins ./tests/integration ./tests/spec ./tests/evals/contracts -count=1

Expected: PASS.

Step 2: Run the narrowest applicable Make quality target discovered from Makefile and make/quality.mk. Record unrelated pre-existing failures separately. Do not bypass hooks or alter Git configuration.

Step 3: Run:
    go run ./cmd/strategist plugins prepare-embedded --check
    go test ./... -count=1

Expected: no embedded-skill drift and the full Go suite passes.

Step 4: Inspect a clean temporary install directly. Confirm .strategist contains the selected binding, canonical pipeline files, valid locks, and control-log-compatible state. Confirm strategist check succeeds only for the coherent runtime.

Step 5: Report changed files, tests, generated artifacts, warnings, environment limits, and whether the analysis has explicit completion evidence. Do not move the analysis to .analysis/done/ without that evidence.

