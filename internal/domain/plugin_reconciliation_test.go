package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func reconciliationContract() SkillPackageContract {
	return SkillPackageContract{
		SchemaVersion: "package/v1", ID: "brainstorming", Version: "1.0.0",
		ContractVersion: CurrentSkillPackageContractVersion, Capabilities: []string{"role.ranger"},
		SupportedRoles: []string{"ranger"}, SupportedSlots: []string{"discovery"},
		EvidenceState: PackageEvidenceDeclared,
		Provenance:    PackageProvenance{NormalizedDigest: "sha256:package", VerificationState: PackageEvidenceDeclared},
	}
}

func reconciliationLock(mode string) PluginLockFile {
	return PluginLockFile{
		Bindings: []SlotBinding{{Slot: "discovery", InstalledInstanceID: "brainstorming", Mode: mode}},
		Lock: PluginLock{Nodes: []PluginLockNode{
			{ID: "brainstorming", Kind: string(PluginResourceAdapter), Digest: "sha256:package"},
			{ID: "ranger:brainstorming", Kind: string(PluginResourceBinding), Digest: "sha256:binding"},
		}},
	}
}

func TestReconcileCustomPackageBindingAcceptsMatchingLock(t *testing.T) {
	require.NoError(t, ReconcileCustomPackageBinding(reconciliationLock(SlotBindingModeCustom), "ranger", "discovery", reconciliationContract()))
}

func TestReconcileCustomPackageBindingRejectsMismatchWithoutRepair(t *testing.T) {
	contract := reconciliationContract()
	contract.Provenance.NormalizedDigest = "sha256:other"
	err := ReconcileCustomPackageBinding(reconciliationLock(SlotBindingModeCustom), "ranger", "discovery", contract)
	require.ErrorContains(t, err, "adapter digest mismatch")
}

func TestReconcileCustomPackageBindingRejectsRankedMirror(t *testing.T) {
	err := ReconcileCustomPackageBinding(reconciliationLock(SlotBindingModeRanked), "ranger", "discovery", reconciliationContract())
	require.ErrorContains(t, err, "not custom")
}

func TestReconcileCustomPackageBindingRejectsAbsentBinding(t *testing.T) {
	lock := reconciliationLock(SlotBindingModeCustom)
	lock.Bindings = nil
	err := ReconcileCustomPackageBinding(lock, "ranger", "discovery", reconciliationContract())
	require.ErrorContains(t, err, "custom binding missing")
}

func TestReconcileRankedPackageCertificationIgnoresRuntimeMirror(t *testing.T) {
	stamp := CatalogRankedStamp{ID: "brainstorming", Roles: []string{"ranger"}, Ranked: true, CertificationDigest: "sha256:cert"}
	// No plugins.lock mirror is supplied: the certified catalog is authoritative.
	require.NoError(t, ReconcileRankedPackageCertification("ranger", reconciliationContract(), stamp))
}

func TestReconcileRankedPackageCertificationRejectsUncertifiedOrWrongRole(t *testing.T) {
	contract := reconciliationContract()
	err := ReconcileRankedPackageCertification("ranger", contract, CatalogRankedStamp{ID: contract.ID, Roles: []string{"archivist"}, Ranked: true})
	require.ErrorContains(t, err, "not certified")

	err = ReconcileRankedPackageCertification("sniper", contract, CatalogRankedStamp{ID: contract.ID, Roles: []string{"ranger"}, Ranked: true, CertificationDigest: "sha256:cert"})
	require.ErrorContains(t, err, "role affinity")
}
