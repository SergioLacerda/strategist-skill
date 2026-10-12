package onboarding

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install/embeddedweapon"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
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
	require.NoError(t, ApplyPluginBinding(store, desiredNew, func(domain.SlotBinding, domain.InstalledInstance) bool { return true }))
	binding, _ := store.Binding("discovery")
	assert.Equal(t, "new", binding.InstalledInstanceID)
	assert.Equal(t, int64(2), binding.Generation)

	same := switchStore()
	require.NoError(t, ApplyPluginBinding(same, domain.SlotBinding{Slot: "discovery", InstalledInstanceID: "old"}, nil), "an unchanged instance is a no-op")
}

func TestSwitchPluginBindingFailsWhenTheProbeFails(t *testing.T) {
	t.Parallel()
	err := ApplyPluginBinding(switchStore(), desiredNew, func(domain.SlotBinding, domain.InstalledInstance) bool { return false })
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
	require.NoError(t, ApplyPluginBindingWithResult(fresh, desiredNew, pass))
	require.Len(t, fresh.Bindings, 1)

	store := switchStore()
	require.NoError(t, ApplyPluginBindingWithResult(store, desiredNew, pass))
	binding, _ := store.Binding("discovery")
	assert.Equal(t, "new", binding.InstalledInstanceID)
	require.NoError(t, ApplyPluginBindingWithResult(store, desiredNew, pass), "already switched")

	require.Error(t, ApplyPluginBindingWithResult(switchStore(), desiredNew, fail))
	missing := switchStore()
	missing.Inventory.Instances = missing.Inventory.Instances[:1]
	require.ErrorContains(t, ApplyPluginBindingWithResult(missing, desiredNew, pass), "planned_instance_missing")
}

func TestBeginPluginBindingErrors(t *testing.T) {
	t.Parallel()
	store := lifecycle.NewStore()
	store.Inventory.Instances = []domain.InstalledInstance{{ID: "new"}}
	_, err := BeginPluginBinding(store, desiredNew)
	require.ErrorContains(t, err, "begin plugin lifecycle transaction")
}

func TestCatalogWeaponCompositionValidation(t *testing.T) {
	t.Parallel()
	child := plugincatalog.Provider{ID: "child", Roles: []string{"ranger"}, SupportedSlots: []string{"discovery"}, CompatibilitySource: "external"}
	parent := func(role, slot string, comps ...string) plugincatalog.Provider {
		composition := &domain.WeaponComposition{Strategy: domain.WeaponCompositionOrderedDAG}
		for _, id := range comps {
			composition.Components = append(composition.Components, domain.WeaponComponent{ID: id, Required: true})
		}
		return plugincatalog.Provider{ID: "parent", Kind: domain.WeaponKindComposite, Roles: []string{role}, SupportedSlots: []string{slot}, Composition: composition}
	}
	validate := func(p plugincatalog.Provider) error {
		return plugincatalog.ValidateCatalogWeaponCompositions(plugincatalog.PluginCatalog{Providers: []plugincatalog.Provider{child, p}})
	}

	require.NoError(t, validate(parent("ranger", "discovery", "child")))
	require.ErrorContains(t, validate(parent("ranger", "discovery", "ghost")), "missing component Weapon ghost")
	require.ErrorContains(t, validate(parent("archivist", "discovery", "child")), "incompatible with Role archivist")
	require.ErrorContains(t, validate(parent("ranger", "execution", "child")), "incompatible with slot execution")

	embedded := child
	embedded.CompatibilitySource = "embedded"
	err := plugincatalog.ValidateCatalogWeaponCompositions(plugincatalog.PluginCatalog{Providers: []plugincatalog.Provider{embedded, parent("ranger", "discovery", "child")}})
	require.ErrorContains(t, err, "is not invocable")
	assert.True(t, plugincatalog.ContainsCatalogString([]string{"a", "b"}, "b"))
	assert.False(t, plugincatalog.ContainsCatalogString(nil, "a"))
}

func TestValidateCatalogProviderRejections(t *testing.T) {
	t.Parallel()
	require.ErrorContains(t, plugincatalog.ValidateCatalogProvider(plugincatalog.Provider{}), "provider id and risk_score are required")
	require.Error(t, plugincatalog.ValidateCatalogProvider(plugincatalog.Provider{ID: "x", RiskScore: "low", CanonicalRole: "pathfinder"}))
	require.Error(t, plugincatalog.ValidateCatalogProvider(plugincatalog.Provider{ID: "x", RiskScore: "low", CanonicalRole: "ranger", Roles: []string{"ranger"}, WeaponContract: domain.WeaponContract{Participation: "bogus"}}), "an invalid weapon contract is rejected")

	manifest := plugincatalog.CatalogProviderManifest(plugincatalog.Provider{ID: "x", CanonicalRole: "ranger"})
	assert.Equal(t, "legacy", manifest.Version)
	assert.Equal(t, []string{"ranger"}, manifest.Roles)
	require.NoError(t, plugincatalog.ValidateEmbeddedCatalogProvider(plugincatalog.Provider{ID: "x"}), "a legacy catalog entry without kind is accepted")
	require.ErrorContains(t, plugincatalog.ValidateEmbeddedCatalogProvider(plugincatalog.Provider{ID: "x", Kind: domain.WeaponKindAtomic}), "must declare supported_slots")
	require.ErrorContains(t, plugincatalog.ValidateEmbeddedCatalogProvider(plugincatalog.Provider{ID: "x", Kind: domain.WeaponKindAtomic, SupportedSlots: []string{"discovery"}}), "Weapon x")
}

func TestAddCompositionViolationsMarksCompositeCandidates(t *testing.T) {
	t.Parallel()
	violations := map[string][]string{}
	embeddedweapon.AddCompositionViolations(violations, []plugincatalog.IngestedSkill{
		{ID: "atomic", Adapter: plugincatalog.ExternalSkillAdapter{Kind: domain.WeaponKindAtomic}},
		{ID: "comp", Adapter: plugincatalog.ExternalSkillAdapter{Kind: "composite"}},
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

	err := SwitchPluginBinding(store, current, desiredNew, instance, func(domain.SlotBinding, domain.InstalledInstance) bool { return true })
	require.ErrorContains(t, err, "begin plugin lifecycle transaction")
	err = SwitchPluginBindingWithResult(store, current, desiredNew, instance, func(domain.SlotBinding, domain.InstalledInstance) lifecycle.ProbeOutcome {
		return lifecycle.ProbeOutcome{}
	})
	require.ErrorContains(t, err, "begin plugin lifecycle transaction")
}
