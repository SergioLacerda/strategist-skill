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
