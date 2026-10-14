package rankedbinding

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRankedRuntimeHelpersAndArtifacts(t *testing.T) {
	catalog := rankedFixtureCatalog()
	catalog.Providers[0].Runtime = domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded}

	assert.True(t, ProviderNeedsRankedRuntime(catalog, "brainstorming"))
	assert.False(t, ProviderNeedsRankedRuntime(catalog, "missing"))
	catalog.Providers[0].Ranked = false
	assert.False(t, ProviderNeedsRankedRuntime(catalog, "brainstorming"))
	catalog.Providers[0].Ranked = true
	catalog.Providers[0].CertificationDigest = ""
	assert.False(t, ProviderNeedsRankedRuntime(catalog, "brainstorming"))

	wc := domain.WizardConfig{}
	SetRankedSlotMode(&wc, string(domain.SlotDiscovery))
	SetRankedSlotMode(&wc, string(domain.SlotRefinement))
	SetRankedSlotMode(&wc, string(domain.SlotExecution))
	assert.Equal(t, domain.SlotBindingModeRanked, wc.DiscoveryMode)
	assert.Equal(t, domain.SlotBindingModeRanked, wc.RefinementMode)
	assert.Equal(t, domain.SlotBindingModeRanked, wc.ExecutionMode)

	empty, err := AttachWeaponBindingArtifacts(domain.PluginLockFile{})
	require.NoError(t, err)
	assert.Nil(t, empty.BindingArtifacts)

	lock := domain.PluginLockFile{Bindings: []domain.SlotBinding{{
		TaxonomyVersion: domain.CanonicalTaxonomyVersion,
		Role:            "ranger", Slot: "discovery", InstalledInstanceID: "brainstorming", WeaponVersion: "1.0.0",
		WeaponDigest: "sha256:weapon", BindingDigest: "sha256:binding", RuntimeKind: domain.RankedRuntimeEmbedded,
		ConnectorID: "strategist-embedded", Entrypoint: "discover", Origin: string(domain.WeaponOriginEmbedded),
	}}}
	withArtifacts, err := AttachWeaponBindingArtifacts(lock)
	require.NoError(t, err)
	require.Len(t, withArtifacts.BindingArtifacts, 1)
	assert.Equal(t, "brainstorming", withArtifacts.BindingArtifacts[0].WeaponID)
}

func TestPromoteRuntimeProvidersToRanked(t *testing.T) {
	catalog := rankedFixtureCatalog()
	catalog.Providers[0].Runtime = domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded}

	lock, err := PromoteRuntimeProvidersToRanked(catalog, map[string]string{
		string(domain.SlotDiscovery): "brainstorming",
	}, domain.PluginLockFile{})
	require.NoError(t, err)
	require.Len(t, lock.Bindings, 1)
	assert.Equal(t, domain.SlotBindingModeRanked, lock.Bindings[0].Mode)
}
