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

	ids, defaultID := compatibleProviderOptions(catalog, "archivist", "schemas/handoff-archivist-to-sniper.schema.yaml", "archivist")
	assert.ElementsMatch(t, []string{"archivist", "openspec-propose"}, ids)
	assert.Equal(t, "openspec-propose", defaultID)
}

func TestCompatibleProviderOptionsExcludesIncompatibleCandidate(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "archivist", RiskScore: "write_analysis", CompatibilitySource: "native_role"},
		{
			ID: "openspec-propose", RiskScore: "write_analysis", CanonicalRole: "archivist",
			CompatibilitySource: "embedded", Default: true,
			// no SupportedHandoffSchemas declared — today's real state
		},
	}}

	ids, defaultID := compatibleProviderOptions(catalog, "archivist", "schemas/handoff-archivist-to-sniper.schema.yaml", "archivist")
	assert.Equal(t, []string{"archivist"}, ids)
	assert.Equal(t, "archivist", defaultID)
}

func TestCompatibleProviderOptionsFallsBackWhenCatalogHasNoNativeEntry(t *testing.T) {
	t.Parallel()

	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "openspec-propose", RiskScore: "write_analysis", CanonicalRole: "archivist", CompatibilitySource: "embedded"},
	}}

	ids, defaultID := compatibleProviderOptions(catalog, "archivist", "schemas/handoff-archivist-to-sniper.schema.yaml", "archivist")
	assert.Equal(t, []string{"archivist"}, ids, "falls back to the passed-in fallbackID even with no catalog entry for it, keeping the wizard usable")
	assert.Equal(t, "archivist", defaultID)
}
