package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func validSkillPackageContract() SkillPackageContract {
	return SkillPackageContract{
		SchemaVersion: "skill-package/v1", ID: "example", Version: "1.0.0",
		ContractVersion: "skill-package/v1", Capabilities: []string{"mission.refine"},
		SupportedRoles: []string{"archivist"}, SupportedSlots: []string{"refinement"},
		EvidenceState: PackageEvidenceDeclared,
		Provenance:    PackageProvenance{OriginalDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", NormalizedDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", VerificationState: PackageEvidenceDeclared},
	}
}

func TestSkillPackageContractValidateAcceptsCompleteContract(t *testing.T) {
	require.NoError(t, validSkillPackageContract().Validate())
}

func TestSkillPackageContractValidateRejectsMissingAndInvalidEvidence(t *testing.T) {
	contract := validSkillPackageContract()
	contract.Capabilities = nil
	contract.EvidenceState = "fabricated"
	contract.Provenance.VerificationState = "fabricated"

	err := contract.Validate()
	require.Error(t, err)
	require.ErrorContains(t, err, "capabilities")
	require.ErrorContains(t, err, "evidence_state")
	require.ErrorContains(t, err, "provenance.verification_state")
}

func TestPackageEvidenceStatesRemainExplicit(t *testing.T) {
	for _, state := range []PackageEvidenceState{
		PackageEvidenceDeclared, PackageEvidenceVerified, PackageEvidenceUnknown,
		PackageEvidenceUnsupported, PackageEvidenceFailed, PackageEvidenceBlocked,
	} {
		contract := validSkillPackageContract()
		contract.EvidenceState = state
		contract.Provenance.VerificationState = state
		require.NoError(t, contract.Validate(), state)
	}
}

func TestNewSkillPackageContractProjectsExistingRecords(t *testing.T) {
	contract := NewSkillPackageContract(
		PluginPackage{SchemaVersion: "package/v1", ID: "example", Version: "1.0.0", Digest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", License: "MIT"},
		AdapterContract{SupportedSlots: []string{"refinement"}, SupportedRoles: []string{"archivist"}, Capabilities: []string{"mission.refine"}, PluginAPIRange: "v1"},
	)
	require.Equal(t, "example", contract.ID)
	require.Equal(t, []string{"mission.refine"}, contract.Capabilities)
	require.Equal(t, "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", contract.Provenance.OriginalDigest)
	require.NoError(t, contract.Validate())
}

func TestSupportsSkillPackageContractUsesCurrentAndPreviousWindow(t *testing.T) {
	require.True(t, SupportsSkillPackageContract(CurrentSkillPackageContractVersion))
	require.True(t, SupportsSkillPackageContract(PreviousSkillPackageContractVersion))
	require.False(t, SupportsSkillPackageContract("skill-package/v99"))
}

func TestSkillPackageContractRejectsMalformedDigestEvidence(t *testing.T) {
	contract := validSkillPackageContract()
	contract.Provenance.NormalizedDigest = "sha256:not-a-digest"
	if err := contract.Validate(); err == nil {
		t.Fatal("expected malformed digest evidence to be rejected")
	}
}

func TestSkillPackageContractRejectsContradictoryRoleAffinity(t *testing.T) {
	contract := validSkillPackageContract()
	contract.SupportedRoles = []string{"archivist"}
	contract.SupportedSlots = []string{"discovery"}
	if err := contract.Validate(); err == nil {
		t.Fatal("expected contradictory role affinity to be rejected")
	}
}
