package domain

import (
	"regexp"
	"strings"
)

func (c SkillPackageContract) missingFields() []string {
	missing := missingContractFields(map[string]string{
		"schema_version": c.SchemaVersion, "id": c.ID, "version": c.Version,
		"contract_version": c.ContractVersion,
	})
	missing = append(missing, missingContractCapabilities(c)...)
	missing = append(missing, invalidAffinityFields(c)...)
	missing = append(missing, missingContractEvidence(c)...)
	return missing
}

func invalidAffinityFields(c SkillPackageContract) []string {
	var invalid []string
	if invalidRole(c.SupportedRoles) {
		invalid = append(invalid, "supported_roles")
	}
	if invalidSlot(c.SupportedSlots) {
		invalid = append(invalid, "supported_slots")
	}
	if contradictoryAffinity(c.SupportedRoles, c.SupportedSlots) {
		invalid = append(invalid, "role/slot affinity")
	}
	return invalid
}

func missingContractCapabilities(c SkillPackageContract) []string {
	var missing []string
	if len(c.Capabilities) == 0 {
		missing = append(missing, "capabilities")
	}
	if !SupportsSkillPackageContract(c.ContractVersion) {
		missing = append(missing, "unsupported contract_version")
	}
	if len(c.SupportedSlots) == 0 {
		missing = append(missing, "supported_slots")
	}
	return missing
}

func missingContractEvidence(c SkillPackageContract) []string {
	var missing []string
	if !validPackageEvidenceState(c.EvidenceState) {
		missing = append(missing, "evidence_state")
	}
	if !validPackageEvidenceState(c.Provenance.VerificationState) {
		missing = append(missing, "provenance.verification_state")
	}
	for name, digest := range map[string]string{"provenance.original_digest": c.Provenance.OriginalDigest, "provenance.normalized_digest": c.Provenance.NormalizedDigest} {
		if digest == "" {
			missing = append(missing, name)
		} else if !validSHA256Digest(digest) {
			missing = append(missing, name)
		}
	}
	return missing
}

// affinitySlot returns the slot a provider role affinity implies: the slot of a
// registered slot-bound role, "auxiliary" for the auxiliary affinity, and "" for
// anything else (including slotless roles such as Scout).
func affinitySlot(role string) string {
	if role == "auxiliary" {
		return "auxiliary"
	}
	if r, ok := DefaultRoleRegistry().Get(role); ok && r.ID == role {
		return r.Slot
	}
	return ""
}

func invalidRole(roles []string) bool {
	for _, role := range roles {
		if affinitySlot(role) == "" {
			return true
		}
	}
	return false
}

func invalidSlot(slots []string) bool {
	for _, slot := range slots {
		if slot != "discovery" && slot != "refinement" && slot != "execution" && slot != "auxiliary" {
			return true
		}
	}
	return false
}

func contradictoryAffinity(roles, slots []string) bool {
	for _, role := range roles {
		expected := affinitySlot(role)
		if expected != "" && !containsString(slots, expected) {
			return true
		}
	}
	return false
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func missingContractFields(fields map[string]string) []string {
	var missing []string
	for name, value := range fields {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	return missing
}

func validPackageEvidenceState(state PackageEvidenceState) bool {
	switch state {
	case PackageEvidenceDeclared, PackageEvidenceVerified, PackageEvidenceUnknown,
		PackageEvidenceUnsupported, PackageEvidenceFailed, PackageEvidenceBlocked:
		return true
	default:
		return false
	}
}

var sha256DigestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func validSHA256Digest(digest string) bool {
	return sha256DigestPattern.MatchString(digest)
}
