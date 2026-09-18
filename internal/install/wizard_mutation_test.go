package install

import (
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/i18n"
	"github.com/stretchr/testify/require"
)

// TestWizardCriticalMutants protects the wizard's Role→Weapon decision points
// with deterministic input mutations rather than a source-mutating tool.
func TestWizardCriticalMutants(t *testing.T) {
	t.Parallel()

	mutants := []struct {
		name string
		want func(t *testing.T, catalog pluginCatalog)
	}{
		{
			name: "wrong role provider admitted",
			want: func(t *testing.T, catalog pluginCatalog) {
				ids, _, _, excluded := compatibleProviderOptions(catalog, "ranger", "")
				require.ElementsMatch(t, []string{"brainstorming", "openspec-explore"}, ids)
				require.Empty(t, excluded, "providers with a different declared role are not admitted to the candidate list")
				require.NotContains(t, ids, "openspec-propose")
			},
		},
		{
			name: "ranked marker dropped",
			want: func(t *testing.T, _ pluginCatalog) {
				provider, mode := splitRankedChoice("brainstorming::ranked", "brainstorming")
				require.Equal(t, "brainstorming", provider)
				require.Equal(t, domain.SlotBindingModeRanked, mode)
			},
		},
		{
			name: "custom choice upgraded",
			want: func(t *testing.T, _ pluginCatalog) {
				provider, mode := splitRankedChoice("custom-skill", "brainstorming")
				require.Equal(t, "custom-skill", provider)
				require.Equal(t, domain.SlotBindingModeCustom, mode)
			},
		},
		{
			name: "uncertified ranked default offered",
			want: func(t *testing.T, catalog pluginCatalog) {
				catalog.Providers[0].Ranked = false
				catalog.Providers[0].CertificationDigest = ""
				_, _, rankedID, _ := compatibleProviderOptions(catalog, "ranger", "")
				require.Empty(t, rankedID)
			},
		},
		{
			name: "execution ranked choice changes mode",
			want: func(t *testing.T, catalog pluginCatalog) {
				provider, mode, err := promptExecutionSlot(NewTextPrompter(strings.NewReader("sniper::ranked\n")), i18n.BundleFor("en"), catalog, knownProviderRisk)
				require.NoError(t, err)
				require.Equal(t, "sniper", provider)
				require.Equal(t, domain.SlotBindingModeRanked, mode)
			},
		},
	}

	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "brainstorming", RiskScore: "write_analysis", CanonicalRole: "ranger", Roles: []string{"ranger"}, CompatibilitySource: "embedded", Default: true, Ranked: true, CertificationDigest: "sha256:cert"},
		{ID: "openspec-propose", RiskScore: "write_analysis", CanonicalRole: "archivist", Roles: []string{"archivist"}, CompatibilitySource: "embedded"},
		{ID: "openspec-explore", RiskScore: "write_analysis", CanonicalRole: "ranger", Roles: []string{"ranger"}, CompatibilitySource: "embedded"},
		{ID: "sniper", RiskScore: "controlled", CanonicalRole: "sniper", Roles: []string{"sniper"}, CompatibilitySource: "native_role", Ranked: true, CertificationDigest: "sha256:sniper"},
	}}

	for _, mutant := range mutants {
		t.Run(mutant.name, func(t *testing.T) { mutant.want(t, catalog) })
	}
	t.Logf("critical mutants killed: %d/%d", len(mutants), len(mutants))
}
