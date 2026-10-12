package plugincatalog

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestProviderContractFromCatalogEntryCarriesSupportedHandoffSchemas(t *testing.T) {
	t.Parallel()

	entry := Provider{
		ID:                      "openspec-propose",
		RiskScore:               "write_analysis",
		CanonicalRole:           "archivist",
		CompatibilitySource:     "embedded",
		SupportedHandoffSchemas: []string{"openspec-native-change-schema"},
	}

	contract := ProviderContractFromCatalogEntry(entry)
	assert.Equal(t, []string{"openspec-native-change-schema"}, contract.SupportedHandoffSchemas)
}

func TestProviderContractFromCatalogEntryCarriesRuntimeIdentity(t *testing.T) {
	t.Parallel()

	contract := ProviderContractFromCatalogEntry(Provider{
		ID: "writing-plans", Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeHost, HostAPI: "strategist-host-skill/v1"},
	})
	assert.Equal(t, domain.RankedRuntimeHost, contract.Runtime.Kind)
	assert.Equal(t, "strategist-host-skill/v1", contract.Runtime.HostAPI)
}

func TestProviderContractFromCatalogEntryLeavesSupportedHandoffSchemasNilWhenUndeclared(t *testing.T) {
	t.Parallel()

	entry := Provider{ID: "brainstorming", RiskScore: "write_analysis", CanonicalRole: "ranger", CompatibilitySource: "embedded"}

	contract := ProviderContractFromCatalogEntry(entry)
	assert.Empty(t, contract.SupportedHandoffSchemas)
}

func TestProviderContractFromCatalogEntryNativeRoleNeedsNoDeclaredSchema(t *testing.T) {
	t.Parallel()

	entry := Provider{ID: "archivist", RiskScore: "write_analysis", CompatibilitySource: "native_role"}

	contract := ProviderContractFromCatalogEntry(entry)
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
