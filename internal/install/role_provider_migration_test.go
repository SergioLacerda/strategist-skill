package install

import (
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

// TestPlanRoleProviderMigrationValidatesOpenspecExploreAsArchivistMigrationCase
// backs tasks.md Task 5.2 ("Validate Brainstorm/Ranger and OpenSpec/
// Archivist as migration cases without coupling Providers directly to
// downstream roles"). openspec-explore is catalogued with
// canonical_role: archivist alongside the native archivist role itself, so
// switching active.slots.refinement from "archivist" to "openspec-explore"
// must resolve as a legitimate, compatible binding for the same role — and
// the execution slot's resolution must be byte-for-byte identical either
// way, proving refinement's Provider choice has no path to influence what
// Sniper (downstream) receives.
func TestPlanRoleProviderMigrationValidatesOpenspecExploreAsArchivistMigrationCase(t *testing.T) {
	t.Parallel()

	withNativeArchivist, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "archivist",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	require.True(t, withNativeArchivist.FullyResolved())

	withOpenspecExplore, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery":  "brainstorming",
		"refinement": "openspec-explore",
		"execution":  "sniper",
	})
	require.NoError(t, err)
	require.True(t, withOpenspecExplore.FullyResolved())

	refinementNative := withNativeArchivist.Entries[1]
	refinementExternal := withOpenspecExplore.Entries[1]
	assert.Equal(t, "archivist", refinementNative.Resolved.Provider.ID)
	assert.Equal(t, "native_role", string(refinementNative.Resolved.Provider.Source))
	assert.Equal(t, "openspec-explore", refinementExternal.Resolved.Provider.ID)
	assert.Equal(t, "embedded", string(refinementExternal.Resolved.Provider.Source))
	assert.True(t, refinementExternal.Resolved.Compatibility.Compatible)
	// Both bind to the same Role — the Role, not the Provider, owns the
	// downstream handoff (proposal.md Decision 6).
	assert.Equal(t, refinementNative.Resolved.Role, refinementExternal.Resolved.Role)

	assert.Equal(t, withNativeArchivist.Entries[2].Resolved, withOpenspecExplore.Entries[2].Resolved,
		"execution slot resolution must be unaffected by the refinement slot's Provider choice")
}

// TestApplyRoleProviderMigrationRefusesPartialMigration proves tasks.md Task
// 4.2's "apply only a fully resolved binding": a preview containing any
// unresolved entry must not activate any binding at all.
func TestApplyRoleProviderMigrationRefusesPartialMigration(t *testing.T) {
	t.Parallel()

	preview, err := PlanRoleProviderMigration(defaultsExtractor{}, map[string]string{
		"discovery": "brainstorming",
		// refinement/execution left unset — with no CurrentProviderID to
		// prefer, refinement resolves ambiguously (archivist vs
		// openspec-explore), so this preview is not fully resolved.
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
// migration moves each slot's binding to its resolved Provider through the
// same lifecycle.Store Begin/Stage/Probe/Activate machinery
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

	require.NoError(t, activateRoleProviderMigration(preview))
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

	require.NoError(t, activateRoleProviderMigration(preview))
}
