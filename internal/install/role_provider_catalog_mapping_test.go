package install

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestProviderContractFromCatalogEntryCarriesSupportedHandoffSchemas(t *testing.T) {
	t.Parallel()

	entry := pluginCatalogProvider{
		ID:                      "openspec-propose",
		RiskScore:               "write_analysis",
		CanonicalRole:           "archivist",
		CompatibilitySource:     "embedded",
		SupportedHandoffSchemas: []string{"openspec-native-change-schema"},
	}

	contract := providerContractFromCatalogEntry(entry)
	assert.Equal(t, []string{"openspec-native-change-schema"}, contract.SupportedHandoffSchemas)
}

func TestProviderContractFromCatalogEntryLeavesSupportedHandoffSchemasNilWhenUndeclared(t *testing.T) {
	t.Parallel()

	entry := pluginCatalogProvider{ID: "brainstorming", RiskScore: "write_analysis", CanonicalRole: "ranger", CompatibilitySource: "embedded"}

	contract := providerContractFromCatalogEntry(entry)
	assert.Empty(t, contract.SupportedHandoffSchemas)
}

func TestProviderContractFromCatalogEntryNativeRoleNeedsNoDeclaredSchema(t *testing.T) {
	t.Parallel()

	entry := pluginCatalogProvider{ID: "archivist", RiskScore: "write_analysis", CompatibilitySource: "native_role"}

	contract := providerContractFromCatalogEntry(entry)
	assert.Equal(t, domain.ProviderSourceNativeRole, contract.Source)
	assert.Empty(t, contract.SupportedHandoffSchemas, "native role providers skip the handoff_schema dimension entirely; they need no declared value")
}

func TestProviderSourceFromCompatibilitySource(t *testing.T) {
	t.Parallel()

	tests := []struct {
		compatibilitySource string
		want                domain.ProviderSource
	}{
		{"embedded", domain.ProviderSourceEmbedded},
		{"native_role", domain.ProviderSourceNativeRole},
		{"external", domain.ProviderSourceExternal},
		{"", domain.ProviderSourceExternal},
		{"unrecognized_value", domain.ProviderSourceExternal},
	}
	for _, tt := range tests {
		t.Run(tt.compatibilitySource, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, providerSourceFromCompatibilitySource(tt.compatibilitySource))
		})
	}
}
