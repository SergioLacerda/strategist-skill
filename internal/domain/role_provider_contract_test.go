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

func TestCheckRoleCompatibilityRejectsRoleMismatch(t *testing.T) {
	t.Parallel()

	role := validRoleContract()
	provider := validProviderContract()
	provider.CanonicalRole = "archivist"

	result := provider.CheckRoleCompatibility(role)
	assert.False(t, result.Compatible)
	assert.Contains(t, result.Reasons, domain.CompatibilityReason{
		Dimension: "canonical_role",
		Code:      "role_mismatch",
		Detail:    `provider declares canonical_role "archivist", role contract is "sniper"`,
	})
}

func TestCheckRoleCompatibilityRejectsUnsupportedRoleContractVersion(t *testing.T) {
	t.Parallel()

	role := validRoleContract()
	provider := validProviderContract()
	provider.SupportedRoleContractVersions = []string{"strategist-role-contract/v0"}

	result := provider.CheckRoleCompatibility(role)
	assert.False(t, result.Compatible)
	require.Len(t, result.Reasons, 1)
	assert.Equal(t, "role_contract_version", result.Reasons[0].Dimension)
	assert.Equal(t, "unsupported_role_contract_version", result.Reasons[0].Code)
}

func TestCheckRoleCompatibilityRejectsUnsupportedHandoffSchema(t *testing.T) {
	t.Parallel()

	role := validRoleContract()
	role.HandoffSchema = "schemas/handoff-archivist-to-sniper.schema.yaml"
	provider := validProviderContract()
	provider.Source = domain.ProviderSourceEmbedded
	provider.SupportedHandoffSchemas = []string{"some-other-schema.yaml"}

	result := provider.CheckRoleCompatibility(role)
	assert.False(t, result.Compatible)
	require.Len(t, result.Reasons, 1)
	assert.Equal(t, "handoff_schema", result.Reasons[0].Dimension)
	assert.Equal(t, "unsupported_handoff_schema", result.Reasons[0].Code)
}

func TestCheckRoleCompatibilityAcceptsMatchingHandoffSchema(t *testing.T) {
	t.Parallel()

	role := validRoleContract()
	role.HandoffSchema = "schemas/handoff-archivist-to-sniper.schema.yaml"
	provider := validProviderContract()
	provider.Source = domain.ProviderSourceEmbedded
	provider.SupportedHandoffSchemas = []string{"schemas/handoff-archivist-to-sniper.schema.yaml"}

	result := provider.CheckRoleCompatibility(role)
	assert.True(t, result.Compatible)
}

func TestCheckRoleCompatibilitySkipsHandoffSchemaDimensionWhenRoleDeclaresNone(t *testing.T) {
	t.Parallel()

	role := validRoleContract() // HandoffSchema is "" (zero value) — e.g. Sniper, the terminal role
	provider := validProviderContract()
	provider.SupportedHandoffSchemas = nil

	result := provider.CheckRoleCompatibility(role)
	assert.True(t, result.Compatible, "a role with no declared handoff schema imposes no constraint on this dimension")
}

func TestCheckRoleCompatibilitySkipsHandoffSchemaDimensionForNativeRoleProviders(t *testing.T) {
	t.Parallel()

	role := validRoleContract()
	role.HandoffSchema = "schemas/handoff-archivist-to-sniper.schema.yaml"
	provider := validProviderContract() // Source: domain.ProviderSourceNativeRole (see validProviderContract())
	provider.SupportedHandoffSchemas = nil

	result := provider.CheckRoleCompatibility(role)
	assert.True(t, result.Compatible, "a native role trivially satisfies its own handoff contract")
}

func TestCheckRoleCompatibilityAcceptsMatchingRoleAndVersion(t *testing.T) {
	t.Parallel()

	result := validProviderContract().CheckRoleCompatibility(validRoleContract())
	assert.True(t, result.Compatible)
	assert.Empty(t, result.Reasons)
}

func TestResolveProviderBindingComposesRoleAndProviderWithoutPersisting(t *testing.T) {
	t.Parallel()

	role := validRoleContract()
	provider := validProviderContract()

	binding := domain.ResolveProviderBinding(role, provider)
	assert.Equal(t, role, binding.Role)
	assert.Equal(t, provider, binding.Provider)
	assert.True(t, binding.Compatibility.Compatible)
}
