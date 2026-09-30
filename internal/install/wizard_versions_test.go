package install

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func versionedRangerCatalog() pluginCatalog {
	weapon := func(version string, certified, isDefault bool) pluginCatalogProvider {
		provider := pluginCatalogProvider{
			ID: "brainstorming", Version: version, RiskScore: "write_analysis", CanonicalRole: "ranger",
			CompatibilitySource: "embedded", Default: isDefault,
		}
		if certified {
			provider.Ranked, provider.CertificationDigest = true, "sha256:cert-"+version
		}
		return provider
	}
	return pluginCatalog{Providers: []pluginCatalogProvider{
		weapon("2.0.0", true, false),
		weapon("1.4.0", true, true),
		weapon("2.1.0-rc1", false, false),
		{ID: "solo", Version: "1.0.0", RiskScore: "write_analysis", CanonicalRole: "ranger", CompatibilitySource: "embedded"},
	}}
}

func TestCompatibleSlotOptionsListsIDAtVersionOnlyWhereSeveralVersionsExist(t *testing.T) {
	t.Parallel()

	options := compatibleSlotOptions(versionedRangerCatalog(), "ranger", "")

	assert.Equal(t, []string{"brainstorming@1.4.0", "brainstorming@2.0.0", "brainstorming@2.1.0-rc1", "solo"}, options.ids,
		"a version is spelled out exactly where a plain id would be ambiguous")
	assert.Equal(t, "brainstorming@1.4.0", options.defaultID, "the catalog's default flag pre-selects, no version is guessed")
}

func TestCompatibleSlotOptionsOffersOnlyCertifiedVersionsAsRanked(t *testing.T) {
	t.Parallel()

	options := compatibleSlotOptions(versionedRangerCatalog(), "ranger", "")

	assert.Equal(t, []string{"brainstorming@1.4.0", "brainstorming@2.0.0"}, options.rankedRefs, "the uncertified 2.1.0-rc1 is Custom only")
	assert.Equal(t, "brainstorming@1.4.0", options.rankedDefault)
}

func TestCompatibleSlotOptionsPreselectsTheHighestCertifiedVersionWhenNoneIsDefault(t *testing.T) {
	t.Parallel()
	catalog := versionedRangerCatalog()
	for i := range catalog.Providers {
		catalog.Providers[i].Default = false
	}

	options := compatibleSlotOptions(catalog, "ranger", "")

	assert.Equal(t, "brainstorming@2.0.0", options.rankedDefault, "only a UI pre-selection: the operator still confirms it")
}

func TestWithRankedOptionsOffersOneRankedEntryPerCertifiedVersion(t *testing.T) {
	t.Parallel()
	options := compatibleSlotOptions(versionedRangerCatalog(), "ranger", "")

	menu, uiDefault := withRankedOptions(options)

	assert.Equal(t, []string{
		"brainstorming@1.4.0::ranked", "brainstorming@2.0.0::ranked",
		"brainstorming@1.4.0", "brainstorming@2.0.0", "brainstorming@2.1.0-rc1", "solo",
	}, menu)
	assert.Equal(t, "brainstorming@1.4.0::ranked", uiDefault)
}

func TestSplitRankedChoiceAmongResolvesOnlyOfferedCertifiedRefs(t *testing.T) {
	t.Parallel()
	refs := []string{"brainstorming@1.4.0", "brainstorming@2.0.0"}

	provider, mode := splitRankedChoiceAmong("brainstorming@2.0.0::ranked", refs)
	assert.Equal(t, "brainstorming@2.0.0", provider)
	assert.Equal(t, domain.SlotBindingModeRanked, mode)

	provider, mode = splitRankedChoiceAmong("brainstorming@2.1.0-rc1", refs)
	assert.Equal(t, "brainstorming@2.1.0-rc1", provider)
	assert.Equal(t, domain.SlotBindingModeCustom, mode, "an uncertified version is Custom")

	provider, mode = splitRankedChoiceAmong("brainstorming@9.9.9::ranked", refs)
	assert.Equal(t, "brainstorming@9.9.9::ranked", provider, "a Ranked suffix on a ref that was not offered is not honored")
	assert.Equal(t, domain.SlotBindingModeCustom, mode)
}

func TestWizardPromptWritesTheChosenVersionedRef(t *testing.T) {
	t.Parallel()
	options := compatibleSlotOptions(versionedRangerCatalog(), "ranger", "")
	prompter := &scriptedSelector{answer: "brainstorming@2.0.0::ranked"}

	provider, mode, err := promptSlotOptions(prompter, "discovery?", options, "custom", map[string]string{}, "write_analysis", "discovery")

	require.NoError(t, err)
	assert.Equal(t, "brainstorming@2.0.0", provider)
	assert.Equal(t, domain.SlotBindingModeRanked, mode)
	assert.Equal(t, "brainstorming@1.4.0::ranked", prompter.gotDefault)
}

// scriptedSelector answers SelectOrInput with a fixed choice and records the
// pre-selected default it was shown.
type scriptedSelector struct {
	answer     string
	gotDefault string
	gotOptions []string
}

func (s *scriptedSelector) Select(_, defaultVal string, options []string) (string, error) {
	s.gotDefault, s.gotOptions = defaultVal, options
	return s.answer, nil
}

func (s *scriptedSelector) Input(_, defaultVal string) (string, error) { return defaultVal, nil }

func (s *scriptedSelector) SelectOrInput(_, defaultVal string, options []string, _ string) (string, error) {
	s.gotDefault, s.gotOptions = defaultVal, options
	return s.answer, nil
}

func TestOnboardingPinsTheChosenVersionInsteadOfTheHighestOne(t *testing.T) {
	t.Parallel()
	catalog := versionedRangerCatalog()

	requirements, err := onboardingRequirements(catalog, map[string]string{"discovery": "brainstorming@1.4.0"})
	require.NoError(t, err)
	require.Len(t, requirements, 1)
	assert.Equal(t, "1.4.0", requirements[0].Constraint, "an older chosen version is pinned exactly: no 'latest' selection")

	lock, err := plugins.Resolve(requirements, catalogResolverCandidates(catalog))
	require.NoError(t, err)
	bindings, err := bindingsFromSlots(map[string]string{"discovery": "brainstorming@1.4.0"}, lock)
	require.NoError(t, err)
	require.Len(t, bindings, 1)
	assert.Equal(t, "brainstorming", lock.Nodes[0].ID)
	assert.Equal(t, catalogProviderDigest(catalog.Providers[1]), lock.Nodes[0].Digest, "the lock node is the 1.4.0 payload, not 2.0.0")
}

func TestOnboardingRefusesAPlainIDThatHasSeveralVersions(t *testing.T) {
	t.Parallel()

	_, err := onboardingRequirements(versionedRangerCatalog(), map[string]string{"discovery": "brainstorming"})

	require.ErrorContains(t, err, "unresolved_active_slot")
	require.ErrorContains(t, err, "brainstorming@1.4.0")
	require.ErrorContains(t, err, "select one as id@version")
}

func TestApplyRankedBindingChoicesPersistsTheChosenCertifiedVersion(t *testing.T) {
	t.Parallel()
	catalog := rankedVersionsCatalog("1.4.0", "2.0.0")
	wc := domain.WizardConfig{DiscoveryProvider: "demo@1.4.0", DiscoveryMode: domain.SlotBindingModeRanked}

	lock, err := applyRankedBindingChoices(catalog, wc, domain.PluginLockFile{})

	require.NoError(t, err)
	require.Len(t, lock.Bindings, 1)
	binding := lock.Bindings[0]
	assert.Equal(t, "demo", binding.InstalledInstanceID)
	assert.Equal(t, "1.4.0", binding.WeaponVersion)
	assert.Equal(t, "sha256:b-1.4.0", binding.BindingDigest)
	assert.True(t, domain.WeaponRefMatchesBinding(wc.DiscoveryProvider, binding), "the lock agrees with the active.yaml reference")

	wc.DiscoveryProvider = "demo"
	_, err = applyRankedBindingChoices(catalog, wc, domain.PluginLockFile{})
	require.ErrorContains(t, err, "several certified versions")
}
