package domain_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRoleWeaponIdentityKeepsFamiliesSeparate(t *testing.T) {
	t.Parallel()

	registry := domain.CompiledRegistry{
		Roles: []domain.CompiledRole{{ID: "brainstorming", Slot: "discovery", ContractDigest: "sha256:role"}},
	}
	binding := domain.SlotBinding{
		Role: "brainstorming", Slot: "discovery", InstalledInstanceID: "brainstorming@1.0.0", WeaponVersion: "1.0.0",
	}

	require.NoError(t, domain.ValidateRoleWeaponIdentity(registry, "brainstorming", "discovery", binding))
}

func TestValidateRoleWeaponIdentityRequiresRegistryRoleAndSlotParity(t *testing.T) {
	t.Parallel()

	registry := validCompiledRegistry()
	binding := domain.SlotBinding{Role: "missing", Slot: "discovery", InstalledInstanceID: "brainstorming", WeaponVersion: "1.0.0"}

	err := domain.ValidateRoleWeaponIdentity(registry, "missing", "discovery", binding)
	require.ErrorContains(t, err, "not present in the compiled registry")

	binding.Role = "ranger"
	binding.Slot = "refinement"
	err = domain.ValidateRoleWeaponIdentity(registry, "ranger", "refinement", binding)
	assert.ErrorContains(t, err, "owns slot")
}
