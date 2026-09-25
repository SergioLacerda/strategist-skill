package provider

import (
	"fmt"
	"strings"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
)

func validateSource(source Source, requestedSlot string) []Reason {
	var reasons []Reason
	if err := source.Package.Validate(); err != nil {
		reasons = append(reasons, Reason{Code: "package_contract_invalid", Detail: err.Error()})
	}
	if err := source.Adapter.Validate(); err != nil {
		reasons = append(reasons, Reason{Code: "adapter_contract_invalid", Detail: err.Error()})
	}
	if source.Package.ID != "" && source.Adapter.ID != "" && source.Package.ID != source.Adapter.ID {
		reasons = append(reasons, Reason{Code: "identity_mismatch", Detail: fmt.Sprintf("package id %q differs from adapter id %q", source.Package.ID, source.Adapter.ID)})
	}
	reasons = append(reasons, validateProvenance(source)...)
	reasons = append(reasons, validateAdapterCompatibility(source, requestedSlot)...)
	reasons = append(reasons, validateLegacyView(source)...)
	reasons = append(reasons, validateExecutionBinding(source, requestedSlot)...)
	return reasons
}

func validateProvenance(source Source) []Reason {
	var reasons []Reason
	if source.Package.ArtifactURI != "" && !strings.HasPrefix(source.Package.ArtifactURI, "file://") && !strings.HasPrefix(source.Package.ArtifactURI, "local://") {
		reasons = append(reasons, Reason{Code: "provenance_not_local", Detail: "artifact_uri must use file:// or local:// for local onboarding"})
	}
	if strings.HasPrefix(source.Package.ArtifactURI, "file://") && strings.TrimPrefix(source.Package.ArtifactURI, "file://") != source.Dir {
		reasons = append(reasons, Reason{Code: "provenance_source_mismatch", Detail: "artifact_uri does not identify the validated source directory"})
	}
	return reasons
}

func validateAdapterCompatibility(source Source, requestedSlot string) []Reason {
	if requestedSlot != "" && !domain.IsValidSlot(requestedSlot) {
		return []Reason{{Code: "slot_invalid", Detail: fmt.Sprintf("%q is not a Strategist slot", requestedSlot)}}
	}
	return append(slotSupportReasons(source, requestedSlot), roleSlotReasons(source)...)
}

// slotSupportReasons reports a requested slot the adapter does not support.
func slotSupportReasons(source Source, requestedSlot string) []Reason {
	if requestedSlot == "" || contains(source.Adapter.SupportedSlots, requestedSlot) {
		return nil
	}
	return []Reason{{Code: "slot_unsupported", Detail: fmt.Sprintf("provider does not support slot %q", requestedSlot)}}
}

// roleSlotReasons reports every supported role that has no matching slot.
func roleSlotReasons(source Source) []Reason {
	var reasons []Reason
	for _, role := range source.Adapter.SupportedRoles {
		if !roleHasSlot(role, source.Adapter.SupportedSlots) {
			reasons = append(reasons, Reason{Code: "role_slot_mismatch", Detail: fmt.Sprintf("role %q has no matching supported slot", role)})
		}
	}
	return reasons
}

func validateLegacyView(source Source) []Reason {
	if source.Legacy == nil {
		return nil
	}
	return append(legacyIdentityConflicts(source), legacySlotConflicts(source)...)
}

// legacyConflict is the single reason shape for a skill.yaml that disagrees
// with the authoritative package or adapter manifest.
func legacyConflict(detail string) Reason {
	return Reason{Code: "legacy_authority_conflict", Detail: detail}
}

func legacyIdentityConflicts(source Source) []Reason {
	var reasons []Reason
	legacy := source.Legacy
	if legacy.ID != "" && legacy.ID != source.Package.ID {
		reasons = append(reasons, legacyConflict("skill.yaml id conflicts with package.yaml"))
	}
	if legacy.CanonicalRole != "" && !contains(source.Adapter.SupportedRoles, legacy.CanonicalRole) {
		reasons = append(reasons, legacyConflict("skill.yaml canonical_role conflicts with adapter.yaml"))
	}
	return reasons
}

func legacySlotConflicts(source Source) []Reason {
	var reasons []Reason
	for _, slot := range source.Legacy.SupportedSlots {
		if !contains(source.Adapter.SupportedSlots, slot) {
			reasons = append(reasons, legacyConflict(fmt.Sprintf("skill.yaml supported slot %q conflicts with adapter.yaml", slot)))
		}
	}
	return reasons
}

func roleHasSlot(role string, slots []string) bool {
	registered, ok := domain.DefaultRoleRegistry().Get(role)
	return ok && registered.ID == role && registered.Slot != "" && contains(slots, registered.Slot)
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
