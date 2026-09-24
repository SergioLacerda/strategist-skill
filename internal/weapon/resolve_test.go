package weapon

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRuntime() domain.WeaponRuntime {
	return domain.WeaponRuntime{Kind: domain.WeaponRuntimeHost, HostAPI: "strategist-host-skill/v1"}
}

func testWeapon(id string) domain.WeaponManifest {
	return domain.WeaponManifest{ID: id, Version: "1.0.0", Kind: domain.WeaponKindAtomic, Origin: domain.WeaponOriginEmbedded, Roles: []string{"ranger"}, SupportedSlots: []string{"discovery"}, RiskScore: "write_analysis", Runtime: testRuntime()}
}

func TestResolveCompositeWeaponUsesRoleAndSlotAndStableOrder(t *testing.T) {
	parent := testWeapon("discovery-suite")
	parent.Kind = domain.WeaponKindComposite
	parent.Composition = &domain.WeaponComposition{Strategy: domain.WeaponCompositionOrderedDAG, Components: []domain.WeaponComponent{
		{ID: "openspec-explore", Required: false, DependsOn: []string{"brainstorming"}},
		{ID: "brainstorming", Required: true},
	}}
	registry := Registry{
		parent.ID:          parent,
		"brainstorming":    testWeapon("brainstorming"),
		"openspec-explore": testWeapon("openspec-explore"),
	}

	resolved, err := registry.Resolve("discovery-suite", "ranger", "discovery")
	require.NoError(t, err)
	require.Len(t, resolved.Components, 2)
	assert.Equal(t, "brainstorming", resolved.Components[0].Manifest.ID)
	assert.Equal(t, "openspec-explore", resolved.Components[1].Manifest.ID)
	assert.True(t, resolved.Components[0].Required)
	assert.False(t, resolved.Components[1].Required)
}

func TestResolveCompositeWeaponFailsForMissingOrIncompatibleComponent(t *testing.T) {
	parent := testWeapon("suite")
	parent.Kind = domain.WeaponKindComposite
	parent.Composition = &domain.WeaponComposition{Strategy: domain.WeaponCompositionOrderedDAG, Components: []domain.WeaponComponent{{ID: "missing", Required: false}}}
	_, err := (Registry{"suite": parent}).Resolve("suite", "ranger", "discovery")
	require.ErrorContains(t, err, "missing component Weapon")

	component := testWeapon("other-slot")
	component.Roles = []string{"archivist"}
	parent.Composition.Components[0].ID = component.ID
	_, err = (Registry{"suite": parent, component.ID: component}).Resolve("suite", "ranger", "discovery")
	assert.ErrorContains(t, err, "incompatible with Role")
}

func TestResolveRejectsLegacyCanonicalRoleOnlyManifest(t *testing.T) {
	legacy := testWeapon("legacy")
	legacy.Roles = nil
	_, err := (Registry{"legacy": legacy}).Resolve("legacy", "ranger", "discovery")
	assert.ErrorContains(t, err, "roles must not be empty")
}
