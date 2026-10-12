package jeweler

import "strings"

// treasureChestTMPAdmissionIdentity is the host-admitted identity for the
// temporary root-backed Weapon. It is derived from the committed embedded
// package lock and is shared by query and maintenance bindings.
var treasureChestTMPAdmissionIdentity = ProviderIdentity{
	Package:    "treasure-chest-tmp",
	Version:    "0.1.0",
	API:        "knowledge-api/v1",
	Digest:     "sha256:8d3110a44f85c668dca219ae78972c1e5bcca9b38dd785fce612b78f92e14c9b",
	Provenance: "local:strategist-skill/treasure-chest",
}

// AdmittedTreasureChestTMPIdentity returns the immutable temporary Weapon identity.
// Returning a value prevents callers from mutating the process-wide pin.
func AdmittedTreasureChestTMPIdentity() ProviderIdentity {
	return treasureChestTMPAdmissionIdentity
}

// TreasureChestTMPAdmissionEvidence converts the explicit binding into static
// admission evidence. Live probe and health remain unknown until the bound
// executable reports them through CLIProvider.Readiness.
func TreasureChestTMPAdmissionEvidence(identity ProviderIdentity, executable string) EvidenceVector {
	return EvidenceVector{
		Package:    admissionMatch(identity.Package, treasureChestTMPAdmissionIdentity.Package),
		Identity:   admissionIdentityMatch(identity),
		Provenance: admissionMatch(identity.Provenance, treasureChestTMPAdmissionIdentity.Provenance),
		API:        admissionMatch(identity.API, treasureChestTMPAdmissionIdentity.API),
		Delegate:   admissionDelegateEvidence(executable),
	}
}

func admissionIdentityMatch(identity ProviderIdentity) EvidenceState {
	if identity.Version == "" || identity.Digest == "" {
		return EvidenceUnknown
	}
	if identity.Version != treasureChestTMPAdmissionIdentity.Version || identity.Digest != treasureChestTMPAdmissionIdentity.Digest {
		return EvidenceBlocked
	}
	return EvidenceCertified
}

func admissionMatch(actual, expected string) EvidenceState {
	if strings.TrimSpace(actual) == "" {
		return EvidenceUnknown
	}
	if actual != expected {
		return EvidenceBlocked
	}
	return EvidenceCertified
}

func admissionDelegateEvidence(executable string) EvidenceState {
	if strings.TrimSpace(executable) == "" {
		return EvidenceUnknown
	}
	return EvidenceCertified
}
