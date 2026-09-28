package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
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

// The boolean `pluggable` key was replaced by `extensibility`. A role file that
// still carries it is rejected with an error that names the replacement, instead
// of being silently reinterpreted.
func TestRoleConfig_Validate_RejectsTheRemovedPluggableKey(t *testing.T) {
	for _, value := range []bool{true, false} {
		v := value
		r := domain.RoleConfig{Role: "ranger", Slot: "discovery", Extensibility: domain.RoleExtensibilityPluggable, Pluggable: &v}
		err := r.Validate()
		require.Error(t, err, "pluggable=%v", value)
		assert.Contains(t, err.Error(), "`pluggable` key was removed")
		assert.Contains(t, err.Error(), "extensibility: fixed|pluggable")
	}
}

func TestRoleConfig_Validate_ExtensibilityIsTheOnlyDeclaration(t *testing.T) {
	require.NoError(t, domain.RoleConfig{Role: "scout", Extensibility: domain.RoleExtensibilityFixed}.Validate())
	require.ErrorContains(t, domain.RoleConfig{Role: "scout"}.Validate(), "extensibility: fixed must be explicit")
	assert.Equal(t, domain.RoleExtensibilityFixed, domain.RoleConfig{Role: "scout"}.EffectiveExtensibility())
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
