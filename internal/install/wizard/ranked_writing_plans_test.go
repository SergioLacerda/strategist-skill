package wizard

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/install/rankedbinding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWizardOffersBothRankedArchivistsAndPreselectsOpenSpecPropose(t *testing.T) {
	t.Parallel()
	catalog := installtest.EmbeddedDefaultsCatalog(t)

	options := plugincatalog.CompatibleSlotOptions(catalog, plugincatalog.SlotRoleID(domain.SlotRefinement), plugincatalog.SlotHandoffSchema(domain.SlotRefinement))

	assert.ElementsMatch(t, []string{"openspec-propose", "writing-plans"}, options.RankedRefs, "writing-plans is selectable alongside openspec-propose")
	assert.Equal(t, "openspec-propose", options.RankedDefault, "openspec-propose stays the default Archivist")
}

func TestWizardLabelsRankedAndCustomWritingPlansDistinctly(t *testing.T) {
	t.Parallel()
	catalog := installtest.EmbeddedDefaultsCatalog(t)
	options := plugincatalog.CompatibleSlotOptions(catalog, plugincatalog.SlotRoleID(domain.SlotRefinement), plugincatalog.SlotHandoffSchema(domain.SlotRefinement))

	menu, _ := WithRankedLabeledOptions(options)

	labels := make(map[string]string, len(menu))
	for _, option := range menu {
		if option.Value == "writing-plans" || option.Value == "writing-plans::ranked" {
			labels[option.Value] = option.Label
		}
	}
	assert.Equal(t, "writing-plans (Ranked)", labels["writing-plans::ranked"])
	assert.Equal(t, "writing-plans (Custom)", labels["writing-plans"])
}

func TestApplyRankedBindingChoicesPersistsWritingPlansAsRankedWithTheCompiledIdentity(t *testing.T) {
	t.Parallel()
	catalog := installtest.EmbeddedDefaultsCatalog(t)
	// The shape the wizard migration used to persist: a custom binding with a
	// bare instance id and no Role/Weapon identity.
	lockFile := domain.PluginLockFile{Bindings: []domain.SlotBinding{
		{Slot: "refinement", InstalledInstanceID: "writing-plans", Mode: domain.SlotBindingModeCustom, Status: "enabled"},
	}}
	wc := domain.WizardConfig{RefinementProvider: "writing-plans", RefinementMode: domain.SlotBindingModeRanked}

	got, err := rankedbinding.ApplyRankedBindingChoices(catalog, wc, lockFile)

	require.NoError(t, err)
	binding := installtest.FindBindingBySlot(t, got.Bindings, "refinement")
	compiled, found, err := rankedbinding.SelectCompiledRankedBinding(catalog.RankedBindings, "archivist", "refinement", "writing-plans")
	require.NoError(t, err)
	require.True(t, found, "the embedded catalog certifies writing-plans for the Archivist")
	assert.Equal(t, domain.SlotBindingModeRanked, binding.Mode)
	assert.Equal(t, "writing-plans", binding.InstalledInstanceID)
	assert.Equal(t, "archivist", binding.Role)
	assert.Equal(t, domain.RankedRuntimeEmbedded, binding.RuntimeKind)
	assert.Equal(t, "refine", binding.Entrypoint)
	assert.Equal(t, compiled.BindingDigest, binding.BindingDigest)
	assert.Equal(t, compiled.WeaponDigest, binding.WeaponDigest)
	assert.Equal(t, compiled.SourceDigest, binding.SourceDigest)
	assert.Equal(t, compiled.CertificationDigest, binding.CertificationDigest)
}

func TestSilentInstallPromotesAnEmbeddedWritingPlansSelectionToRanked(t *testing.T) {
	t.Parallel()
	catalog := installtest.EmbeddedDefaultsCatalog(t)
	slots := map[string]string{"discovery": "brainstorming", "refinement": "writing-plans", "execution": "sniper"}
	lockFile := domain.PluginLockFile{Bindings: []domain.SlotBinding{
		{Slot: "refinement", InstalledInstanceID: "writing-plans", Mode: domain.SlotBindingModeCustom, Status: "enabled"},
	}}

	got, err := rankedbinding.PromoteRuntimeProvidersToRanked(catalog, slots, lockFile)

	require.NoError(t, err)
	assert.Equal(t, domain.SlotBindingModeRanked, installtest.FindBindingBySlot(t, got.Bindings, "refinement").Mode)
}

func TestRankedWritingPlansBindingResolvesThroughTheRoleWeaponBinding(t *testing.T) {
	t.Parallel()
	catalog := installtest.EmbeddedDefaultsCatalog(t)
	lockFile, err := rankedbinding.ApplyRankedBindingChoices(catalog,
		domain.WizardConfig{RefinementProvider: "writing-plans", RefinementMode: domain.SlotBindingModeRanked}, domain.PluginLockFile{})
	require.NoError(t, err)
	registry, err := domain.CompiledRegistryFromDocument(domain.CompiledRegistryDocument{
		SchemaVersion: catalog.SchemaVersion, TaxonomyVersion: catalog.TaxonomyVersion, Weapons: catalog.Weapons, Roles: catalog.Roles,
		Compatibility: catalog.Compatibility, RankedBindings: catalog.RankedBindings,
	})
	require.NoError(t, err)

	resolved, err := domain.ResolveRoleWeaponBinding(domain.ActiveConfig{Slots: map[string]string{"refinement": "writing-plans"}}, lockFile, registry, "archivist", "refinement")

	require.NoError(t, err, "a Ranked selection is never persisted as a binding the invoke path rejects")
	assert.Equal(t, domain.SlotBindingModeRanked, resolved.Mode)
	assert.Equal(t, domain.RankedRuntimeEmbedded, resolved.RuntimeKind)
	assert.Equal(t, "prompt_bridge", resolved.ExecutionMode)
}

func TestValidateRankedSelectionsAcceptsWritingPlansForRefinement(t *testing.T) {
	t.Parallel()
	options := plugincatalog.NewInstallOptionSet(installtest.EmbeddedDefaultsCatalog(t))

	err := options.ValidateRankedSelections(
		map[string]string{"refinement": "writing-plans"},
		map[string]string{"refinement": domain.SlotBindingModeRanked},
	)

	require.NoError(t, err)
}
