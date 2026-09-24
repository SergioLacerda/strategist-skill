package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validRoleContract() domain.RoleContract {
	return domain.RoleContract{
		SchemaVersion: domain.RoleContractSchemaVersion,
		Role:          "sniper",
		Slot:          "execution",
		Origin:        domain.RoleOriginNative,
		Extensibility: domain.RoleExtensibilityFixed,
		Must:          []string{"execute approved tasks"},
		MustNot:       []string{"bypass the approval gate"},
	}
}

func validProviderContract() domain.ProviderContract {
	return domain.ProviderContract{
		SchemaVersion:                 "strategist-provider-contract/v1",
		ID:                            "sniper",
		Version:                       "1.0.0",
		ProviderSchemaVersion:         "1",
		CanonicalRole:                 "sniper",
		RiskScore:                     "controlled",
		Source:                        domain.ProviderSourceNativeRole,
		Materialization:               domain.MaterializationActive,
		SupportedRoleContractVersions: []string{domain.RoleContractSchemaVersion},
	}
}

func TestRoleContractFromConfigReusesRoleConfigAsSourceOfTruth(t *testing.T) {
	t.Parallel()

	cfg := domain.RoleConfig{
		Role:    "sniper",
		Slot:    "execution",
		Must:    []string{"execute approved tasks"},
		MustNot: []string{"bypass the approval gate"},
	}
	contract := domain.RoleContractFromConfig(cfg, "schemas/handoff-archivist-to-sniper.schema.yaml")

	assert.Equal(t, domain.RoleContractSchemaVersion, contract.SchemaVersion)
	assert.Equal(t, cfg.Role, contract.Role)
	assert.Equal(t, cfg.Slot, contract.Slot)
	assert.Equal(t, cfg.Must, contract.Must)
	assert.Equal(t, cfg.MustNot, contract.MustNot)
	assert.Equal(t, domain.RoleOriginNative, contract.Origin)
	assert.Equal(t, domain.RoleExtensibilityFixed, contract.Extensibility)
	require.NoError(t, contract.Validate())
}

func TestRoleContractCarriesIndependentTaxonomy(t *testing.T) {
	t.Parallel()

	contract := domain.RoleContractFromConfig(domain.RoleConfig{
		Role:          "ranger",
		Slot:          "discovery",
		Origin:        domain.RoleOriginNative,
		Extensibility: domain.RoleExtensibilityPluggable,
	}, "schemas/handoff-ranger-to-archivist.schema.yaml")

	assert.Equal(t, domain.RoleOriginNative, contract.Origin)
	assert.Equal(t, domain.RoleExtensibilityPluggable, contract.Extensibility)
	require.NoError(t, contract.Validate())
}

func TestRoleContractValidationRejectsUnknownSlot(t *testing.T) {
	t.Parallel()

	contract := validRoleContract()
	contract.Slot = "planning"
	err := contract.Validate()
	require.ErrorContains(t, err, `slot "planning" is not one of`)
}

func TestProviderContractValidationRejectsUnknownEnumsAndMissingFields(t *testing.T) {
	t.Parallel()

	require.NoError(t, validProviderContract().Validate())

	missingVersions := validProviderContract()
	missingVersions.SupportedRoleContractVersions = nil
	err := missingVersions.Validate()
	require.ErrorContains(t, err, "supported_role_contract_versions must have at least one entry")

	badSource := validProviderContract()
	badSource.Source = "cloud"
	err = badSource.Validate()
	require.ErrorContains(t, err, `source "cloud" is not a known provider source`)

	badMaterialization := validProviderContract()
	badMaterialization.Materialization = "pending"
	err = badMaterialization.Validate()
	require.ErrorContains(t, err, `materialization "pending" is not a known materialization state`)
}

func TestCheckRoleAffinityRejectsRoleMismatch(t *testing.T) {
	t.Parallel()

	role := validRoleContract()
	provider := validProviderContract()
	provider.CanonicalRole = "archivist"

	result := provider.CheckRoleAffinity(role)
	assert.False(t, result.Compatible)
	assert.Contains(t, result.Reasons, domain.CompatibilityReason{
		Dimension: "role_affinity",
		Code:      "role_mismatch",
		Detail:    `provider declares roles [archivist], role contract is "sniper"`,
	})
}

func TestCheckRoleAffinityRejectsUnsupportedRoleContractVersion(t *testing.T) {
	t.Parallel()

	role := validRoleContract()
	provider := validProviderContract()
	provider.SupportedRoleContractVersions = []string{"strategist-role-contract/v0"}

	result := provider.CheckRoleAffinity(role)
	assert.False(t, result.Compatible)
	require.Len(t, result.Reasons, 1)
	assert.Equal(t, "role_contract_version", result.Reasons[0].Dimension)
	assert.Equal(t, "unsupported_role_contract_version", result.Reasons[0].Code)
}

func TestCheckRoleAffinityAcceptsMatchingRoleAndVersion(t *testing.T) {
	t.Parallel()

	result := validProviderContract().CheckRoleAffinity(validRoleContract())
	assert.True(t, result.Compatible)
	assert.Empty(t, result.Reasons)
}

func TestProviderContractValidateRejectsUnverifiedRoleAffinity(t *testing.T) {
	t.Parallel()

	contract := validProviderContract()
	contract.CanonicalRole = "pathfinder"

	err := contract.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not approved for activation")
}

func TestRoleContractValidateRejectsUnverifiedRole(t *testing.T) {
	t.Parallel()

	contract := validRoleContract()
	contract.Role = "jewelcrafter"

	err := contract.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not approved for activation")
}
