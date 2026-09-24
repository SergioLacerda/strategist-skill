package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestRoleConfig_Validate_Valid(t *testing.T) {
	r := domain.RoleConfig{Role: "sniper", Slot: "execution"}
	require.NoError(t, r.Validate())
}

func TestRoleConfig_Validate_MissingFields(t *testing.T) {
	require.Error(t, domain.RoleConfig{}.Validate())
	require.Error(t, domain.RoleConfig{Role: "sniper"}.Validate())
}

func TestRoleConfig_Validate_UnknownSlot(t *testing.T) {
	err := domain.RoleConfig{Role: "sniper", Slot: "bogus"}.Validate()
	require.ErrorContains(t, err, `slot "bogus"`)
}

func TestRoleConfig_Validate_Taxonomy(t *testing.T) {
	r := domain.RoleConfig{
		Role:          "ranger",
		Slot:          "discovery",
		Origin:        domain.RoleOriginNative,
		Extensibility: domain.RoleExtensibilityPluggable,
	}
	require.NoError(t, r.Validate())

	r.Origin = "provider"
	require.ErrorContains(t, r.Validate(), "role origin")
	r.Origin = domain.RoleOriginNative
	r.Extensibility = "conditional"
	require.ErrorContains(t, r.Validate(), "role extensibility")
}

func TestRoleConfig_Validate_RejectsConflictingLegacyPluggable(t *testing.T) {
	legacyFalse := false
	r := domain.RoleConfig{
		Role:          "ranger",
		Slot:          "discovery",
		Extensibility: domain.RoleExtensibilityPluggable,
		Pluggable:     &legacyFalse,
	}
	require.ErrorContains(t, r.Validate(), "pluggable conflicts with extensibility")
}

func TestRoleConfig_Validate_RejectsUnverifiedRoleActivation(t *testing.T) {
	r := domain.RoleConfig{Role: "pathfinder", Extensibility: domain.RoleExtensibilityFixed}
	require.ErrorContains(t, r.Validate(), "not approved for activation")
}

func TestRoleSlotMap_Validate_Valid(t *testing.T) {
	m := domain.RoleSlotMap{"discovery": "ranger", "refinement": "archivist", "execution": "sniper"}
	require.NoError(t, m.Validate())
}

func TestRoleSlotMap_Validate_MissingSlots(t *testing.T) {
	m := domain.RoleSlotMap{"discovery": "ranger"}
	err := m.Validate()
	require.ErrorContains(t, err, "missing slot: refinement")
	require.ErrorContains(t, err, "missing slot: execution")
}
