package domain

import (
	"fmt"
	"strings"
)

// RoleWeaponBinding is the single resolved Role→Weapon identity consumed by
// install, readiness, and runtime adapters.
type RoleWeaponBinding struct {
	Role                string
	Slot                string
	WeaponID            string
	WeaponVersion       string
	Mode                string
	WeaponDigest        string
	SourceDigest        string
	RoleDigest          string
	BindingDigest       string
	ExecutionMode       string
	Origin              string
	RuntimeKind         string
	ConnectorID         string
	Entrypoint          string
	CertificationDigest string
}

// WeaponRefMatchesBinding reports whether an active.yaml slot value (an id or an
// "id@version" reference) names the Weapon a lock binding records. A Custom
// package id that already carries its version matches by equality; otherwise
// the id must match and, when the reference pins a version, so must the version.
func WeaponRefMatchesBinding(ref string, binding SlotBinding) bool {
	if ref == "" {
		return false
	}
	if binding.InstalledInstanceID == ref {
		return true
	}
	id, version := ParseWeaponRef(ref)
	if binding.InstalledInstanceID != id {
		return false
	}
	return version == "" || binding.WeaponVersion == version
}

// ResolveRoleWeaponBinding reconciles operator intent, the persisted lock,
// and the build registry. It performs no filesystem or connector I/O.
func ResolveRoleWeaponBinding(active ActiveConfig, lock PluginLockFile, registry CompiledRegistry, role, slot string) (RoleWeaponBinding, error) {
	providerID := active.Slots[slot]
	if providerID == "" {
		return RoleWeaponBinding{}, fmt.Errorf("role/weapon binding: active.yaml has no Weapon for slot %q", slot)
	}
	binding, err := SingleLockBindingForSlot(lock, slot)
	if err != nil {
		return RoleWeaponBinding{}, err
	}
	if !binding.ValidMode() {
		return RoleWeaponBinding{}, fmt.Errorf("role/weapon binding: slot %q has invalid mode %q", slot, binding.Mode)
	}
	if !WeaponRefMatchesBinding(providerID, binding) {
		return RoleWeaponBinding{}, fmt.Errorf("role/weapon binding: active Weapon %q does not match lock Weapon %q for slot %q", providerID, binding.InstalledInstanceID, slot)
	}
	if binding.Role != "" && binding.Role != role {
		return RoleWeaponBinding{}, fmt.Errorf("role/weapon binding: lock Role %q does not match requested Role %q", binding.Role, role)
	}

	if binding.EffectiveMode() == SlotBindingModeRanked {
		return resolveRankedBinding(binding, registry, role, slot)
	}
	return RoleWeaponBinding{
		Role: role, Slot: slot, WeaponID: binding.InstalledInstanceID, WeaponVersion: binding.WeaponVersion, Mode: SlotBindingModeCustom,
		WeaponDigest: binding.WeaponDigest, BindingDigest: binding.BindingDigest, Origin: binding.Origin,
		SourceDigest: binding.SourceDigest, ExecutionMode: binding.ExecutionMode,
		RuntimeKind: binding.RuntimeKind, ConnectorID: binding.ConnectorID, Entrypoint: binding.Entrypoint,
		CertificationDigest: binding.CertificationDigest,
	}, nil
}

func resolveRankedBinding(binding SlotBinding, registry CompiledRegistry, role, slot string) (RoleWeaponBinding, error) {
	if strings.TrimSpace(binding.WeaponVersion) == "" {
		return RoleWeaponBinding{}, fmt.Errorf("role/weapon binding: Ranked lock for %s/%s has no weapon version; re-run install to migrate the lock (certified: %s)", role, slot, registry.RankedOfferNames(role, slot))
	}
	compiled, ok := registry.RankedBinding(role, slot, binding.InstalledInstanceID, binding.WeaponVersion)
	if !ok {
		return RoleWeaponBinding{}, fmt.Errorf("role/weapon binding: no compiled Ranked binding for %s/%s/%s (certified: %s)", role, slot, WeaponIdentity(binding.InstalledInstanceID, binding.WeaponVersion), registry.RankedOfferNames(role, slot))
	}
	if compiled.WeaponDigest != binding.WeaponDigest || compiled.SourceDigest != binding.SourceDigest || compiled.BindingDigest != binding.BindingDigest || compiled.ExecutionMode != binding.ExecutionMode {
		return RoleWeaponBinding{}, fmt.Errorf("role/weapon binding: Ranked lock does not match compiled binding for %s/%s", role, slot)
	}
	return RoleWeaponBinding{
		Role: role, Slot: slot, WeaponID: compiled.WeaponID, WeaponVersion: compiled.WeaponVersion, Mode: SlotBindingModeRanked,
		WeaponDigest: compiled.WeaponDigest, RoleDigest: compiled.RoleDigest, BindingDigest: compiled.BindingDigest,
		SourceDigest: compiled.SourceDigest, ExecutionMode: compiled.ExecutionMode,
		Origin: string(WeaponOriginEmbedded), RuntimeKind: compiled.Runtime.Kind, ConnectorID: compiled.ConnectorID,
		Entrypoint: compiled.Entrypoint, CertificationDigest: compiled.CertificationDigest,
	}, nil
}

// RankedOfferNames names the certified id@version bindings offered for role
// and slot, for diagnostics; "none" when the registry offers no binding.
func (r CompiledRegistry) RankedOfferNames(role, slot string) string {
	offers := r.RankedBindingsFor(role, slot)
	if len(offers) == 0 {
		return "none"
	}
	names := make([]string, 0, len(offers))
	for _, offer := range offers {
		names = append(names, WeaponIdentity(offer.WeaponID, offer.WeaponVersion))
	}
	return strings.Join(names, ", ")
}
