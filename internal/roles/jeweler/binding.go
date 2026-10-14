package jeweler

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

// The values below mirror the admitted ROSTER adapter contract. They are
// deliberately kept in the JEWELER boundary as validation constants: a
// catalog entry can describe a Weapon, but it cannot by itself provide a
// runtime delegate.
const (
	BindingRuntimeKind = "host"
	BindingHostAPI     = "strategist-host-skill/v1"
	BindingConnectorID = "strategist-host"
	BindingEntrypoint  = "invoke"
)

// ExecutableEnv names the host binding that supplies the admitted provider
// executable. It is never discovered from PATH.
const ExecutableEnv = "STRATEGIST_TREASURE_CHEST_EXECUTABLE"

// ProviderBinding is the explicit runtime handoff from ROSTER to JEWELER.
//
// ROSTER remains responsible for selecting and pinning the Weapon. The
// executable is an operator/host binding and is intentionally not discovered
// from PATH, catalog metadata, or the local Treasure Chest implementation.
// ProviderBinding is therefore a transport-neutral seam: future connectors
// may populate the same contract without changing Broker authorization.
type ProviderBinding struct {
	Role        string
	Slot        string
	RuntimeKind string
	HostAPI     string
	ConnectorID string
	Entrypoint  string
	Executable  string
	Identity    ProviderIdentity
}

// Validate proves that a binding addresses the canonical admitted JEWELER
// Weapon and contains a concrete delegate. Catalog admission alone is not a
// binding and must fail this check.
func (b ProviderBinding) Validate() error {
	if err := validateBindingFields(b); err != nil {
		return err
	}
	if err := validateBindingContract(b); err != nil {
		return err
	}
	if strings.TrimSpace(b.Executable) == "" {
		return fmt.Errorf("jeweler: external provider executable is not configured")
	}
	if err := b.Identity.Validate(); err != nil {
		return err
	}
	if b.Identity != AdmittedTreasureChestTMPIdentity() {
		return fmt.Errorf("jeweler: provider binding identity does not match admitted TREASURE-CHEST-TMP identity")
	}
	return nil
}

func validateBindingFields(b ProviderBinding) error {
	for name, value := range map[string]string{
		"role": b.Role, "slot": b.Slot, "runtime kind": b.RuntimeKind,
		"host API": b.HostAPI, "connector": b.ConnectorID,
		"entrypoint": b.Entrypoint,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("jeweler: provider binding %s is required", name)
		}
	}
	return nil
}

func validateBindingContract(b ProviderBinding) error {
	if b.Role != RoleID {
		return fmt.Errorf("jeweler: provider binding role %q is not canonical %q", b.Role, RoleID)
	}
	if b.Slot != SupportSlot {
		return fmt.Errorf("jeweler: provider binding slot %q is not supported slot %q", b.Slot, SupportSlot)
	}
	if b.RuntimeKind != BindingRuntimeKind || b.HostAPI != BindingHostAPI {
		return fmt.Errorf("jeweler: provider binding runtime %q/%q is not %q/%q", b.RuntimeKind, b.HostAPI, BindingRuntimeKind, BindingHostAPI)
	}
	if b.ConnectorID != BindingConnectorID || b.Entrypoint != BindingEntrypoint {
		return fmt.Errorf("jeweler: provider binding contract %q/%q is not %q/%q", b.ConnectorID, b.Entrypoint, BindingConnectorID, BindingEntrypoint)
	}
	return nil
}

// NewCLIProviderFromBinding converts the validated ROSTER handoff into the
// only production CLI adapter. It does not probe the delegate; Broker calls
// Readiness before serving a query or maintenance operation.
func NewCLIProviderFromBinding(binding ProviderBinding) (CLIProvider, error) {
	if err := binding.Validate(); err != nil {
		return CLIProvider{}, err
	}
	return CLIProvider{
		Executable: binding.Executable,
		Identity:   binding.Identity,
		Evidence:   TreasureChestTMPAdmissionEvidence(binding.Identity, binding.Executable),
	}, nil
}

// NewCLIProviderFromRoleWeaponBinding adapts the resolved ROSTER binding to
// JEWELER. The executable and provider identity remain separate arguments
// because neither belongs to the read-only RoleWeaponBinding projection: the
// former is a host delegate, and the latter is immutable provider evidence.
func NewCLIProviderFromRoleWeaponBinding(binding domain.RoleWeaponBinding, identity ProviderIdentity, executable string) (CLIProvider, error) {
	if binding.WeaponID != identity.Package || binding.WeaponVersion != identity.Version {
		return CLIProvider{}, fmt.Errorf("jeweler: resolved Role/Weapon identity does not match provider identity")
	}
	return NewCLIProviderFromBinding(ProviderBinding{
		Role:        binding.Role,
		Slot:        binding.Slot,
		RuntimeKind: binding.RuntimeKind,
		HostAPI:     BindingHostAPI,
		ConnectorID: binding.ConnectorID,
		Entrypoint:  binding.Entrypoint,
		Executable:  executable,
		Identity:    identity,
	})
}

// NewCLIProviderFromCompiledWeapon adapts the build-generated ROSTER Weapon
// entry to JEWELER. The compiled registry proves role/slot/runtime contract
// shape; the host still supplies the executable and the immutable admission
// identity separately.
func NewCLIProviderFromCompiledWeapon(weapon domain.CompiledWeapon, executable string) (CLIProvider, error) {
	identity := AdmittedTreasureChestTMPIdentity()
	if weapon.ID != identity.Package || weapon.Version != identity.Version {
		return CLIProvider{}, fmt.Errorf("jeweler: ROSTER Weapon %q@%s does not match admitted TREASURE-CHEST-TMP identity", weapon.ID, weapon.Version)
	}
	if !contains(weapon.SupportedRoles, RoleID) || !contains(weapon.SupportedSlots, SupportSlot) {
		return CLIProvider{}, fmt.Errorf("jeweler: ROSTER Weapon %s@%s is not bound to %s/%s", weapon.ID, weapon.Version, RoleID, SupportSlot)
	}
	if weapon.Runtime.Kind != BindingRuntimeKind || weapon.Runtime.HostAPI != BindingHostAPI {
		return CLIProvider{}, fmt.Errorf("jeweler: ROSTER Weapon %s@%s has runtime %q/%q, want %q/%q", weapon.ID, weapon.Version, weapon.Runtime.Kind, weapon.Runtime.HostAPI, BindingRuntimeKind, BindingHostAPI)
	}
	return NewCLIProviderFromBinding(ProviderBinding{
		Role:        RoleID,
		Slot:        SupportSlot,
		RuntimeKind: weapon.Runtime.Kind,
		HostAPI:     weapon.Runtime.HostAPI,
		ConnectorID: weapon.ConnectorID,
		Entrypoint:  weapon.Entrypoint,
		Executable:  executable,
		Identity:    identity,
	})
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
