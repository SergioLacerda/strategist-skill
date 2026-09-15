# Role↔Weapon Handoff-Schema Compatibility Implementation Plan

> **REQUIRED SUB-SKILL:** Use executing-plans to implement this plan task-by-task.

**Goal:** Give `CheckRoleCompatibility` a third dimension — `handoff_schema` — so a weapon whose real output shape doesn't match its bound role's contract (like `openspec-propose` writing OpenSpec's own artifact shape instead of Archivist's four-file refined package) is caught statically, before mission-time invocation, and make the Wizard's refinement/discovery prompts reflect that compatibility instead of a hardcoded option list.

**Architecture:** `domain.ProviderContract` gains `SupportedHandoffSchemas []string`; `CheckRoleCompatibility` rejects a candidate whose declared schemas don't include the role's `HandoffSchema` (skipped for native-role providers, which trivially satisfy their own role). `internal/install` wires this through the existing `providerContractsForRole` → `ResolveRoleBinding` → `PlanRoleProviderMigration` pipeline (already landed via ADR-0037) by populating `RoleContract.HandoffSchema` (currently hardcoded `""`) and by adding a native `ranger` catalog entry so discovery still resolves once `brainstorming`/`openspec-explore` are honestly declared incompatible. The Wizard's `promptSlots` option lists become compatibility-driven instead of a static two-item slice.

**Tech Stack:** Go, `gopkg.in/yaml.v3`, `stretchr/testify` (`assert`/`require`).

**Context this plan assumes you've read:** `.analysis/pending/2026-09-14-role-weapon-binding-gap-design.md` (the approved spec), `docs/adr/0035-embedded-weapon-fallback-policy.md`, `docs/adr/0037-wizard-role-binding-persistence.md`, `.analysis/done/20260728-ranger-drift-eval/` (why discovery's native-only rule exists and why a *declarative* manifest check has a known limit).

**Important — read before Task 1:** ADR-0037's Role/Provider migration machinery (`internal/install/role_provider_migration.go`, `internal/plugins/role_binding.go`) already exists and is already wired into `runWizard`. Several *existing* tests currently assert that `brainstorming` (discovery) and `openspec-propose` (refinement) resolve as **compatible** — that assertion is the exact bug this plan fixes, so those tests must be rewritten, not just left passing. Task 7 below enumerates every one by name. Do not skip it — a green test suite that still asserts the old (wrong) behavior means the fix didn't actually land.

---

### Task 1: `handoff_schema` compatibility dimension on `ProviderContract`

**Files:**
- Modify: `internal/domain/role_provider_contract.go:99-163`
- Test: `internal/domain/role_provider_contract_test.go`

**Step 1: Write the failing tests**

Add to `internal/domain/role_provider_contract_test.go` (after `TestCheckRoleCompatibilityRejectsUnsupportedRoleContractVersion`):

```go
func TestCheckRoleCompatibilityRejectsUnsupportedHandoffSchema(t *testing.T) {
	t.Parallel()

	role := validRoleContract()
	role.HandoffSchema = "schemas/handoff-archivist-to-sniper.schema.yaml"
	provider := validProviderContract()
	provider.CanonicalRole = "sniper" // matches role's Role field ("sniper")
	provider.SupportedHandoffSchemas = []string{"some-other-schema.yaml"}

	result := provider.CheckRoleCompatibility(role)
	assert.False(t, result.Compatible)
	require.Len(t, result.Reasons, 1)
	assert.Equal(t, "handoff_schema", result.Reasons[0].Dimension)
	assert.Equal(t, "unsupported_handoff_schema", result.Reasons[0].Code)
}

func TestCheckRoleCompatibilityAcceptsMatchingHandoffSchema(t *testing.T) {
	t.Parallel()

	role := validRoleContract()
	role.HandoffSchema = "schemas/handoff-archivist-to-sniper.schema.yaml"
	provider := validProviderContract()
	provider.SupportedHandoffSchemas = []string{"schemas/handoff-archivist-to-sniper.schema.yaml"}

	result := provider.CheckRoleCompatibility(role)
	assert.True(t, result.Compatible)
}

func TestCheckRoleCompatibilitySkipsHandoffSchemaDimensionWhenRoleDeclaresNone(t *testing.T) {
	t.Parallel()

	role := validRoleContract() // HandoffSchema is "" (zero value) — e.g. Sniper, the terminal role
	provider := validProviderContract()
	provider.SupportedHandoffSchemas = nil

	result := provider.CheckRoleCompatibility(role)
	assert.True(t, result.Compatible, "a role with no declared handoff schema imposes no constraint on this dimension")
}

func TestCheckRoleCompatibilitySkipsHandoffSchemaDimensionForNativeRoleProviders(t *testing.T) {
	t.Parallel()

	role := validRoleContract()
	role.HandoffSchema = "schemas/handoff-archivist-to-sniper.schema.yaml"
	provider := validProviderContract() // Source: domain.ProviderSourceNativeRole (see validProviderContract())
	provider.SupportedHandoffSchemas = nil

	result := provider.CheckRoleCompatibility(role)
	assert.True(t, result.Compatible, "a native role trivially satisfies its own handoff contract")
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/domain/... -run TestCheckRoleCompatibility -v`
Expected: the four new tests FAIL (`SupportedHandoffSchemas` field doesn't exist yet — compile error) or, once the field compiles but the check isn't added, `TestCheckRoleCompatibilityRejectsUnsupportedHandoffSchema` FAILs because `result.Compatible` is `true`.

**Step 3: Add the field and the compatibility check**

In `internal/domain/role_provider_contract.go`, add the field to `ProviderContract` (after `SupportedRoleContractVersions`, around line 113):

```go
	SupportedRoleContractVersions []string             `yaml:"supported_role_contract_versions"`
	// SupportedHandoffSchemas declares which handoff_schema value(s)
	// (RoleContract.HandoffSchema) this Provider's real output actually
	// conforms to. A Provider whose manifest omits this field supports none
	// — CheckRoleCompatibility then correctly reports it incompatible with
	// any role that declares a HandoffSchema, rather than defaulting to
	// "compatible" the way SupportedRoleContractVersions' absence would not
	// (that field is always synthesized as compatible today — see
	// internal/install/role_provider_catalog_mapping.go). This is what
	// closes the gap mission 20260914-role-weapon-structure-review hit
	// live: openspec-propose passed canonical_role/role_contract_version
	// compatibility while writing OpenSpec's own artifact shape instead of
	// Archivist's.
	SupportedHandoffSchemas []string `yaml:"supported_handoff_schemas,omitempty"`
```

In `CheckRoleCompatibility` (currently ends at line 162 with `return CompatibilityResult{Compatible: true}`), insert the new check between the `role_contract_version` check and the final return:

```go
	if p.Source != ProviderSourceNativeRole && role.HandoffSchema != "" &&
		!hasString(stringSet(p.SupportedHandoffSchemas...), role.HandoffSchema) {
		return CompatibilityResult{Compatible: false, Reasons: []CompatibilityReason{{
			Dimension: "handoff_schema",
			Code:      "unsupported_handoff_schema",
			Detail:    fmt.Sprintf("%s does not declare support for handoff schema %s", p.ID, role.HandoffSchema),
		}}}
	}
	return CompatibilityResult{Compatible: true}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/domain/... -run TestCheckRoleCompatibility -v`
Expected: PASS (all four new tests, plus the three pre-existing `TestCheckRoleCompatibility*` tests, which are unaffected since `validRoleContract()`/`validProviderContract()` leave `HandoffSchema`/`SupportedHandoffSchemas` at their zero value).

**Step 5: Commit**

```bash
git add internal/domain/role_provider_contract.go internal/domain/role_provider_contract_test.go
git commit -m "feat: add handoff_schema compatibility dimension to CheckRoleCompatibility"
```

---

### Task 2: Populate `RoleContract.HandoffSchema` in the migration pipeline

Today `internal/install/role_provider_migration.go:43` calls `domain.RoleContractFromConfig(roleCfg, "")` — the handoff schema argument is hardcoded empty, so Task 1's new check is currently a no-op in the one place that matters. `roles/<name>.yaml`'s own `canonical.handoff_contract` field is prose-only (inside a YAML list of behaviors, not machine-parsed) — this task adds the machine-readable counterpart, matching the existing small `defaultSkillBySlot`-style map pattern in `internal/install/paths.go` rather than growing the `RoleConfig` schema for two values.

**Files:**
- Create: `internal/install/role_handoff_schemas.go`
- Modify: `internal/install/role_provider_migration.go:43`
- Test: `internal/install/role_provider_migration_test.go`

**Step 1: Create the lookup map**

```go
package install

// roleHandoffSchema mirrors each native role's own handoff_contract
// declaration in roles/<name>.yaml's `canonical:` block (human-readable
// prose there, not machine-parsed) — kept here as the machine-readable
// counterpart PlanRoleProviderMigration needs to populate
// domain.RoleContract.HandoffSchema. If roles/<name>.yaml/handoff_contract
// changes, update this map to match. Sniper has no entry: it is the
// pipeline's terminal role and hands nothing downstream, so its
// RoleContract.HandoffSchema is correctly "" (Task 1's check is then
// skipped for it, imposing no constraint on that dimension).
var roleHandoffSchema = map[string]string{
	"ranger":    "schemas/handoff-ranger-to-archivist.schema.yaml",
	"archivist": "schemas/handoff-archivist-to-sniper.schema.yaml",
}
```

**Step 2: Wire it into `PlanRoleProviderMigration`**

In `internal/install/role_provider_migration.go:43`, change:

```go
		role := domain.RoleContractFromConfig(roleCfg, "")
```

to:

```go
		role := domain.RoleContractFromConfig(roleCfg, roleHandoffSchema[roleName])
```

**Step 3: Run the existing migration tests — expect specific, known failures**

Run: `go test ./internal/install/... -run TestPlanRoleProviderMigration -v`

Expected: this alone does NOT yet change behavior, because no `ProviderContract` in the catalog declares `SupportedHandoffSchemas` yet (Task 3/6 below), so `hasString(stringSet()...)` is always false against a non-empty `role.HandoffSchema` — **every** external candidate for `ranger`/`archivist` becomes incompatible at once, including the ones that are actually fine. Do not be alarmed if `TestPlanRoleProviderMigrationIsFullyResolvedForDefaultSlots` and friends fail here — this is expected mid-plan state. Task 7 fixes the assertions once Tasks 3-6 give the catalog real data to check against. Commit Task 2 anyway; the failing tests are addressed by name in Task 7.

**Step 4: Commit**

```bash
git add internal/install/role_handoff_schemas.go internal/install/role_provider_migration.go
git commit -m "feat: populate RoleContract.HandoffSchema in the role/provider migration pipeline"
```

---

### Task 3: `SupportedHandoffSchemas` on the catalog provider shape

**Files:**
- Modify: `internal/install/plugin_catalog.go:22-40` (`pluginCatalogProvider` struct)
- Modify: `internal/install/role_provider_catalog_mapping.go:25-38` (`providerContractFromCatalogEntry`)
- Test: `internal/install/role_provider_catalog_mapping_test.go` (new file — none exists today)

**Step 1: Write the failing test**

Create `internal/install/role_provider_catalog_mapping_test.go`:

```go
package install

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestProviderContractFromCatalogEntryCarriesSupportedHandoffSchemas(t *testing.T) {
	t.Parallel()

	entry := pluginCatalogProvider{
		ID:                  "openspec-propose",
		RiskScore:           "write_analysis",
		CanonicalRole:       "archivist",
		CompatibilitySource: "embedded",
		SupportedHandoffSchemas: []string{"openspec-native-change-schema"},
	}

	contract := providerContractFromCatalogEntry(entry)
	assert.Equal(t, []string{"openspec-native-change-schema"}, contract.SupportedHandoffSchemas)
}

func TestProviderContractFromCatalogEntryLeavesSupportedHandoffSchemasNilWhenUndeclared(t *testing.T) {
	t.Parallel()

	entry := pluginCatalogProvider{ID: "brainstorming", RiskScore: "write_analysis", CanonicalRole: "ranger", CompatibilitySource: "embedded"}

	contract := providerContractFromCatalogEntry(entry)
	assert.Empty(t, contract.SupportedHandoffSchemas)
}

func TestProviderContractFromCatalogEntryNativeRoleNeedsNoDeclaredSchema(t *testing.T) {
	t.Parallel()

	entry := pluginCatalogProvider{ID: "archivist", RiskScore: "write_analysis", CompatibilitySource: "native_role"}

	contract := providerContractFromCatalogEntry(entry)
	assert.Equal(t, domain.ProviderSourceNativeRole, contract.Source)
	assert.Empty(t, contract.SupportedHandoffSchemas, "native role providers skip the handoff_schema dimension entirely (Task 1); they need no declared value")
}
```

**Step 2: Run to verify it fails**

Run: `go test ./internal/install/... -run TestProviderContractFromCatalogEntry -v`
Expected: compile error — `pluginCatalogProvider` has no field `SupportedHandoffSchemas`.

**Step 3: Add the field and wire it through**

In `internal/install/plugin_catalog.go`, add to `pluginCatalogProvider` (after `Dependencies`):

```go
	Dependencies        []pluginCatalogDependency `yaml:"dependencies,omitempty"`
	// SupportedHandoffSchemas declares which RoleContract.HandoffSchema
	// value(s) this weapon's real output conforms to (see
	// domain.ProviderContract.SupportedHandoffSchemas). Omitted/empty means
	// none — the weapon has no declared compatibility with any role's
	// handoff contract on this dimension.
	SupportedHandoffSchemas []string `yaml:"supported_handoff_schemas,omitempty"`
```

In `internal/install/role_provider_catalog_mapping.go`'s `providerContractFromCatalogEntry`, add the field to the returned `domain.ProviderContract` literal (after `SupportedRoleContractVersions`):

```go
		SupportedRoleContractVersions: []string{domain.RoleContractSchemaVersion},
		SupportedHandoffSchemas:       provider.SupportedHandoffSchemas,
```

**Step 4: Run to verify it passes**

Run: `go test ./internal/install/... -run TestProviderContractFromCatalogEntry -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/install/plugin_catalog.go internal/install/role_provider_catalog_mapping.go internal/install/role_provider_catalog_mapping_test.go
git commit -m "feat: carry supported_handoff_schemas from the catalog into ProviderContract"
```

---

### Task 4: Native `ranger` catalog entry

Without this, once `brainstorming`/`openspec-explore` are (correctly) reported incompatible for `ranger` in Task 6, discovery has **zero** compatible candidates at all — `ResolveRoleBinding` returns `role_binding_missing` instead of falling back to native Ranger, breaking `PlanRoleProviderMigration.FullyResolved()` for every default install. `archivist` and `sniper` already have exactly this kind of entry; `ranger` does not.

**Files:**
- Modify: `internal/embed/defaults/plugins/catalog.yaml`

**Step 1: Add the entry**

Insert alphabetically (matches the file's existing id-sorted ordering), after `openspec-propose` and before `sdd-ask`... actually alphabetically `ranger` sorts between `openspec-propose` and `sdd-ask`. Insert at line 50, right before the `sdd-ask` entry:

```yaml
    - id: ranger
      risk_score: write_analysis
      compatibility_source: native_role
```

**Step 2: Verify it parses and resolves**

Run: `go test ./internal/install/... -run TestPlanRoleProviderMigration -v`
Expected: still failing at this point exactly as Task 2 Step 3 predicted (brainstorming/openspec-explore incompatible, and now `ranger` exists as a candidate but `providerContractFromCatalogEntry` gives it `SupportedHandoffSchemas: nil` while `Source: native_role` — Task 1's guard skips the check for it, so it resolves compatible). Confirm this specifically:

Run: `go test ./internal/install/... -run TestPlanRoleProviderMigrationSeparatesRoleFromProviderCandidates -v`
Expected: still FAILs on the `discovery.Resolved.Provider.ID` assertion (still asserts `"brainstorming"` — fixed in Task 7), but you can manually confirm resolution now targets `ranger` by temporarily adding `t.Log(discovery.Resolved.Provider.ID)` or just proceeding to Task 7 where the assertion is corrected.

**Step 3: Commit**

```bash
git add internal/embed/defaults/plugins/catalog.yaml
git commit -m "feat: add native ranger catalog entry as discovery's compatible fallback"
```

---

### Task 5: `strategist.yaml` sidecars stay honest (no supported_handoff_schemas)

This task is documentation-only — Tasks 3/6 already default an undeclared field to "supports nothing," which is the *correct* value for all three weapons today. This task makes that omission explicit and intentional rather than looking like an oversight.

**Files:**
- Modify: `external-skills-source/openspec-propose/strategist.yaml`
- Modify: `external-skills-source/brainstorming/strategist.yaml`
- Modify: `external-skills-source/openspec-explore/strategist.yaml`

**Step 1: Add an explanatory comment to each (no field value)**

Append to `external-skills-source/openspec-propose/strategist.yaml`:

```yaml
# supported_handoff_schemas intentionally omitted: this package's own
# SKILL.md instructs writing OpenSpec's native artifact shape
# (openspec/changes/<name>/{proposal,design,tasks}.md via the `openspec`
# CLI), not Archivist's handoff-archivist-to-sniper.schema.yaml shape
# (<base_path>/refined/<mission_id>/{analysis,proposal,design,tasks}.md).
# Confirmed live in mission 20260914-role-weapon-structure-review — do not
# add a value here without first changing what this package actually
# writes.
```

Append to `external-skills-source/brainstorming/strategist.yaml` and `external-skills-source/openspec-explore/strategist.yaml`:

```yaml
# supported_handoff_schemas intentionally omitted: this package's own
# SKILL.md is a multi-turn, approval-gated design flow, structurally
# incompatible with Ranger's autonomous single-shot
# handoff-ranger-to-archivist.schema.yaml output — see
# .analysis/done/20260728-ranger-drift-eval/. Discovery never live-invokes
# this weapon regardless (00-routing.md § Discovery Weapon Resolution by
# Subtype); this omission keeps the Wizard's install-time compatibility
# preview honest about the same fact.
```

**Step 2: Commit**

```bash
git add external-skills-source/openspec-propose/strategist.yaml external-skills-source/brainstorming/strategist.yaml external-skills-source/openspec-explore/strategist.yaml
git commit -m "docs: document why the embedded weapons declare no supported_handoff_schemas"
```

---

### Task 6: Wire `supported_handoff_schemas` through embedded-skill ingestion

Forward-compatibility for the day a weapon *does* declare a real value (Task 5's sidecars stay empty today, but the ingestion pipeline that regenerates `catalog.yaml` from `strategist.yaml` needs to know the field exists, or a future declared value would be silently dropped).

**Files:**
- Modify: `internal/install/embedded_skill_ingestion.go:27-33` (`externalSkillAdapter`)
- Modify: `internal/install/embedded_skill_catalog_entry.go:10-26` (`catalogProviderFromIngestedSkill`)
- Test: `internal/install/embedded_skill_ingestion_test.go`

**Step 1: Write the failing test**

Find the existing test that builds a `strategist.yaml`-equivalent fixture and asserts the resulting `pluginCatalogProvider` fields (grep `embedded_skill_ingestion_test.go` for an existing `TestIngestExternalSkills...` case that constructs an adapter YAML and checks the mapped catalog entry — extend that fixture's YAML with a `supported_handoff_schemas: [example-schema.yaml]` line and assert `catalog.Providers[...].SupportedHandoffSchemas == []string{"example-schema.yaml"}` in the same style as the surrounding assertions in that test).

**Step 2: Run to verify it fails**

Run: `go test ./internal/install/... -run TestIngestExternalSkills -v`
Expected: FAIL — the new YAML field is silently dropped (unknown field, ignored by yaml.v3 by default).

**Step 3: Wire the field through**

In `internal/install/embedded_skill_ingestion.go`, add to `externalSkillAdapter`:

```go
type externalSkillAdapter struct {
	CanonicalRole  string   `yaml:"canonical_role"`
	RiskScore      string   `yaml:"risk_score"`
	Category       string   `yaml:"category"`
	Default        bool     `yaml:"default,omitempty"`
	AuxiliaryTools []string `yaml:"auxiliary_tools_allowed,omitempty"`
	// SupportedHandoffSchemas — see domain.ProviderContract's field of the
	// same name and internal/install/role_handoff_schemas.go. Omitted by
	// every embedded weapon today (Task 5) — added here only so a future
	// honest declaration isn't silently dropped by ingestion.
	SupportedHandoffSchemas []string `yaml:"supported_handoff_schemas,omitempty"`
}
```

In `internal/install/embedded_skill_catalog_entry.go`'s `catalogProviderFromIngestedSkill`, add to the returned `pluginCatalogProvider` literal:

```go
		AuxiliaryTools:      skill.Adapter.AuxiliaryTools,
		SupportedHandoffSchemas: skill.Adapter.SupportedHandoffSchemas,
```

**Step 4: Run to verify it passes**

Run: `go test ./internal/install/... -run TestIngestExternalSkills -v`
Expected: PASS.

**Step 5: Regenerate the embedded catalog and lock**

Run: `strategist plugins prepare-embedded` (or `go run ./cmd/strategist plugins prepare-embedded` if the binary isn't on `PATH`) from the repo root.
Expected: `internal/embed/defaults/plugins/catalog.yaml`'s `openspec-propose`/`brainstorming`/`openspec-explore` entries regenerate unchanged (Task 5 added comments only, no field value) except for whatever routine drift the tool already produces; `external-skills-source.lock.yaml` refreshes to include `openspec-propose` if it doesn't already (this closes the stale-lock gap the prior evaluation flagged — KF-08 in `.analysis/pending/20260914-role-weapon-structure-review-analysis.md` — as a side effect). **Do not hand-edit the regenerated `catalog.yaml` entries** — if the tool's output differs from what Task 4 hand-added for `ranger`, that's expected: `ranger` is `compatibility_source: native_role`, never touched by this generator (same as `archivist`/`sniper` today).

Run: `git diff internal/embed/defaults/plugins/catalog.yaml external-skills-source.lock.yaml` and review before staging — confirm only the expected fields changed.

**Step 6: Commit**

```bash
git add internal/install/embedded_skill_ingestion.go internal/install/embedded_skill_catalog_entry.go internal/install/embedded_skill_ingestion_test.go internal/embed/defaults/plugins/catalog.yaml external-skills-source.lock.yaml
git commit -m "feat: wire supported_handoff_schemas through embedded-skill ingestion"
```

---

### Task 7: Fix existing tests whose premise Task 1-6 just corrected

These tests currently encode the bug (asserting `brainstorming`/`openspec-propose` are compatible weapons for their roles). Fixing the code without fixing these tests would leave the suite lying about what's actually true — do not skip any of these.

**Files:**
- Modify: `internal/install/role_provider_migration_test.go`

**Step 1: `TestPlanRoleProviderMigrationSeparatesRoleFromProviderCandidates`**

Change the discovery block's assertions (currently `discovery.Resolved.Provider.ID == "brainstorming"`, `Source == "embedded"`):

```go
	discovery := preview.Entries[0]
	assert.Equal(t, "discovery", discovery.Slot)
	assert.Equal(t, "ranger", discovery.RoleName)
	assert.Equal(t, "brainstorming", discovery.CurrentProviderID)
	require.NotEmpty(t, discovery.Candidates)
	assert.Empty(t, discovery.ResolutionError)
	// brainstorming declares no supported_handoff_schemas (honest — see
	// external-skills-source/brainstorming/strategist.yaml) so it is
	// correctly incompatible; resolution falls to the native ranger
	// fallback candidate instead (Task 4).
	assert.Equal(t, "ranger", discovery.Resolved.Provider.ID)
	assert.Equal(t, "native_role", string(discovery.Resolved.Provider.Source))
	assert.True(t, discovery.Resolved.Compatibility.Compatible)
```

**Step 2: `TestPlanRoleProviderMigrationIsFullyResolvedForDefaultSlots`**

Change:
```go
	assert.Contains(t, rendered, "resolved -> brainstorming")
```
to:
```go
	assert.Contains(t, rendered, "resolved -> ranger")
```

**Step 3: `TestRoleProviderMigrationPreviewEvidenceRecordsOneEventPerSlot`**

Change:
```go
	assert.Equal(t, "brainstorming", events[0].Attributes["strategist.role_binding.provider_id"])
```
to:
```go
	assert.Equal(t, "ranger", events[0].Attributes["strategist.role_binding.provider_id"])
```

**Step 4: `TestPlanRoleProviderMigrationValidatesOpenspecProposeAsArchivistMigrationCase`**

This test's entire premise — that switching `active.slots.refinement` to `openspec-propose` is "a legitimate, compatible binding" — is the bug mission `20260914-role-weapon-structure-review` hit live. Rewrite the docstring and body to assert the corrected behavior:

```go
// TestPlanRoleProviderMigrationReportsOpenspecProposeIncompatibleForArchivist
// replaces the old TestPlanRoleProviderMigrationValidatesOpenspecProposeAsArchivistMigrationCase,
// which asserted the opposite of what's true: openspec-propose passing
// canonical_role/role_contract_version compatibility does not mean its real
// output matches Archivist's contract. Confirmed live in mission
// 20260914-role-weapon-structure-review: it writes OpenSpec's native
// openspec/changes/<name>/ shape, not
// <base_path>/refined/<mission_id>/{analysis,proposal,design,tasks}.md +
// handoff-archivist-to-sniper.schema.yaml. The handoff_schema dimension
// (Task 1) now catches this: openspec-propose declares no
// supported_handoff_schemas (Task 5), so it is correctly incompatible, and
// resolution falls back to native archivist regardless of what
// active.slots.refinement names — exactly the discovery-side precedent
// already established for brainstorming/ranger
// (.analysis/done/20260728-ranger-drift-eval/).
func TestPlanRoleProviderMigrationReportsOpenspecProposeIncompatibleForArchivist(t *testing.T) {
	t.Parallel()

	withNativeArchivist, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "archivist",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	require.True(t, withNativeArchivist.FullyResolved())

	withOpenspecPropose, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "openspec-propose",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	require.True(t, withOpenspecPropose.FullyResolved(),
		"resolution must still succeed by falling back to native archivist, even though the configured openspec-propose is incompatible")

	refinementNative := withNativeArchivist.Entries[1]
	refinementFallback := withOpenspecPropose.Entries[1]
	assert.Equal(t, "archivist", refinementNative.Resolved.Provider.ID)
	assert.Equal(t, "archivist", refinementFallback.Resolved.Provider.ID,
		"must resolve to native archivist, not openspec-propose, despite active.slots.refinement naming it")
	assert.Equal(t, "native_role", string(refinementFallback.Resolved.Provider.Source))
	assert.True(t, refinementFallback.Resolved.Compatibility.Compatible)
	assert.Equal(t, "openspec-propose", refinementFallback.CurrentProviderID,
		"CurrentProviderID still reflects the configured (incompatible) value — only Resolved changes")

	assert.Equal(t, withNativeArchivist.Entries[2].Resolved, withOpenspecPropose.Entries[2].Resolved,
		"execution slot resolution must be unaffected by the refinement slot's provider choice")
}
```

**Step 5: `TestPlanRoleProviderMigrationValidatesOpenspecExploreAsRangerMigrationCase`**

Same treatment for the discovery side — rewrite:

```go
// TestPlanRoleProviderMigrationReportsOpenspecExploreIncompatibleForRanger
// replaces the old TestPlanRoleProviderMigrationValidatesOpenspecExploreAsRangerMigrationCase.
// openspec-explore's canonical_role is ranger (ADR-0036) but it declares no
// supported_handoff_schemas (Task 5) — same structural mismatch as
// brainstorming, per .analysis/done/20260728-ranger-drift-eval/. Resolution
// must fall back to native ranger regardless of what active.slots.discovery
// names.
func TestPlanRoleProviderMigrationReportsOpenspecExploreIncompatibleForRanger(t *testing.T) {
	t.Parallel()

	withBrainstorming, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "archivist",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	require.True(t, withBrainstorming.FullyResolved())

	withOpenspecExplore, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "openspec-explore",
		"refinement": "archivist",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	require.True(t, withOpenspecExplore.FullyResolved())

	discoveryDefault := withBrainstorming.Entries[0]
	discoveryFallback := withOpenspecExplore.Entries[0]
	assert.Equal(t, "ranger", discoveryDefault.Resolved.Provider.ID)
	assert.Equal(t, "ranger", discoveryFallback.Resolved.Provider.ID,
		"must resolve to native ranger, not openspec-explore, despite active.slots.discovery naming it")
	assert.Equal(t, "native_role", string(discoveryFallback.Resolved.Provider.Source))
	assert.Equal(t, "openspec-explore", discoveryFallback.CurrentProviderID)
}
```

**Step 6: `TestApplyRoleProviderMigrationActivatesFullyResolvedBindings`**

Change:
```go
	discovery, ok := store.Binding("discovery")
	require.True(t, ok)
	assert.Equal(t, "brainstorming", discovery.InstalledInstanceID)
```
to:
```go
	discovery, ok := store.Binding("discovery")
	require.True(t, ok)
	assert.Equal(t, "ranger", discovery.InstalledInstanceID)
```

Note the `store.Inventory.Instances` seed list in this test includes `{ID: "brainstorming", ...}` but not `{ID: "ranger", ...}` — add it:
```go
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: "brainstorming", State: lifecycle.StateActive, LastKnownGood: true},
		{ID: "ranger", State: lifecycle.StateActive, LastKnownGood: true},
		{ID: "archivist", State: lifecycle.StateActive, LastKnownGood: true},
		{ID: "sniper", State: lifecycle.StateActive, LastKnownGood: true},
	}
```

**Step 7: `TestActivateRoleProviderMigrationResolvesBindingsForPersistence`**

Change:
```go
	assert.Equal(t, "brainstorming", discoveryBinding.InstalledInstanceID)
```
to:
```go
	assert.Equal(t, "ranger", discoveryBinding.InstalledInstanceID)
```

**Step 8: Run the full package test suite**

Run: `go test ./internal/domain/... ./internal/install/... ./internal/plugins/... -v`
Expected: PASS across the board. If anything else fails, it's an assertion this plan's enumeration missed — read the failure, confirm it's genuinely about the same brainstorming/openspec-propose-is-compatible premise, and fix it the same way (do not weaken an assertion just to make it pass; confirm the new expected value is actually correct per this task's rationale).

**Step 9: Commit**

```bash
git add internal/install/role_provider_migration_test.go
git commit -m "test: correct role/provider migration tests to expect native fallback for incompatible weapons"
```

---

### Task 8: Dynamic, compatibility-driven Wizard prompt options

Today `promptSlots` (`internal/install/wizard_prompts.go`) offers a hardcoded option list for refinement (`[refinementDefault, "openspec-explore"]`) and discovery (`[discoveryDefault]`) — neither reflects the compatibility resolution Tasks 1-6 now compute. This task makes the prompt option list itself compatibility-driven, so an incompatible weapon (like `openspec-propose` today) simply stops being offered, and the native role becomes the shown default — for both slots, via the same mechanism, satisfying the "discovery shown but effectively locked to native" requirement as a natural consequence rather than a special case.

**Files:**
- Modify: `internal/install/wizard_prompts.go`
- Modify: `internal/install/wizard.go:103` (call site — needs `catalog` passed through)
- Test: `internal/install/wizard_prompts_test.go` (new file — none exists today; check first with `find internal/install -iname 'wizard_prompts_test.go'`)

**Step 1: Write the failing test**

Create `internal/install/wizard_prompts_test.go`:

```go
package install

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompatibleProviderOptionsPrefersDefaultCompatibleCandidate(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "archivist", RiskScore: "write_analysis", CompatibilitySource: "native_role"},
		{
			ID: "openspec-propose", RiskScore: "write_analysis", CanonicalRole: "archivist",
			CompatibilitySource: "embedded", Default: true,
			SupportedHandoffSchemas: []string{"schemas/handoff-archivist-to-sniper.schema.yaml"},
		},
	}}

	ids, defaultID := compatibleProviderOptions(catalog, "archivist", "schemas/handoff-archivist-to-sniper.schema.yaml", "archivist")
	assert.ElementsMatch(t, []string{"archivist", "openspec-propose"}, ids)
	assert.Equal(t, "openspec-propose", defaultID)
}

func TestCompatibleProviderOptionsExcludesIncompatibleCandidate(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "archivist", RiskScore: "write_analysis", CompatibilitySource: "native_role"},
		{
			ID: "openspec-propose", RiskScore: "write_analysis", CanonicalRole: "archivist",
			CompatibilitySource: "embedded", Default: true,
			// no SupportedHandoffSchemas declared — today's real state
		},
	}}

	ids, defaultID := compatibleProviderOptions(catalog, "archivist", "schemas/handoff-archivist-to-sniper.schema.yaml", "archivist")
	assert.Equal(t, []string{"archivist"}, ids)
	assert.Equal(t, "archivist", defaultID)
}

func TestCompatibleProviderOptionsFallsBackWhenCatalogHasNoNativeEntry(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "openspec-propose", RiskScore: "write_analysis", CanonicalRole: "archivist", CompatibilitySource: "embedded"},
	}}

	ids, defaultID := compatibleProviderOptions(catalog, "archivist", "schemas/handoff-archivist-to-sniper.schema.yaml", "archivist")
	assert.Equal(t, []string{"archivist"}, ids, "falls back to the passed-in fallbackID even with no catalog entry for it, keeping the wizard usable")
	assert.Equal(t, "archivist", defaultID)
}
```

**Step 2: Run to verify it fails**

Run: `go test ./internal/install/... -run TestCompatibleProviderOptions -v`
Expected: compile error — `compatibleProviderOptions` doesn't exist yet.

**Step 3: Implement `compatibleProviderOptions` and wire it into `promptSlots`**

Add to `internal/install/wizard_prompts.go` (needs a new import: `"github.com/SergioLacerda/strategist-skill/internal/domain"`):

```go
// compatibleProviderOptions returns the catalog candidate IDs for roleName
// that CheckRoleCompatibility reports compatible against handoffSchema, plus
// which one should be pre-selected: whichever compatible candidate is
// marked default in the catalog, or the first compatible one otherwise.
// When nothing is compatible (e.g. every external weapon for this role is
// honestly declared unable to produce the role's handoff shape), it falls
// back to a single-item list naming the catalog's native_role candidate for
// roleName, or fallbackID if the catalog has none — either way the wizard
// stays usable and the operator is not offered a weapon known not to work.
func compatibleProviderOptions(catalog pluginCatalog, roleName, handoffSchema, fallbackID string) (ids []string, defaultID string) {
	role := domain.RoleContract{
		SchemaVersion: domain.RoleContractSchemaVersion,
		Role:          roleName,
		HandoffSchema: handoffSchema,
	}
	var nativeID string
	for _, candidate := range providerContractsForRole(catalog, roleName) {
		if candidate.Source == domain.ProviderSourceNativeRole {
			nativeID = candidate.ID
		}
		if !candidate.CheckRoleCompatibility(role).Compatible {
			continue
		}
		ids = append(ids, candidate.ID)
		if candidate.Default {
			defaultID = candidate.ID
		}
	}
	if defaultID == "" && len(ids) > 0 {
		defaultID = ids[0]
	}
	if len(ids) == 0 {
		if nativeID == "" {
			nativeID = fallbackID
		}
		ids = []string{nativeID}
		defaultID = nativeID
	}
	return ids, defaultID
}
```

Change `promptSlots`'s signature (add `catalog pluginCatalog` parameter) and body:

```go
func promptSlots(p Prompter, b i18n.WizardStrings, catalog pluginCatalog, providerRisk map[string]string) (discovery, refinement, execution string, err error) {
	fmt.Println(b.HeaderSlots)

	discoveryIDs, discoveryDefault := compatibleProviderOptions(catalog, "ranger", roleHandoffSchema["ranger"], "ranger")
	discovery, err = promptProvider(p, b.PromptDiscovery, discoveryDefault, discoveryIDs, b.LabelCustomInput, providerRisk, "write_analysis", "discovery")
	if err != nil {
		return "", "", "", err
	}

	refinementIDs, refinementDefault := compatibleProviderOptions(catalog, "archivist", roleHandoffSchema["archivist"], "archivist")
	refinement, err = promptProvider(p, b.PromptRefinement, refinementDefault, refinementIDs, b.LabelCustomInput, providerRisk, "write_analysis", "refinement")
	if err != nil {
		return "", "", "", err
	}

	if _, err = promptProvider(p, b.PromptExecution, nativeExecutionProvider, []string{nativeExecutionProvider}, b.LabelCustomInput, providerRisk, "controlled", "execution"); err != nil {
		return "", "", "", err
	}
	return discovery, refinement, nativeExecutionProvider, nil
}
```

Remove the now-unused `discoveryDefault := defaultSkillBySlot["discovery"]` / `refinementDefault := defaultSkillBySlot["refinement"]` local declarations (the ones this replaces) — check `defaultSkillBySlot` still has other callers before deleting the map itself (grep `defaultSkillBySlot` across `internal/install/`); leave the map in place if anything else still reads it.

Update the call site in `internal/install/wizard.go:103`:

```go
	discovery, refinement, execution, err := promptSlots(p, b, catalog, providerRisk)
```

(`catalog` is already in scope there — it's loaded at `wizard.go:88` before this call.)

**Step 4: Run to verify it passes**

Run: `go test ./internal/install/... -run TestCompatibleProviderOptions -v`
Expected: PASS.

Run: `go build ./...`
Expected: clean — confirms the `promptSlots` signature change didn't break any other call site (there should be exactly one, `runWizard`).

**Step 5: Check and fix any wizard test relying on the old static option list**

Run: `go test ./internal/install/... -run TestRunWizard -v`

If any scripted-prompter test in `wizard_test.go` asserts a specific option list shown for discovery/refinement (search for `"openspec-explore"` or `PromptRefinement`/`PromptDiscovery` in that file), update its expected options to match what `compatibleProviderOptions` now computes against the real embedded catalog (post Task 6: `discovery → ["ranger"]` only, since `brainstorming`/`openspec-explore` are incompatible; `refinement → ["archivist"]` only, since `openspec-propose` is incompatible). If a test scripts a specific typed answer (e.g. `"openspec-propose"` as free-text/custom input) rather than picking from the list, that still works unchanged — `SelectOrInput` always allows custom text regardless of the listed options — no test change needed for that shape.

**Step 6: Commit**

```bash
git add internal/install/wizard_prompts.go internal/install/wizard.go internal/install/wizard_prompts_test.go
git commit -m "feat: make wizard slot prompts compatibility-driven instead of a static option list"
```

(If Step 5 required test fixes, include the modified test file(s) in this commit too.)

---

### Task 9: Recompile this workspace and verify the migration preview

This applies the finished code to this repository's own dogfooded `.strategist/` instance and confirms the original blocked mission can proceed.

**Step 1: Rebuild and reinstall**

```bash
make install
```
(or `go build -o $(go env GOPATH)/bin/strategist ./cmd/strategist` if there's no `make install` target — check `Makefile` first: `grep -n "^install:" Makefile`)

**Step 2: Re-run install in this workspace**

```bash
strategist install
```

When prompted for refinement, confirm the option list now shows only `archivist` (no `openspec-propose` offered) — select it. Confirm discovery shows only `ranger`.

**Step 3: Verify**

```bash
strategist check
```

Expected: `SLOTS` shows `refinement archivist kind=native role` (no more `ask_required` `fallback=archivist(native_role)` annotation — there's nothing left to fall back from). `WEAPON LINKS` still reports `openspec-propose→archivist ok` and `brainstorming→ranger ok` — that section is the pre-existing structural check (KF-10, `check_weapon_bindings.go`, untouched by this plan; see design.md Non-goals) and is a different, narrower claim ("the manifest fields structurally pair up") than what this plan added ("the weapon's real output shape matches").

```bash
git diff .strategist/active.yaml
```
Expected: `slots.refinement: archivist` (was `openspec-propose`).

**Step 4: Confirm the originally-blocked mission can proceed**

Re-attempt the `/strategist` refinement invocation for mission `20260914-role-weapon-structure-review` (the one that emitted `role_invocation_failed` earlier in this session) and confirm refinement now runs via native Archivist without hitting a provider mismatch.

**Step 5: Commit**

```bash
git add .strategist/active.yaml
git commit -m "fix: switch this workspace's refinement slot to native archivist"
```

(Confirm no other `.strategist/` files changed unexpectedly — `git status --porcelain` before staging; if `strategist install`/`compile` touched generated files under `.strategist/contracts/` etc. as part of the routine mirror refresh, that's expected and fine to include.)

---

### Task 10: Full verification

**Step 1: Build, vet, full test suite**

```bash
go build ./...
go vet ./...
go test ./... -v
```
Expected: all clean/PASS. If a `-tags spec` suite exists (per the earlier `20260728-ranger-drift-eval` precedent), also run:
```bash
go test -tags spec ./tests/spec/... -v
```

**Step 2: Review the full diff before considering this done**

```bash
git log --oneline -12
git diff main...develop -- internal/domain/role_provider_contract.go internal/install/ internal/embed/defaults/plugins/catalog.yaml external-skills-source/ docs/adr/ docs/plans/
```

Confirm every change traces back to a task above — no stray edits.

**Step 3: Update `.analysis/pending/2026-09-14-role-weapon-binding-gap-design.md`'s status**

Change the frontmatter `status: design-approved` to `status: implemented` once Tasks 1-9 are verified working, so the design doc's own header reflects reality (the file is gitignored in this repo, so this is a local-only edit, not a commit).
