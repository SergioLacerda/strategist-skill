package jeweler

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func validProviderBinding() ProviderBinding {
	return ProviderBinding{
		Role:        RoleID,
		Slot:        SupportSlot,
		RuntimeKind: BindingRuntimeKind,
		HostAPI:     BindingHostAPI,
		ConnectorID: BindingConnectorID,
		Entrypoint:  BindingEntrypoint,
		Executable:  "/opt/treasure-chest",
		Identity:    AdmittedTreasureChestTMPIdentity(),
	}
}

func TestProviderBindingRequiresExplicitDelegate(t *testing.T) {
	binding := validProviderBinding()
	binding.Executable = ""

	err := binding.Validate()
	require.ErrorContains(t, err, "external provider executable is not configured")
}

func TestProviderBindingRejectsCatalogOnlyOrWrongRuntimeContract(t *testing.T) {
	binding := validProviderBinding()
	binding.ConnectorID = "strategist-embedded"

	err := binding.Validate()
	require.ErrorContains(t, err, "provider binding contract")
}

func TestProviderBindingRejectsNonAdmittedIdentity(t *testing.T) {
	binding := validProviderBinding()
	binding.Identity.Digest = "sha256:stale"

	err := binding.Validate()
	require.ErrorContains(t, err, "does not match admitted TREASURE-CHEST-TMP identity")
}

func TestNewCLIProviderFromBindingCarriesAdmissionEvidence(t *testing.T) {
	provider, err := NewCLIProviderFromBinding(validProviderBinding())
	require.NoError(t, err)
	require.Equal(t, "/opt/treasure-chest", provider.Executable)
	require.Equal(t, EvidenceCertified, provider.Evidence.Package)
	require.Equal(t, EvidenceCertified, provider.Evidence.Identity)
	require.Equal(t, EvidenceCertified, provider.Evidence.Delegate)
	require.Empty(t, provider.Evidence.Probe)
	require.Empty(t, provider.Evidence.Health)
}

func TestNewCLIProviderFromRoleWeaponBindingRequiresResolvedIdentity(t *testing.T) {
	identity := AdmittedTreasureChestTMPIdentity()
	provider, err := NewCLIProviderFromRoleWeaponBinding(domain.RoleWeaponBinding{
		Role:          RoleID,
		Slot:          SupportSlot,
		WeaponID:      identity.Package,
		WeaponVersion: identity.Version,
		RuntimeKind:   BindingRuntimeKind,
		ConnectorID:   BindingConnectorID,
		Entrypoint:    BindingEntrypoint,
	}, identity, "/opt/treasure-chest")
	require.NoError(t, err)
	require.Equal(t, "/opt/treasure-chest", provider.Executable)

	_, err = NewCLIProviderFromRoleWeaponBinding(domain.RoleWeaponBinding{
		Role:          RoleID,
		Slot:          SupportSlot,
		WeaponID:      "catalog-only",
		WeaponVersion: identity.Version,
	}, identity, "/opt/treasure-chest")
	require.ErrorContains(t, err, "resolved Role/Weapon identity")
}

func TestNewCLIProviderFromCompiledWeaponRequiresRosterContract(t *testing.T) {
	identity := AdmittedTreasureChestTMPIdentity()
	provider, err := NewCLIProviderFromCompiledWeapon(domain.CompiledWeapon{
		ID:             identity.Package,
		Version:        identity.Version,
		SupportedRoles: []string{RoleID},
		SupportedSlots: []string{SupportSlot},
		Runtime: domain.WeaponRuntime{
			Kind:    BindingRuntimeKind,
			HostAPI: BindingHostAPI,
		},
		ConnectorID: BindingConnectorID,
		Entrypoint:  BindingEntrypoint,
	}, "/opt/treasure-chest")
	require.NoError(t, err)
	require.Equal(t, "/opt/treasure-chest", provider.Executable)

	_, err = NewCLIProviderFromCompiledWeapon(domain.CompiledWeapon{
		ID:             identity.Package,
		Version:        identity.Version,
		SupportedRoles: []string{"ranger"},
		SupportedSlots: []string{SupportSlot},
		Runtime:        domain.WeaponRuntime{Kind: BindingRuntimeKind, HostAPI: BindingHostAPI},
		ConnectorID:    BindingConnectorID,
		Entrypoint:     BindingEntrypoint,
	}, "/opt/treasure-chest")
	require.ErrorContains(t, err, "not bound")
}
