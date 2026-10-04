package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoleLoadoutIsStageAwareAndDeterministic(t *testing.T) {
	t.Parallel()
	resolution, err := domain.ResolveStage(domain.StageResolutionRequest{Route: "full_pipeline", Role: "ranger"})
	require.NoError(t, err)
	loadout, err := domain.NewRoleLoadout(resolution, loadoutWeapon(),
		[]domain.LoadoutCapability{{Family: domain.TaxonomyFeat, ID: "initiative", Availability: domain.CapabilityAvailable}},
		[]domain.LoadoutCapability{{Family: domain.TaxonomyTool, ID: "leveling", Availability: domain.CapabilityAvailable}},
	)
	require.NoError(t, err)
	require.NoError(t, loadout.Validate())
	assert.Equal(t, domain.StageFull, loadout.Resolution.Stage)
	assert.Equal(t, "initiative", loadout.Feats[0].ID)
}

func TestRoleLoadoutRejectsUnavailableCapabilityAndRuntime(t *testing.T) {
	t.Parallel()
	resolution, err := domain.ResolveStage(domain.StageResolutionRequest{Route: "implementation_short_route", Role: "ranger"})
	require.NoError(t, err)
	_, err = domain.NewRoleLoadout(resolution, loadoutWeapon(), []domain.LoadoutCapability{{
		Family: domain.TaxonomyFeat, ID: "critical_hit", Availability: domain.CapabilityStaleDigest,
	}}, nil)
	require.ErrorContains(t, err, "stale_digest")

	weapon := loadoutWeapon()
	weapon.Runtime.Kind = domain.RankedRuntimeNone
	_, err = domain.NewRoleLoadout(resolution, weapon, nil, nil)
	require.ErrorContains(t, err, "not invocable")
	weapon.Runtime.Kind = "future-runtime"
	_, err = domain.NewRoleLoadout(resolution, weapon, nil, nil)
	require.ErrorContains(t, err, "unsupported")
}

func loadoutWeapon() domain.RoleInvocationPlan {
	return domain.RoleInvocationPlan{
		Role: "ranger", Slot: "discovery", WeaponID: "brainstorming", WeaponVersion: "1.0.0",
		WeaponDigest: "sha256:weapon", BindingDigest: "sha256:binding",
		Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded},
	}
}
