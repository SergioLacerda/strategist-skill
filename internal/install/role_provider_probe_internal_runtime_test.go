package install

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func internalRuntimeBinding(runtimeKind, connector string) (domain.SlotBinding, domain.InstalledInstance) {
	return domain.SlotBinding{
			Slot: string(domain.SlotRefinement), InstalledInstanceID: "writing-plans", RuntimeKind: runtimeKind,
			ConnectorID: connector, Entrypoint: "refine",
		},
		domain.InstalledInstance{ID: "writing-plans", ConnectorID: connector}
}

func TestConnectorProbeAcceptsAStructurallyReadyInternalRuntime(t *testing.T) {
	t.Parallel()
	cases := map[string]struct{ kind, connector string }{
		"embedded prompt bridge": {domain.RankedRuntimeEmbedded, "strategist-embedded"},
		"private openspec root":  {domain.RankedRuntimeOpenSpecRoot, "strategist-ranked-runtime"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			binding, instance := internalRuntimeBinding(tc.kind, tc.connector)

			outcome := connectorProbe(binding, instance)

			assert.Equal(t, domain.ReadinessReady, outcome.Status)
			assert.Equal(t, "internal_runtime_structural_ready", outcome.ReasonCode)
			assert.Contains(t, outcome.Detail, "live invocation remains unverified")
		})
	}
}

func TestConnectorProbeRejectsAnInternalRuntimeWithAnotherConnectorIdentity(t *testing.T) {
	t.Parallel()
	binding, instance := internalRuntimeBinding(domain.RankedRuntimeEmbedded, "strategist-host")

	outcome := connectorProbe(binding, instance)

	assert.Equal(t, domain.ReadinessBlocked, outcome.Status)
	assert.Equal(t, "internal_runtime_identity_mismatch", outcome.ReasonCode)
}

func TestConnectorProbeRejectsAnInternalRuntimeWithoutAnEntrypoint(t *testing.T) {
	t.Parallel()
	binding, instance := internalRuntimeBinding(domain.RankedRuntimeEmbedded, "strategist-embedded")
	binding.Entrypoint = ""

	outcome := connectorProbe(binding, instance)

	assert.Equal(t, domain.ReadinessBlocked, outcome.Status)
	assert.Equal(t, "internal_runtime_metadata_incomplete", outcome.ReasonCode)
}

// A switch from a persisted Ranked refinement binding to another certified
// Weapon is the wizard re-run on an existing installation: it must activate
// through the lifecycle probe instead of rolling the whole install back.
func TestActivateRoleProviderMigrationSwitchesBetweenCertifiedRankedWeapons(t *testing.T) {
	t.Parallel()
	catalog := embeddedDefaultsCatalog(t)
	persisted := domain.PluginLockFile{
		Inventory: domain.PluginInventory{Instances: []domain.InstalledInstance{
			{ID: "openspec-propose", State: lifecycle.StateActive, LastKnownGood: true, ConnectorID: "strategist-ranked-runtime"},
		}},
		Bindings: []domain.SlotBinding{{
			Slot: "refinement", InstalledInstanceID: "openspec-propose", Role: "archivist", Generation: 1, Status: "enabled",
			Mode: domain.SlotBindingModeRanked, RuntimeKind: domain.RankedRuntimeOpenSpecRoot,
		}},
	}
	dir := t.TempDir()
	require.NoError(t, writePluginLockFile(dir, persisted))
	preview, err := planRoleProviderMigrationWithCatalog(defaultsExtractor{}, catalog, map[string]string{
		"discovery": "brainstorming", "refinement": "writing-plans", "execution": "sniper",
	})
	require.NoError(t, err)

	got, err := activateRoleProviderMigration(dir, domain.PluginLock{}, preview)

	require.NoError(t, err)
	binding, ok := findSlotBinding(got.Bindings, "refinement")
	require.True(t, ok)
	assert.Equal(t, "writing-plans", binding.InstalledInstanceID)
	assert.Equal(t, int64(2), binding.Generation, "the switch is a generation bump of the same slot binding")
}

func TestActivateRoleProviderMigrationRefreshesAStaleInstanceIdentityFromAnOlderBinary(t *testing.T) {
	t.Parallel()
	catalog := embeddedDefaultsCatalog(t)
	// writing-plans was a host-runtime Weapon before ADR-0067; an inventory
	// written then still names the host connector.
	persisted := domain.PluginLockFile{
		Inventory: domain.PluginInventory{Instances: []domain.InstalledInstance{
			{ID: "openspec-propose", State: lifecycle.StateActive, LastKnownGood: true, ConnectorID: "strategist-ranked-runtime"},
			{ID: "writing-plans", State: lifecycle.StateActive, ConnectorID: "strategist-host", Entrypoint: "refine"},
		}},
		Bindings: []domain.SlotBinding{{
			Slot: "refinement", InstalledInstanceID: "openspec-propose", Role: "archivist", Generation: 1, Status: "enabled",
			Mode: domain.SlotBindingModeRanked, RuntimeKind: domain.RankedRuntimeOpenSpecRoot,
		}},
	}
	dir := t.TempDir()
	require.NoError(t, writePluginLockFile(dir, persisted))
	preview, err := planRoleProviderMigrationWithCatalog(defaultsExtractor{}, catalog, map[string]string{
		"discovery": "brainstorming", "refinement": "writing-plans", "execution": "sniper",
	})
	require.NoError(t, err)

	got, err := activateRoleProviderMigration(dir, domain.PluginLock{}, preview)

	require.NoError(t, err)
	var instance domain.InstalledInstance
	for _, candidate := range got.Inventory.Instances {
		if candidate.ID == "writing-plans" {
			instance = candidate
		}
	}
	assert.Equal(t, "strategist-embedded", instance.ConnectorID, "the instance identity follows the catalog, not the older inventory")
}
