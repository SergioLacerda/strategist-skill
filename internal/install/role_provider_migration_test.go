package install

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanRoleProviderMigrationSeparatesRoleFromProviderCandidates(t *testing.T) {
	t.Parallel()

	preview, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "archivist",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	require.Len(t, preview.Entries, 3)

	discovery := preview.Entries[0]
	assert.Equal(t, "discovery", discovery.Slot)
	assert.Equal(t, "ranger", discovery.RoleName)
	assert.Equal(t, "brainstorming", discovery.CurrentProviderID)
	require.NotEmpty(t, discovery.Candidates)
	assert.Empty(t, discovery.ResolutionError)
	assert.Equal(t, "brainstorming", discovery.Resolved.Provider.ID)
	assert.Equal(t, "embedded", string(discovery.Resolved.Provider.Source))
	assert.True(t, discovery.Resolved.Compatibility.Compatible)

	execution := preview.Entries[2]
	assert.Equal(t, "execution", execution.Slot)
	assert.Equal(t, "sniper", execution.RoleName)
	assert.Equal(t, "sniper", execution.Resolved.Provider.ID)
	assert.Equal(t, "native_role", string(execution.Resolved.Provider.Source))
	assert.Equal(t, "active", string(execution.Resolved.Provider.Materialization))
}

func TestPlanRoleProviderMigrationIsFullyResolvedForDefaultSlots(t *testing.T) {
	t.Parallel()

	preview, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "archivist",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	assert.True(t, preview.FullyResolved())

	rendered := preview.Preview()
	assert.Contains(t, rendered, "slot=discovery role=ranger current_provider=brainstorming")
	assert.Contains(t, rendered, "resolved -> brainstorming")
}

func TestPlanRoleProviderMigrationFlagsUnresolvedActiveSlot(t *testing.T) {
	t.Parallel()

	preview, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "does-not-exist-in-catalog",
		"refinement": "archivist",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	require.False(t, preview.FullyResolved())

	discovery := preview.Entries[0]
	assert.Contains(t, discovery.ResolutionError, "unresolved_active_slot")
	assert.Contains(t, discovery.ResolutionError, "does-not-exist-in-catalog")
	assert.Equal(t, "does-not-exist-in-catalog", discovery.CurrentProviderID)

	// Other slots stay unaffected.
	assert.Empty(t, preview.Entries[1].ResolutionError)
	assert.Empty(t, preview.Entries[2].ResolutionError)
}

func TestRoleProviderMigrationPreviewEmptyIsNotFullyResolved(t *testing.T) {
	t.Parallel()

	var preview RoleProviderMigrationPreview
	assert.False(t, preview.FullyResolved())
}

// TestRoleProviderMigrationPreviewEvidenceRecordsOneEventPerSlot backs
// tasks.md Task 6.1 ("Emit binding, resolution, collision, migration, and
// conformance evidence through existing standalone-safe telemetry/
// governance boundaries"): every slot in a migration preview must produce
// one valid, resolvable telemetry event, whether the slot resolved cleanly
// or was blocked.
func TestRoleProviderMigrationPreviewEvidenceRecordsOneEventPerSlot(t *testing.T) {
	t.Parallel()

	preview, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "archivist",
		"execution":  "sniper",
	})
	require.NoError(t, err)

	events := preview.Evidence()
	require.Len(t, events, 3)
	for i, event := range events {
		require.NoErrorf(t, event.Validate(), "entry %d", i)
	}
	assert.Equal(t, "strategist.role_binding.resolved", events[0].Name)
	assert.Equal(t, "brainstorming", events[0].Attributes["strategist.role_binding.provider_id"])
}

// TestRoleProviderMigrationPreviewEvidenceRecordsBlockedOutcome proves the
// evidence path also covers a blocked resolution (an unresolved
// active.slots entry), not only the happy path.
func TestRoleProviderMigrationPreviewEvidenceRecordsBlockedOutcome(t *testing.T) {
	t.Parallel()

	preview, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "does-not-exist-in-catalog",
		"refinement": "archivist",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	require.False(t, preview.FullyResolved())

	events := preview.Evidence()
	require.Len(t, events, 3)
	require.NoError(t, events[0].Validate())
	assert.Equal(t, "strategist.role_binding.unresolved_active_slot", events[0].Name)
	assert.Empty(t, events[0].Attributes["strategist.role_binding.provider_id"])
}

func TestPlanRoleProviderMigrationAcceptsAffiliatedOpenspecProposeForArchivist(t *testing.T) {
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
	require.True(t, withOpenspecPropose.FullyResolved())

	refinementNative := withNativeArchivist.Entries[1]
	refinementFallback := withOpenspecPropose.Entries[1]
	assert.Equal(t, "archivist", refinementNative.Resolved.Provider.ID)
	assert.Equal(t, "openspec-propose", refinementFallback.Resolved.Provider.ID)
	assert.Equal(t, "embedded", string(refinementFallback.Resolved.Provider.Source))
	assert.True(t, refinementFallback.Resolved.Compatibility.Compatible)
	assert.Equal(t, "openspec-propose", refinementFallback.CurrentProviderID,
		"CurrentProviderID still reflects the configured (incompatible) value — only Resolved changes")

	assert.Equal(t, withNativeArchivist.Entries[2].Resolved, withOpenspecPropose.Entries[2].Resolved,
		"execution slot resolution must be unaffected by the refinement slot's provider choice")
}

func TestPlanRoleProviderMigrationAcceptsAffiliatedOpenspecExploreForRanger(t *testing.T) {
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
	assert.Equal(t, "brainstorming", discoveryDefault.Resolved.Provider.ID)
	assert.Equal(t, "openspec-explore", discoveryFallback.Resolved.Provider.ID)
	assert.Equal(t, "embedded", string(discoveryFallback.Resolved.Provider.Source))
	assert.Equal(t, "openspec-explore", discoveryFallback.CurrentProviderID)
}

// TestApplyRoleProviderMigrationRefusesPartialMigration proves tasks.md Task
// 4.2's "apply only a fully resolved binding": a preview containing any
// unresolved entry must not activate any binding at all.
func TestApplyRoleProviderMigrationRefusesPartialMigration(t *testing.T) {
	t.Parallel()

	preview, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "does-not-exist-in-catalog",
		"refinement": "archivist",
		"execution":  "sniper",
		// discovery's active.slots value names a provider absent from the
		// catalog entirely (unresolved_active_slot), leaving this preview
		// not fully resolved regardless of how any individual role's
		// candidates resolve.
	})
	require.NoError(t, err)
	require.False(t, preview.FullyResolved())

	store := lifecycle.NewStore()
	err = ApplyRoleProviderMigration(store, preview, func(domain.SlotBinding, domain.InstalledInstance) bool { return true })
	require.Error(t, err)
	require.ErrorContains(t, err, "role_provider_migration_not_fully_resolved")
}

// TestApplyRoleProviderMigrationActivatesFullyResolvedBindings proves
// tasks.md Task 3.3/4.2's staged-activation half: applying a fully resolved
// migration moves the discovery/refinement bindings to their resolved
// Providers through the same lifecycle.Store Begin/Stage/Probe/Activate machinery
// applyPluginOnboardingPlan already exercises for legacy bindings.
func TestApplyRoleProviderMigrationActivatesFullyResolvedBindings(t *testing.T) {
	t.Parallel()

	preview, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "archivist",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	require.True(t, preview.FullyResolved())

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: "brainstorming", State: lifecycle.StateActive, LastKnownGood: true},
		{ID: "ranger", State: lifecycle.StateActive, LastKnownGood: true},
		{ID: "archivist", State: lifecycle.StateActive, LastKnownGood: true},
		{ID: "sniper", State: lifecycle.StateActive, LastKnownGood: true},
	}
	store.Bindings = []domain.SlotBinding{
		{Slot: "discovery", InstalledInstanceID: "native-ranger", Generation: 1, Status: "enabled"},
		{Slot: "refinement", InstalledInstanceID: "native-archivist", Generation: 4, Status: "enabled"},
		{Slot: "execution", InstalledInstanceID: "native-sniper", Generation: 2, Status: "enabled"},
	}

	require.NoError(t, ApplyRoleProviderMigration(store, preview, func(domain.SlotBinding, domain.InstalledInstance) bool {
		return true
	}))

	discovery, ok := store.Binding("discovery")
	require.True(t, ok)
	// brainstorming is no longer compatible (no declared
	// supported_handoff_schemas) — resolution falls back to native ranger.
	assert.Equal(t, "brainstorming", discovery.InstalledInstanceID)
	assert.Equal(t, int64(2), discovery.Generation)

	refinement, ok := store.Binding("refinement")
	require.True(t, ok)
	assert.Equal(t, "archivist", refinement.InstalledInstanceID)
	assert.Equal(t, int64(5), refinement.Generation)
}

// TestApplyRoleProviderMigrationRollsBackOnProbeFailure proves the
// last-known-good rollback half of tasks.md Task 3.3: a failing probe for
// any slot leaves that slot's binding untouched, via the same
// Store.Rollback path applyPluginOnboardingPlan already relies on.
func TestApplyRoleProviderMigrationRollsBackOnProbeFailure(t *testing.T) {
	t.Parallel()

	preview, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "archivist",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	require.True(t, preview.FullyResolved())

	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		// The slots' prior bindings must themselves be known instances —
		// otherwise Rollback can't find them by ID and falls back to
		// whichever instance is LastKnownGood, which would mask this test's
		// own intent (proving the ORIGINAL binding survives a failed probe).
		{ID: "native-ranger", State: lifecycle.StateActive, LastKnownGood: true},
		{ID: "brainstorming", State: lifecycle.StateActive},
		{ID: "ranger", State: lifecycle.StateActive},
		{ID: "archivist", State: lifecycle.StateActive},
		{ID: "sniper", State: lifecycle.StateActive},
	}
	store.Bindings = []domain.SlotBinding{
		{Slot: "discovery", InstalledInstanceID: "native-ranger", Generation: 1, Status: "enabled"},
		{Slot: "refinement", InstalledInstanceID: "native-archivist", Generation: 4, Status: "enabled"},
		{Slot: "execution", InstalledInstanceID: "native-sniper", Generation: 2, Status: "enabled"},
	}

	err = ApplyRoleProviderMigration(store, preview, func(domain.SlotBinding, domain.InstalledInstance) bool {
		return false
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "activation_requires_successful_probe")

	discovery, ok := store.Binding("discovery")
	require.True(t, ok)
	assert.Equal(t, "native-ranger", discovery.InstalledInstanceID, "failed slot must keep its prior binding")
}

// TestActivateRoleProviderMigrationDrivesRealStagedActivation proves tasks.md
// Task 5 (.analysis/refined/20260913-embedded-skill-directory-catalog):
// activateRoleProviderMigration — the function runWizard now calls for real
// — actually exercises the lifecycle.Store staged-activation machinery for a
// realistic, fully-resolved preview, not just a display-only preview.
// Rollback-on-probe-failure for this exact function is covered by
// TestApplyRoleProviderMigrationRollsBackOnProbeFailure, since
// activateRoleProviderMigration is a thin, direct wrapper around
// ApplyRoleProviderMigration with the same probe contract.
func TestActivateRoleProviderMigrationDrivesRealStagedActivation(t *testing.T) {
	t.Parallel()

	preview, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "archivist",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	require.True(t, preview.FullyResolved())

	_, err = activateRoleProviderMigration("", domain.PluginLock{}, preview)
	require.NoError(t, err)
}

// TestActivateRoleProviderMigrationResolvesBindingsForPersistence proves
// ADR-0037's DEC-001: activateRoleProviderMigration returns the resolved
// discovery/refinement bindings so the caller (applyWizardConfig) can persist
// them to plugins.lock, instead of the state being rebuilt from scratch and
// discarded on every `strategist install` invocation. Persistence itself
// (the actual disk write, gated on active.yaml landing first) is
// applyWizardConfig's responsibility — see installer_config_whitebox_test.go.
func TestActivateRoleProviderMigrationResolvesBindingsForPersistence(t *testing.T) {
	t.Parallel()

	preview, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "archivist",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	require.True(t, preview.FullyResolved())

	lockFile, err := activateRoleProviderMigration("", domain.PluginLock{SchemaVersion: "test", GraphDigest: "sha256:test"}, preview)
	require.NoError(t, err)
	assert.Equal(t, "sha256:test", lockFile.Lock.GraphDigest)

	discoveryBinding, ok := findSlotBinding(lockFile.Bindings, "discovery")
	require.True(t, ok)
	assert.Equal(t, "brainstorming", discoveryBinding.InstalledInstanceID)
	assert.Equal(t, domain.SlotBindingModeCustom, discoveryBinding.Mode, "today's only implemented pipeline is Custom")
	refinementBinding, ok := findSlotBinding(lockFile.Bindings, "refinement")
	require.True(t, ok)
	assert.Equal(t, "archivist", refinementBinding.InstalledInstanceID)
	assert.Equal(t, domain.SlotBindingModeCustom, refinementBinding.Mode)
	_, ok = findSlotBinding(lockFile.Bindings, "execution")
	assert.False(t, ok, "role/provider persistence is scoped to discovery and refinement")
}

// TestActivateRoleProviderMigrationIsIdempotentAcrossPersistedRuns proves
// ADR-0037's DEC-003: re-running activation against a strategistDir whose
// plugins.lock already reflects the resolution must not re-resolve, switch,
// or otherwise churn the persisted binding — a bound weapon is immutable
// absent an actual change to the resolution inputs.
func TestActivateRoleProviderMigrationIsIdempotentAcrossPersistedRuns(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	preview, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "archivist",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	require.True(t, preview.FullyResolved())

	first, err := activateRoleProviderMigration(dir, domain.PluginLock{SchemaVersion: "test", GraphDigest: "sha256:test"}, preview)
	require.NoError(t, err)
	require.NoError(t, writePluginLockFile(dir, first))

	second, err := activateRoleProviderMigration(dir, domain.PluginLock{SchemaVersion: "test", GraphDigest: "sha256:test"}, preview)
	require.NoError(t, err)

	assert.Equal(t, first, second, "re-running activation against an already-persisted, unchanged resolution must be idempotent")
}

func findSlotBinding(bindings []domain.SlotBinding, slot string) (domain.SlotBinding, bool) {
	for _, b := range bindings {
		if b.Slot == slot {
			return b, true
		}
	}
	return domain.SlotBinding{}, false
}

// TestActivateRoleProviderMigrationHandlesNoCurrentProvider proves a slot
// with no prior provider (CurrentProviderID empty) is bound directly rather
// than erroring — activateRoleProviderMigration must not assume every slot
// already has a current binding.
func TestActivateRoleProviderMigrationHandlesNoCurrentProvider(t *testing.T) {
	t.Parallel()

	// "execution" is intentionally left unset: sniper is the sole catalog
	// candidate for the sniper role, so it still resolves unambiguously with
	// no preference, unlike "refinement" (archivist vs openspec-explore).
	preview, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "archivist",
	})
	require.NoError(t, err)
	require.True(t, preview.FullyResolved())

	execution := preview.Entries[2]
	require.Equal(t, "execution", execution.Slot)
	require.Empty(t, execution.CurrentProviderID)

	_, err = activateRoleProviderMigration("", domain.PluginLock{}, preview)
	require.NoError(t, err)
}

func TestPlanRoleProviderMigration_LoadRoleSlotMapErrorPropagates(t *testing.T) {
	t.Parallel()

	_, err := PlanRoleProviderMigration(brokenPathExtractor{missing: roleSlotMapPath}, map[string]string{"discovery": "brainstorming"})
	require.Error(t, err)
	require.ErrorContains(t, err, "role/provider migration")
}

func TestPlanRoleProviderMigration_LoadCatalogErrorPropagates(t *testing.T) {
	t.Parallel()

	_, err := PlanRoleProviderMigration(brokenPathExtractor{missing: pluginCatalogPath}, map[string]string{"discovery": "brainstorming"})
	require.Error(t, err)
	require.ErrorContains(t, err, "role/provider migration")
}

func TestPlanRoleProviderMigration_LoadRoleConfigErrorPropagates(t *testing.T) {
	t.Parallel()

	_, err := PlanRoleProviderMigration(brokenPathExtractor{missing: "roles/ranger.yaml"}, map[string]string{"discovery": "brainstorming"})
	require.Error(t, err)
	require.ErrorContains(t, err, "role/provider migration: slot discovery")
}

// brokenPathExtractor delegates to the real embedded defaults for every path
// except missing, which it fails to read — for exercising one specific
// downstream-read error branch at a time without a hand-authored fixture
// tree.
type brokenPathExtractor struct {
	missing string
}

func (b brokenPathExtractor) Extract(_ string, _ bool) error { return nil }

func (b brokenPathExtractor) ReadFile(relPath string) ([]byte, error) {
	if relPath == b.missing {
		return nil, fmt.Errorf("brokenPathExtractor: %s deliberately unreadable", relPath)
	}
	return defaultsExtractor{}.ReadFile(relPath)
}

func TestActivateRoleProviderMigration_ReadLockFileErrorPropagates(t *testing.T) {
	t.Parallel()

	preview, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery": "brainstorming", "refinement": "archivist", "execution": "sniper",
	})
	require.NoError(t, err)
	require.True(t, preview.FullyResolved())

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, pluginLockFileName), []byte("not: [valid"), 0o644))

	_, err = activateRoleProviderMigration(dir, domain.PluginLock{}, preview)
	require.Error(t, err)
	require.ErrorContains(t, err, "activate role/provider migration")
}

func TestActivateRoleProviderMigration_ApplyErrorPropagates(t *testing.T) {
	t.Parallel()

	preview, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "does-not-exist-in-catalog",
		"refinement": "archivist",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	require.False(t, preview.FullyResolved())

	_, err = activateRoleProviderMigration("", domain.PluginLock{}, preview)
	require.Error(t, err)
	require.ErrorContains(t, err, "role_provider_migration_not_fully_resolved")
}

// TestActivateRoleProviderMigration_SwitchesWhenCurrentDiffersFromResolved
// exercises the one path no other test in this file reaches: a slot whose
// seeded current binding (from CurrentProviderID) differs from the newly
// resolved Provider, forcing applyPluginBinding through switchPluginBinding
// (Begin/Stage/Probe/Activate) instead of either the no-op or
// create-fresh-binding shortcuts. Built directly from a hand-authored
// preview rather than PlanRoleProviderMigration's real catalog resolution
// (which, for this repo's catalog, treats a preference that doesn't match
// one of several role-compatible candidates as role_binding_ambiguous
// rather than silently overriding it) — the resolution algorithm itself is
// covered elsewhere; this test only needs activateRoleProviderMigration's
// own seed/apply wiring.
func TestActivateRoleProviderMigration_SwitchesWhenCurrentDiffersFromResolved(t *testing.T) {
	t.Parallel()

	preview := RoleProviderMigrationPreview{Entries: []RoleProviderPreviewEntry{
		{
			Slot: "discovery", RoleName: "ranger", CurrentProviderID: "openspec-explore",
			Resolved: domain.ProviderBinding{
				Provider:      domain.ProviderContract{ID: "brainstorming"},
				Compatibility: domain.CompatibilityResult{Compatible: true},
			},
		},
	}}
	require.True(t, preview.FullyResolved())

	lockFile, err := activateRoleProviderMigration("", domain.PluginLock{}, preview)
	require.NoError(t, err)
	binding, ok := findSlotBinding(lockFile.Bindings, "discovery")
	require.True(t, ok)
	assert.Equal(t, "brainstorming", binding.InstalledInstanceID)
}
