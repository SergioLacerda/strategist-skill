package domain

import (
	"fmt"
	"strings"
)

// PackageEvidenceState describes the strength of package metadata. Declared
// metadata is useful for diagnostics but is never runtime authorization.
type PackageEvidenceState string

const (
	// PackageEvidenceDeclared indicates metadata supplied by the package.
	PackageEvidenceDeclared PackageEvidenceState = "declared"
	// PackageEvidenceVerified indicates metadata independently verified.
	PackageEvidenceVerified PackageEvidenceState = "verified"
	// PackageEvidenceUnknown indicates metadata with unknown reliability.
	PackageEvidenceUnknown PackageEvidenceState = "unknown"
	// PackageEvidenceUnsupported indicates metadata for an unsupported package.
	PackageEvidenceUnsupported PackageEvidenceState = "unsupported"
	// PackageEvidenceFailed indicates metadata validation failed.
	PackageEvidenceFailed PackageEvidenceState = "failed"
	// PackageEvidenceBlocked indicates policy blocked metadata validation.
	PackageEvidenceBlocked PackageEvidenceState = "blocked"
	// CurrentSkillPackageContractVersion is the active package contract version.
	CurrentSkillPackageContractVersion = "skill-package/v1"
	// PreviousSkillPackageContractVersion is the supported compatibility version.
	PreviousSkillPackageContractVersion = "skill-package/v0"
)

// SkillPackageContract is the canonical projection of package and adapter
// metadata. Binding and readiness remain owned by PluginLock/SlotBinding and
// the runtime probe respectively.
type SkillPackageContract struct {
	SchemaVersion           string               `yaml:"schema_version"`
	ID                      string               `yaml:"id"`
	Version                 string               `yaml:"version"`
	ContractVersion         string               `yaml:"contract_version"`
	Capabilities            []string             `yaml:"capabilities,omitempty"`
	SupportedRoles          []string             `yaml:"supported_roles,omitempty"`
	SupportedSlots          []string             `yaml:"supported_slots,omitempty"`
	SupportedHandoffSchemas []string             `yaml:"supported_handoff_schemas,omitempty"`
	CompatibilityRange      string               `yaml:"compatibility_range,omitempty"`
	Provenance              PackageProvenance    `yaml:"provenance"`
	EvidenceState           PackageEvidenceState `yaml:"evidence_state"`
	TrustTier               string               `yaml:"trust_tier,omitempty"`
	Freshness               string               `yaml:"freshness,omitempty"`
	Limitations             []string             `yaml:"limitations,omitempty"`
}

// NewSkillPackageContract projects the existing publisher and adapter records
// into the canonical contract without changing ownership of either record.
func NewSkillPackageContract(pkg PluginPackage, adapter AdapterContract) SkillPackageContract {
	version := pkg.Version
	if strings.TrimSpace(version) == "" {
		version = "unknown"
	}
	return SkillPackageContract{
		SchemaVersion: pkg.SchemaVersion, ID: pkg.ID, Version: version,
		ContractVersion: CurrentSkillPackageContractVersion, Capabilities: append([]string(nil), adapter.Capabilities...),
		SupportedRoles:          append([]string(nil), adapter.SupportedRoles...),
		SupportedSlots:          append([]string(nil), adapter.SupportedSlots...),
		SupportedHandoffSchemas: append([]string(nil), adapter.SupportedHandoffSchemas...),
		CompatibilityRange:      adapter.PluginAPIRange,
		EvidenceState:           PackageEvidenceDeclared,
		Provenance: PackageProvenance{
			OriginalDigest: pkg.Digest, NormalizedDigest: pkg.Digest,
			License: pkg.License, VerificationState: PackageEvidenceDeclared,
		},
	}
}

// SupportsSkillPackageContract reports the explicit current/N-1 compatibility
// window. Unknown versions are rejected instead of being silently migrated.
func SupportsSkillPackageContract(version string) bool {
	return version == CurrentSkillPackageContractVersion || version == PreviousSkillPackageContractVersion
}

// PackageProvenance records source facts without claiming facts that the
// ingestion process cannot independently establish.
type PackageProvenance struct {
	CanonicalSource    string               `yaml:"canonical_source,omitempty"`
	UpstreamRevision   string               `yaml:"upstream_revision,omitempty"`
	OriginalDigest     string               `yaml:"original_digest,omitempty"`
	NormalizedDigest   string               `yaml:"normalized_digest,omitempty"`
	License            string               `yaml:"license,omitempty"`
	AcquisitionMethod  string               `yaml:"acquisition_method,omitempty"`
	AcquiredAt         string               `yaml:"acquired_at,omitempty"`
	Transformation     string               `yaml:"transformation,omitempty"`
	LocalModifications string               `yaml:"local_modifications,omitempty"`
	VerificationState  PackageEvidenceState `yaml:"verification_state"`
}

// Validate checks the portable contract shape. It deliberately does not
// promote package metadata to binding or runtime readiness.
func (c SkillPackageContract) Validate() error {
	missing := c.missingFields()
	if len(missing) > 0 {
		return fmt.Errorf("skill package contract: required/valid fields: %s", strings.Join(missing, ", "))
	}
	return nil
}
