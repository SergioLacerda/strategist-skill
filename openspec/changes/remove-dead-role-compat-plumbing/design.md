## Context

See proposal.md - Why. This change touches a single package
(`internal/check`), one constant in `internal/domain`, and two doc files.
No cross-cutting refactor, no new dependency, no data-model or migration
concern — the design surface is small enough that this document exists mainly
to record the removal boundary explicitly, so a future pass doesn't need to
re-derive it from scratch.

## Goals / Non-Goals

**Goals:**
- Remove `loadSupportedHandoffSchemas()` and its call sites in
  `internal/check/check_role_compatibility.go` without changing
  `CheckRoleAffinity`'s observable pass/fail behavior.
- Remove the unreferenced `SlotExtensionLabel` constant in
  `internal/domain/plugin_types.go`.
- Correct ADR-0038's stale paragraph and add the CHANGELOG entry.

**Non-Goals:**
- Any change to `plugin_catalog_legacy.go`, the jewel legacy-status guard, the
  Wizard's hardcoded fallback maps, the Go/product vocabulary split, or the
  anti-regression guard tests — see proposal.md's "Explicitly out of scope"
  list. This design does not revisit those; they were already adjudicated by
  ADR-0030, ADR-0012, and ADR-0035, or are deliberate/protective by design.
- Changing `loadKnownProviders` or the values inside `installableDefaultProviders`/
  `knownProviderRisk` — only `resolveInstallableDefaultProviders`'s error
  handling changes (SQ-2), not the maps or the sibling function.
- Closing the second silent-fallback branch in `resolveInstallableDefaultProviders`
  (lines 117-119, where a successfully-loaded but zero-installable-entries
  catalog falls back to the static map). That branch is not an error case —
  it is a structurally valid but empty catalog — and treating it the same as
  the error case is a separate, not-yet-requested design decision.
- Renaming, refactoring, or otherwise touching `SupportedHandoffSchemas` on
  `ProviderContract` itself, or the catalog/ingestion code that populates it
  (`plugin_catalog.go`, `embedded_skill_ingestion.go`,
  `embedded_skill_catalog_entry.go`, `role_provider_catalog_mapping.go`). The
  audit found the field is still written by five files and covered by tests
  unrelated to this change's scope; removing the field itself is a larger,
  separate change this proposal does not take on.

## Decisions

- **Remove the dead call site, not just mark it unused**: delete
  `loadSupportedHandoffSchemas()` and the two call-site lines that consume its
  result (`providerContract.SupportedHandoffSchemas = ...` and the
  `RoleContractFromConfig` branch keyed on it), rather than leaving the
  function in place unused. Alternative considered: leave the helper as a
  `//nolint:unused`-tagged utility for a hypothetical future consumer.
  Rejected — `SupportedHandoffSchemas` on `ProviderContract` stays (it is
  populated and tested elsewhere, per proposal.md's non-goals); only the
  *consumption* of it inside this one compatibility check is dead. Keeping an
  unused helper around "for later" is exactly the kind of legacy accumulation
  this mission is mapping.
- **Delete `SlotExtensionLabel` outright rather than wiring it in**: the
  audit's discovery brief flagged two options — delete the dead constant, or
  actually use it wherever `"provider"` strings are user-facing. Wiring it in
  is a product-language change with its own review surface (which
  user-visible strings change, and where) that is out of proportion to a
  legacy-cleanup mission. Deleting the unused constant is the minimal,
  reversible choice; wiring it in can be a separate, deliberately-scoped
  change if someone wants it.
- **Bundle the ADR-0038 doc fix and CHANGELOG entry into the same change**
  rather than splitting them into separate proposals: both are small,
  low-risk, and directly traceable to this same discovery pass. Splitting
  would add process overhead (three OpenSpec changes instead of one) without
  reducing review risk, since none of the three edits touch overlapping code.
- **Fix `resolveInstallableDefaultProviders` by propagating the error, not by
  deleting the fallback map it falls back to**: the discovery brief's SQ-2
  named `loadKnownProviders`/`resolveInstallableDefaultProviders` together,
  but ADR-0035 already settled that the underlying maps
  (`installableDefaultProviders`, `knownProviderRisk`) are permanent, and
  `loadKnownProviders` is already documented and practically unreachable.
  Re-verifying the call graph (not just trusting the ADR's context prose)
  showed `resolveInstallableDefaultProviders` is in the same
  "unreachable via current callers" position as `loadKnownProviders` — so the
  fix here is the same shape as `runWizard`'s existing guard: propagate the
  error instead of silently substituting. Alternative considered: delete the
  fallback map read entirely and make catalog-load failure always fatal
  inside this function. Rejected — that would remove the last-resort default
  for a hypothetical future caller that does not have an upstream guard,
  which is more than ADR-0035 or the user's request asked for; propagating
  the error (rather than the map) preserves the map as a documented fallback
  value while still failing loudly when it would actually be used.

## Risks / Trade-offs

- [Risk: removing `loadSupportedHandoffSchemas()`'s call sites changes which
  `RoleContractFromConfig` variant is constructed, even though the
  *compatibility outcome* is unchanged] → Mitigation: run
  `internal/check`'s existing test suite before/after; the discovery brief
  already confirmed no test asserts on the branch selection itself, only on
  `CheckRoleAffinity`'s final result.
- [Risk: a future feature does eventually want `SupportedHandoffSchemas`
  consulted inside role compatibility, and this change deletes the only
  wiring that read it] → Mitigation: low — the field itself is untouched and
  still populated end-to-end; re-adding a consumer later is a small, additive
  change, not a revert of this one.
- [Risk: this proposal's tasks are `implementation_handoff`, not
  Sniper-executable — someone could mistake "proposal approved" for
  "code will be changed automatically"] → Mitigation: tasks.md states this
  explicitly, and the Strategist Approval Gate presentation for this mission
  will restate that execution requires a separately authorized provider.
- [Risk: propagating the error from `resolveInstallableDefaultProviders`
  changes `writeSelectedProviderManifest`'s error surface — a caller that
  previously never saw an error from this path could now see one] →
  Mitigation: both known callers (`writeSelectedProviderManifests`'s two call
  sites) already propagate errors from sibling calls in the same function
  (e.g. `writeTreasureChestManifest`, `activateRoleProviderMigration`), so
  adding one more propagated error is consistent with the existing error
  contract, not a new pattern. Verify with `go test ./internal/install/...`
  before and after.

## Migration Plan

Not applicable — this is a deletion of dead code and a doc correction, not a
data or runtime migration. No rollback strategy beyond a normal `git revert`
of the eventual commit.
