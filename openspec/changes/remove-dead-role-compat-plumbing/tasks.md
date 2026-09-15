<!--
Strategist task_type classification (handoff-archivist-to-sniper.schema.yaml):
every task below is `implementation_handoff` (source/doc mutation). None is a
Sniper-executable `documentation_target` item under the default execution
contract — materialization requires a separately authorized execution
provider outside Strategist.
-->

## 1. Remove dead handoff-schema plumbing

- [x] 1.1 [implementation_handoff] Delete `loadSupportedHandoffSchemas()` in `internal/check/check_role_compatibility.go` (~lines 98-127) and its two call sites (~lines 70-74, 85) that assign into `providerContract.SupportedHandoffSchemas` and branch `RoleContractFromConfig`; verify by running `go build ./internal/check/...` and `go test ./internal/check/...` with no failures.
- [x] 1.2 [implementation_handoff] Confirm `CheckRoleAffinity`'s pass/fail outcome is unchanged for every existing `internal/check` test case after 1.1; verify by re-running the full `internal/check` test suite before and after the change and diffing pass/fail results.

## 2. Remove dead constant

- [x] 2.1 [implementation_handoff] Delete the `SlotExtensionLabel` constant in `internal/domain/plugin_types.go` (~lines 40-42); verify by running `grep -rn SlotExtensionLabel .` across the repo and confirming zero remaining references, then `go build ./...`.

## 3. Documentation corrections

- [x] 3.1 [implementation_handoff] Correct the stale "Negative consequences" paragraph in `docs/adr/0038-docs-information-architecture-and-mandate-layer.md` (~lines 100-107) to reflect that `cmd/strategist/handoff_verify.go`, `internal/dojo/checker_pipeline.go`, and `tests/spec/cicd_residual_contract_test.go` already reference the current doc paths; verify by re-reading the three cited files and confirming the ADR text matches their current state.
- [x] 3.2 [implementation_handoff] Add a `CHANGELOG.md` `[Unreleased] ### Removed` entry covering: (a) this change's deletion of the dead handoff-schema plumbing and `SlotExtensionLabel`, and (b) the already-staged removal of `CheckRoleCompatibility`/`ResolveProviderBinding` from `internal/domain/role_provider_compatibility.go`; verify by confirming the entry lists both removals with a one-line rationale each.

## 4. Harden ADR-0035 ask-first gap (SQ-2, incorporated at Approval Gate revision)

- [x] 4.1 [implementation_handoff] In `internal/install/plugin_catalog.go`'s `resolveInstallableDefaultProviders` (~lines 112-122), change the `loadPluginCatalog` error branch (~lines 113-115) to propagate the error (return `(map[string]string, error)`) instead of silently returning `installableDefaultProviders`; update the function signature and both call sites (`installer_config.go:127`, and its caller `writeSelectedProviderManifest`) accordingly; verify by running `go build ./internal/install/...`.
- [x] 4.2 [implementation_handoff] Add a doc comment to `resolveInstallableDefaultProviders` stating it is unreachable-in-practice via its two current callers (both already guard `loadPluginCatalog` earlier in the call stack: `runWizard` at `wizard.go:88-91`, `activateSilentRoleProviderBindings` at `installer_silent_role_bindings.go:24-27`), mirroring the existing comment style on `loadKnownProviders`; verify by reading the updated comment against both call sites.
- [x] 4.3 [implementation_handoff] Add or extend an `internal/install` test asserting `writeSelectedProviderManifest`/`writeSelectedProviderManifests` returns an error (rather than silently using the fallback map) when `loadPluginCatalog` fails, per ADR-0035's Validation Requirements; verify by running `go test ./internal/install/...` and confirming the new test fails against the pre-4.1 code and passes after.
- [x] 4.4 [implementation_handoff] Confirm the sibling zero-installable-entries branch (~lines 117-119) and `loadKnownProviders` are left unchanged (design.md Non-Goals); verify by diffing this change against `internal/install/plugin_catalog.go` and `internal/install/wizard_fallback_providers.go` and confirming only the lines named in 4.1/4.2 changed.

## 5. Confirm scope boundary

- [x] 5.1 [implementation_handoff] Confirm no changes were made to `internal/install/plugin_catalog_legacy.go`, `LegacyManifestPath`, `internal/domain/jewel_grade.go`'s legacy-status guard, the `installableDefaultProviders`/`knownProviderRisk` map values themselves, or `tests/spec/mission_status_vocabulary_test.go`/`tests/spec/legacy_terminology_test.go`; verify by running `git diff --stat` against the base branch and confirming none of those paths (or map literals) appear.
