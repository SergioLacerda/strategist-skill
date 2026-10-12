package onboarding

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectorProbeAcceptsStructurallyReadyHostBinding(t *testing.T) {
	t.Parallel()

	outcome := ConnectorProbe(
		domain.SlotBinding{
			Slot: string(domain.SlotRefinement), InstalledInstanceID: "writing-plans",
			RuntimeKind: domain.RankedRuntimeHost, ConnectorID: "strategist-host", Entrypoint: "refine",
		},
		domain.InstalledInstance{ID: "writing-plans", ConnectorID: "strategist-host"},
	)

	assert.Equal(t, domain.ReadinessReady, outcome.Status)
	assert.Equal(t, "host_package_structural_ready", outcome.ReasonCode)
	assert.Contains(t, outcome.Detail, "live invocation remains unverified")
}

func TestRoleProviderMigrationActivatesHostProviderWithCatalogIdentity(t *testing.T) {
	t.Parallel()

	preview := RoleProviderMigrationPreview{Entries: []RoleProviderPreviewEntry{
		{
			Slot: "refinement", RoleName: "archivist", CurrentProviderID: "openspec-propose",
			Resolved: domain.ProviderBinding{Provider: domain.ProviderContract{
				ID: "writing-plans", Version: "1.0.0", Source: domain.ProviderSourceEmbedded,
				Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeHost, HostAPI: "strategist-host-skill/v1"},
			}, Compatibility: domain.CompatibilityResult{Compatible: true}},
		},
	}}

	store := lifecycle.NewStore()
	SeedRoleProviderMigrationEntry(store, preview.Entries[0])
	require.NoError(t, ApplyRoleProviderMigrationWithResult(store, preview, ConnectorProbe))

	binding, ok := store.Binding("refinement")
	require.True(t, ok)
	assert.Equal(t, "writing-plans", binding.InstalledInstanceID)
	assert.Equal(t, domain.RankedRuntimeHost, binding.RuntimeKind)
	assert.Equal(t, "strategist-host", binding.ConnectorID)
	assert.Equal(t, "refine", binding.Entrypoint)
	assert.Equal(t, domain.SlotBindingModeCustom, binding.Mode)

	instance, ok := store.Instance("writing-plans")
	require.True(t, ok)
	assert.Equal(t, "strategist-host", instance.ConnectorID)
	assert.Equal(t, "catalog_host_package_structural_validation", instance.VerificationEvidence)
}

func TestPluginOnboardingCarriesCatalogHostRuntimeIntoRoleMigration(t *testing.T) {
	t.Parallel()

	catalog, err := plugincatalog.LoadPluginCatalog(installtest.DefaultsExtractor{})
	require.NoError(t, err)
	// writing-plans moved to the embedded prompt bridge (ADR-0067), so the
	// catalog's remaining host-runtime Archivist-compatible candidate for the
	// discovery slot, openspec-explore, carries the host runtime now.
	plan, err := PlanPluginOnboardingWithModes(installtest.DefaultsExtractor{}, catalog, map[string]string{
		"discovery": "openspec-explore", "refinement": "openspec-propose", "execution": "sniper",
	}, map[string]string{
		"discovery": domain.SlotBindingModeCustom, "refinement": domain.SlotBindingModeRanked, "execution": domain.SlotBindingModeRanked,
	})
	require.NoError(t, err)

	for _, entry := range plan.RoleMigration.Entries {
		if entry.Slot != string(domain.SlotDiscovery) {
			continue
		}
		assert.Equal(t, "openspec-explore", entry.Resolved.Provider.ID)
		assert.Equal(t, domain.RankedRuntimeHost, entry.Resolved.Provider.Runtime.Kind)
		return
	}
	t.Fatal("discovery role migration entry not found")
}

func TestRoleProviderMigrationEnrichesConvergedPersistedHostInstance(t *testing.T) {
	t.Parallel()

	preview := RoleProviderMigrationPreview{Entries: []RoleProviderPreviewEntry{{
		Slot: "refinement", RoleName: "archivist", CurrentProviderID: "writing-plans",
		Resolved: domain.ProviderBinding{Provider: domain.ProviderContract{
			ID: "writing-plans", Version: "1.0.0", Source: domain.ProviderSourceEmbedded,
			Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeHost, HostAPI: "strategist-host-skill/v1"},
		}, Compatibility: domain.CompatibilityResult{Compatible: true}},
	}}}
	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: "writing-plans", State: lifecycle.StateActive, LastKnownGood: true}}
	store.Bindings = []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: "openspec-propose", Generation: 1, Status: "enabled"}}
	SeedRoleProviderMigrationEntry(store, preview.Entries[0])
	require.NoError(t, ApplyRoleProviderMigrationWithResult(store, preview, ConnectorProbe))

	instance, ok := store.Instance("writing-plans")
	require.True(t, ok)
	assert.Equal(t, "strategist-host", instance.ConnectorID)
	require.NotEmpty(t, instance.VerificationEvidence)
}
