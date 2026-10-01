package install

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/plugins/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func switchStore() *lifecycle.Store {
	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{
		{ID: "old", State: "active"}, {ID: "new", State: "installed"},
	}
	store.Bindings = []domain.SlotBinding{{Slot: "discovery", InstalledInstanceID: "old", Generation: 1, Status: "enabled"}}
	return store
}

var desiredNew = domain.SlotBinding{Slot: "discovery", InstalledInstanceID: "new", Status: "enabled"}

func TestSwitchPluginBindingActivatesOnAPassingProbe(t *testing.T) {
	t.Parallel()
	store := switchStore()
	require.NoError(t, applyPluginBinding(store, desiredNew, func(domain.SlotBinding, domain.InstalledInstance) bool { return true }))
	binding, _ := store.Binding("discovery")
	assert.Equal(t, "new", binding.InstalledInstanceID)
	assert.Equal(t, int64(2), binding.Generation)

	same := switchStore()
	require.NoError(t, applyPluginBinding(same, domain.SlotBinding{Slot: "discovery", InstalledInstanceID: "old"}, nil), "an unchanged instance is a no-op")
}

func TestSwitchPluginBindingFailsWhenTheProbeFails(t *testing.T) {
	t.Parallel()
	err := applyPluginBinding(switchStore(), desiredNew, func(domain.SlotBinding, domain.InstalledInstance) bool { return false })
	require.ErrorContains(t, err, "activate plugin lifecycle transaction")
}

func TestApplyPluginBindingWithResult(t *testing.T) {
	t.Parallel()
	pass := func(domain.SlotBinding, domain.InstalledInstance) lifecycle.ProbeOutcome {
		return lifecycle.ProbeOutcome{Status: domain.ReadinessReady}
	}
	fail := func(domain.SlotBinding, domain.InstalledInstance) lifecycle.ProbeOutcome {
		return lifecycle.ProbeOutcome{Status: domain.ReadinessBlocked, ReasonCode: "probe_failed"}
	}

	fresh := lifecycle.NewStore()
	require.NoError(t, applyPluginBindingWithResult(fresh, desiredNew, pass))
	require.Len(t, fresh.Bindings, 1)

	store := switchStore()
	require.NoError(t, applyPluginBindingWithResult(store, desiredNew, pass))
	binding, _ := store.Binding("discovery")
	assert.Equal(t, "new", binding.InstalledInstanceID)
	require.NoError(t, applyPluginBindingWithResult(store, desiredNew, pass), "already switched")

	require.Error(t, applyPluginBindingWithResult(switchStore(), desiredNew, fail))
	missing := switchStore()
	missing.Inventory.Instances = missing.Inventory.Instances[:1]
	require.ErrorContains(t, applyPluginBindingWithResult(missing, desiredNew, pass), "planned_instance_missing")
}

func TestBeginPluginBindingErrors(t *testing.T) {
	t.Parallel()
	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: "new"}}
	_, err := beginPluginBinding(store, desiredNew)
	require.ErrorContains(t, err, "begin plugin lifecycle transaction")
}

func TestCatalogWeaponCompositionValidation(t *testing.T) {
	t.Parallel()
	child := pluginCatalogProvider{ID: "child", Roles: []string{"ranger"}, SupportedSlots: []string{"discovery"}, CompatibilitySource: "external"}
	parent := func(role, slot string, comps ...string) pluginCatalogProvider {
		composition := &domain.WeaponComposition{Strategy: domain.WeaponCompositionOrderedDAG}
		for _, id := range comps {
			composition.Components = append(composition.Components, domain.WeaponComponent{ID: id, Required: true})
		}
		return pluginCatalogProvider{ID: "parent", Kind: domain.WeaponKindComposite, Roles: []string{role}, SupportedSlots: []string{slot}, Composition: composition}
	}
	validate := func(p pluginCatalogProvider) error {
		return validateCatalogWeaponCompositions(pluginCatalog{Providers: []pluginCatalogProvider{child, p}})
	}

	require.NoError(t, validate(parent("ranger", "discovery", "child")))
	require.ErrorContains(t, validate(parent("ranger", "discovery", "ghost")), "missing component Weapon ghost")
	require.ErrorContains(t, validate(parent("archivist", "discovery", "child")), "incompatible with Role archivist")
	require.ErrorContains(t, validate(parent("ranger", "execution", "child")), "incompatible with slot execution")

	embedded := child
	embedded.CompatibilitySource = "embedded"
	err := validateCatalogWeaponCompositions(pluginCatalog{Providers: []pluginCatalogProvider{embedded, parent("ranger", "discovery", "child")}})
	require.ErrorContains(t, err, "is not invocable")
	assert.True(t, containsCatalogString([]string{"a", "b"}, "b"))
	assert.False(t, containsCatalogString(nil, "a"))
}

func TestValidateCatalogProviderRejections(t *testing.T) {
	t.Parallel()
	require.ErrorContains(t, validateCatalogProvider(pluginCatalogProvider{}), "provider id and risk_score are required")
	require.Error(t, validateCatalogProvider(pluginCatalogProvider{ID: "x", RiskScore: "low", CanonicalRole: "pathfinder"}))
	require.Error(t, validateCatalogProvider(pluginCatalogProvider{ID: "x", RiskScore: "low", CanonicalRole: "ranger", Roles: []string{"ranger"}, WeaponContract: domain.WeaponContract{Participation: "bogus"}}), "an invalid weapon contract is rejected")

	manifest := catalogProviderManifest(pluginCatalogProvider{ID: "x", CanonicalRole: "ranger"})
	assert.Equal(t, "legacy", manifest.Version)
	assert.Equal(t, []string{"ranger"}, manifest.Roles)
	require.NoError(t, validateEmbeddedCatalogProvider(pluginCatalogProvider{ID: "x"}), "a legacy catalog entry without kind is accepted")
	require.ErrorContains(t, validateEmbeddedCatalogProvider(pluginCatalogProvider{ID: "x", Kind: domain.WeaponKindAtomic}), "must declare supported_slots")
	require.ErrorContains(t, validateEmbeddedCatalogProvider(pluginCatalogProvider{ID: "x", Kind: domain.WeaponKindAtomic, SupportedSlots: []string{"discovery"}}), "Weapon x")
}

func TestAddCompositionViolationsMarksCompositeCandidates(t *testing.T) {
	t.Parallel()
	violations := map[string][]string{}
	addCompositionViolations(violations, []IngestedSkill{
		{ID: "atomic", Adapter: externalSkillAdapter{Kind: domain.WeaponKindAtomic}},
		{ID: "comp", Adapter: externalSkillAdapter{Kind: "composite"}},
	}, assert.AnError)
	assert.Contains(t, violations, "comp")
	assert.NotContains(t, violations, "atomic")
}

func TestSwitchPluginBindingBeginFailsForAnUnknownInstance(t *testing.T) {
	t.Parallel()
	store := switchStore()
	store.Inventory.Instances = store.Inventory.Instances[:1]
	current, _ := store.Binding("discovery")
	instance, _ := store.Instance("old")

	err := switchPluginBinding(store, current, desiredNew, instance, func(domain.SlotBinding, domain.InstalledInstance) bool { return true })
	require.ErrorContains(t, err, "begin plugin lifecycle transaction")
	err = switchPluginBindingWithResult(store, current, desiredNew, instance, func(domain.SlotBinding, domain.InstalledInstance) lifecycle.ProbeOutcome {
		return lifecycle.ProbeOutcome{}
	})
	require.ErrorContains(t, err, "begin plugin lifecycle transaction")
}
