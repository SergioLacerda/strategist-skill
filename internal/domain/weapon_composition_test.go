package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestWeaponCompositionTopologicalOrderIsDeterministic(t *testing.T) {
	t.Parallel()

	composition := domain.WeaponComposition{
		Strategy: domain.WeaponCompositionOrderedDAG,
		Components: []domain.WeaponComponent{
			{ID: "zeta"},
			{ID: "alpha"},
			{ID: "omega", DependsOn: []string{"alpha", "zeta"}},
		},
	}

	require.NoError(t, composition.Validate())
	order, err := composition.TopologicalOrder()
	require.NoError(t, err)
	require.Equal(t, []string{"alpha", "zeta", "omega"}, order)
}

func TestWeaponCompositionRejectsDuplicateMissingAndCyclicDependencies(t *testing.T) {
	t.Parallel()

	duplicate := domain.WeaponComposition{Strategy: domain.WeaponCompositionOrderedDAG, Components: []domain.WeaponComponent{{ID: "a"}, {ID: "a"}}}
	require.ErrorContains(t, duplicate.Validate(), "duplicate")

	missing := domain.WeaponComposition{Strategy: domain.WeaponCompositionOrderedDAG, Components: []domain.WeaponComponent{{ID: "a", DependsOn: []string{"missing"}}}}
	require.ErrorContains(t, missing.Validate(), "unknown component")

	cycle := domain.WeaponComposition{Strategy: domain.WeaponCompositionOrderedDAG, Components: []domain.WeaponComponent{
		{ID: "a", DependsOn: []string{"b"}},
		{ID: "b", DependsOn: []string{"a"}},
	}}
	require.ErrorContains(t, cycle.Validate(), "cycle")
}

func TestWeaponCompositionPreservesRequiredAndOptionalComponents(t *testing.T) {
	t.Parallel()

	composition := domain.WeaponComposition{Strategy: domain.WeaponCompositionOrderedDAG, Components: []domain.WeaponComponent{
		{ID: "required", Required: true},
		{ID: "optional", Required: false},
	}}

	require.NoError(t, composition.Validate())
	require.True(t, composition.Components[0].Required)
	require.False(t, composition.Components[1].Required)
}
