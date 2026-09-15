package install

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompatibleProviderOptionsPrefersDefaultCompatibleCandidate(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "archivist", RiskScore: "write_analysis", CompatibilitySource: "native_role"},
		{
			ID: "openspec-propose", RiskScore: "write_analysis", CanonicalRole: "archivist",
			CompatibilitySource: "embedded", Default: true,
			SupportedHandoffSchemas: []string{"schemas/handoff-archivist-to-sniper.schema.yaml"},
		},
	}}

	ids, defaultID, excluded := compatibleProviderOptions(catalog, "archivist", "schemas/handoff-archivist-to-sniper.schema.yaml", "archivist")
	assert.ElementsMatch(t, []string{"archivist", "openspec-propose"}, ids)
	assert.Equal(t, "openspec-propose", defaultID)
	assert.Empty(t, excluded)
}

func TestCompatibleProviderOptionsIncludesAffiliatedCandidateWithoutNativeHandoff(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "archivist", RiskScore: "write_analysis", CompatibilitySource: "native_role"},
		{
			ID: "openspec-propose", RiskScore: "write_analysis", CanonicalRole: "archivist",
			CompatibilitySource: "embedded", Default: true,
			// The fixed Archivist role owns normalization into its handoff.
		},
	}}

	ids, defaultID, excluded := compatibleProviderOptions(catalog, "archivist", "schemas/handoff-archivist-to-sniper.schema.yaml", "archivist")
	assert.ElementsMatch(t, []string{"archivist", "openspec-propose"}, ids)
	assert.Equal(t, "openspec-propose", defaultID)
	assert.Empty(t, excluded)
}

func TestCompatibleProviderOptionsFallsBackWhenCatalogHasNoNativeEntry(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "openspec-propose", RiskScore: "write_analysis", CanonicalRole: "archivist", CompatibilitySource: "embedded"},
	}}

	ids, defaultID, excluded := compatibleProviderOptions(catalog, "archivist", "schemas/handoff-archivist-to-sniper.schema.yaml", "archivist")
	assert.Equal(t, []string{"openspec-propose"}, ids)
	assert.Equal(t, "openspec-propose", defaultID)
	assert.Empty(t, excluded)
}
