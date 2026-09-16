package install

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rankedFixtureCatalog() pluginCatalog {
	return pluginCatalog{Providers: []pluginCatalogProvider{
		{
			ID: "brainstorming", RiskScore: "write_analysis", CanonicalRole: "ranger",
			CompatibilitySource: "embedded", Default: true,
			Ranked: true, CertificationDigest: "sha256:cert", RankedBindingGeneration: 1, RankedBindingStatus: "active",
		},
		{
			ID: "openspec-propose", RiskScore: "write_analysis", CanonicalRole: "archivist",
			CompatibilitySource: "embedded",
			Ranked:              true, CertificationDigest: "sha256:cert2", RankedBindingGeneration: 1, RankedBindingStatus: "active",
		},
	}}
}

func TestApplyRankedBindingChoices_ActivatesPreGeneratedRecord(t *testing.T) {
	catalog := rankedFixtureCatalog()
	wc := domain.WizardConfig{
		DiscoveryProvider: "brainstorming", RefinementProvider: "openspec-propose", ExecutionProvider: "sniper",
		DiscoveryMode: domain.SlotBindingModeRanked,
	}
	lockFile := domain.PluginLockFile{Bindings: []domain.SlotBinding{
		{Slot: "discovery", InstalledInstanceID: "brainstorming", Mode: domain.SlotBindingModeCustom, Status: "enabled"},
		{Slot: "refinement", InstalledInstanceID: "openspec-propose", Mode: domain.SlotBindingModeCustom, Status: "enabled"},
	}}

	got, err := applyRankedBindingChoices(catalog, wc, lockFile)
	require.NoError(t, err)

	byDiscovery := findBindingBySlot(t, got.Bindings, "discovery")
	assert.Equal(t, domain.SlotBindingModeRanked, byDiscovery.Mode)
	assert.Equal(t, "brainstorming", byDiscovery.InstalledInstanceID)
	assert.Equal(t, int64(1), byDiscovery.Generation)
	assert.Equal(t, "active", byDiscovery.Status)

	byRefinement := findBindingBySlot(t, got.Bindings, "refinement")
	assert.Equal(t, domain.SlotBindingModeCustom, byRefinement.Mode, "a slot the wizard resolved to Custom is left untouched")
}

// TestApplyRankedBindingChoices_ActivatesRefinementSlot is the
// 20260916-ranked-skills-end-to-end-evaluation Layer 2 coverage-gap fix:
// the refinement-slot counterpart to
// TestApplyRankedBindingChoices_ActivatesPreGeneratedRecord, which only
// ever exercised the discovery slot's Ranked activation.
func TestApplyRankedBindingChoices_ActivatesRefinementSlot(t *testing.T) {
	catalog := rankedFixtureCatalog()
	wc := domain.WizardConfig{
		DiscoveryProvider: "brainstorming", RefinementProvider: "openspec-propose", ExecutionProvider: "sniper",
		RefinementMode: domain.SlotBindingModeRanked,
	}
	lockFile := domain.PluginLockFile{Bindings: []domain.SlotBinding{
		{Slot: "discovery", InstalledInstanceID: "brainstorming", Mode: domain.SlotBindingModeCustom, Status: "enabled"},
		{Slot: "refinement", InstalledInstanceID: "openspec-propose", Mode: domain.SlotBindingModeCustom, Status: "enabled"},
	}}

	got, err := applyRankedBindingChoices(catalog, wc, lockFile)
	require.NoError(t, err)

	byRefinement := findBindingBySlot(t, got.Bindings, "refinement")
	assert.Equal(t, domain.SlotBindingModeRanked, byRefinement.Mode)
	assert.Equal(t, "openspec-propose", byRefinement.InstalledInstanceID)
	assert.Equal(t, int64(1), byRefinement.Generation)
	assert.Equal(t, "active", byRefinement.Status)

	byDiscovery := findBindingBySlot(t, got.Bindings, "discovery")
	assert.Equal(t, domain.SlotBindingModeCustom, byDiscovery.Mode, "a slot the wizard resolved to Custom is left untouched")
}

func TestApplyRankedBindingChoices_NoopWhenNoSlotIsRanked(t *testing.T) {
	catalog := rankedFixtureCatalog()
	wc := domain.WizardConfig{DiscoveryProvider: "brainstorming", RefinementProvider: "openspec-propose"}
	lockFile := domain.PluginLockFile{Bindings: []domain.SlotBinding{
		{Slot: "discovery", InstalledInstanceID: "brainstorming", Mode: domain.SlotBindingModeCustom},
	}}

	got, err := applyRankedBindingChoices(catalog, wc, lockFile)
	require.NoError(t, err)
	assert.Equal(t, lockFile, got)
}

func TestApplyRankedBindingChoices_ErrorsWhenProviderNotCertified(t *testing.T) {
	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "openspec-explore", RiskScore: "write_analysis", CanonicalRole: "ranger", CompatibilitySource: "embedded"},
	}}
	wc := domain.WizardConfig{DiscoveryProvider: "openspec-explore", DiscoveryMode: domain.SlotBindingModeRanked}

	_, err := applyRankedBindingChoices(catalog, wc, domain.PluginLockFile{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a certified ranked candidate")
}

func findBindingBySlot(t *testing.T, bindings []domain.SlotBinding, slot string) domain.SlotBinding {
	t.Helper()
	for _, b := range bindings {
		if b.Slot == slot {
			return b
		}
	}
	t.Fatalf("no binding found for slot %q", slot)
	return domain.SlotBinding{}
}
