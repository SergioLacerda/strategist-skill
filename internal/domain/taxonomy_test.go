package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanonicalIdentityIsFamilyAware(t *testing.T) {
	t.Parallel()

	for _, family := range []domain.TaxonomyFamily{
		domain.TaxonomyRole,
		domain.TaxonomyWeapon,
		domain.TaxonomyFeat,
		domain.TaxonomyTool,
		domain.TaxonomyMechanism,
		domain.TaxonomyStage,
		domain.TaxonomyArtifact,
	} {
		identity := domain.CanonicalIdentity{Family: family, ID: "example"}
		if family == domain.TaxonomyWeapon {
			identity.Version = "1.0.0"
		}
		require.NoError(t, identity.Validate(), family)
	}

	require.Error(t, (domain.CanonicalIdentity{Family: "ability", ID: "initiative"}).Validate())
	require.Error(t, (domain.CanonicalIdentity{Family: domain.TaxonomyWeapon, ID: "brainstorming"}).Validate())

	roleIdentity := (domain.Role{ID: "ranger"}).CanonicalIdentity()
	assert.Equal(t, domain.TaxonomyRole, roleIdentity.Family)
	assert.Equal(t, "ranger", roleIdentity.ID)

	weaponIdentity := (domain.WeaponManifest{ID: "brainstorming", Version: "1.0.0"}).CanonicalIdentity()
	assert.Equal(t, domain.TaxonomyWeapon, weaponIdentity.Family)
	assert.Equal(t, "1.0.0", weaponIdentity.Version)
}

func TestResolveStagePreservesLegacyCorrelation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		route     string
		stage     domain.Stage
		feat      string
		execution bool
	}{
		{name: "full", route: domain.MissionRouteFullPipeline, stage: domain.StageFull},
		{name: "short", route: "implementation_short_route", stage: domain.StageShort},
		{name: "critical hit", route: "critical_hit", stage: domain.StageShort, feat: "critical_hit"},
		{name: "roster", route: "roster", stage: domain.StageRoster},
		{name: "direct execute", route: domain.MissionRouteDirectExecute, stage: domain.StageShort, execution: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolution, err := domain.ResolveStage(domain.StageResolutionRequest{
				Route:            tt.route,
				Role:             "sniper",
				MissionExecution: tt.execution,
			})
			require.NoError(t, err)
			require.NoError(t, resolution.Validate())
			assert.Equal(t, tt.stage, resolution.Stage)
			assert.Equal(t, tt.route, resolution.LegacyRoute)
			if tt.feat != "" {
				assert.Equal(t, tt.feat, resolution.Feat)
			}
		})
	}
}

func TestResolveStageFailsClosed(t *testing.T) {
	t.Parallel()

	_, err := domain.ResolveStage(domain.StageResolutionRequest{Route: "unknown"})
	require.Error(t, err)

	_, err = domain.ResolveStage(domain.StageResolutionRequest{Route: domain.MissionRouteDirectExecute})
	require.Error(t, err)
}

func TestValidateTaxonomyVersionAllowsLegacyAndCanonicalValues(t *testing.T) {
	t.Parallel()

	assert.NoError(t, domain.ValidateTaxonomyVersion(""))
	assert.NoError(t, domain.ValidateTaxonomyVersion(domain.CanonicalTaxonomyVersion))
}

func TestValidateTaxonomyVersionRejectsUnknownValues(t *testing.T) {
	t.Parallel()

	err := domain.ValidateTaxonomyVersion("strategist-taxonomy/v0")
	assert.ErrorContains(t, err, "unsupported taxonomy version")
}
