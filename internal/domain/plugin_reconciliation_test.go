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
		Provenance:    PackageProvenance{OriginalDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", NormalizedDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", VerificationState: PackageEvidenceDeclared},
	}
}

func reconciliationFacts() CustomPackageFacts {
	return CustomPackageFacts{
		PackageID: "brainstorming", PackageVersion: "1.0.0", Role: "ranger", Slot: "discovery",
		PackageDigest: "sha256:pkg", AdapterDigest: "sha256:adapter",
		RuntimeKind: RankedRuntimeHost, ConnectorID: "local_path", Entrypoint: "discover",
	}
}

func reconciliationLock(t *testing.T) PluginLockFile {
	t.Helper()
	evidence, err := NewCustomBindingEvidence(reconciliationFacts(), 1, "active")
	require.NoError(t, err)
	return PluginLockFile{Bindings: []SlotBinding{evidence.Binding}, Lock: PluginLock{Nodes: evidence.Nodes}}
}

func reconciliationEvidence() CustomPackageEvidence {
	return CustomPackageEvidence{
		PackageID: "brainstorming", Version: "1.0.0", PackageDigest: "sha256:pkg", AdapterDigest: "sha256:adapter",
		Roles: []string{"ranger"}, Slots: []string{"discovery"}, Entrypoints: []string{"discover"},
	}
}

func TestReconcileCustomPackageBindingAcceptsMatchingLock(t *testing.T) {
	require.NoError(t, ReconcileCustomPackageBinding(reconciliationLock(t), "ranger", "discovery", reconciliationEvidence()))
}

func TestReconcileCustomPackageBindingFailsClosedWithoutRepair(t *testing.T) {
	cases := map[string]struct {
		mutateLock     func(*PluginLockFile)
		mutateEvidence func(*CustomPackageEvidence)
		want           string
	}{
		"bare package id":         {func(l *PluginLockFile) { l.Bindings[0].InstalledInstanceID = "brainstorming" }, nil, "not the versioned"},
		"package version":         {nil, func(e *CustomPackageEvidence) { e.Version = "2.0.0" }, "identifies"},
		"package digest":          {nil, func(e *CustomPackageEvidence) { e.PackageDigest = "sha256:other" }, "package digest mismatch"},
		"adapter digest":          {nil, func(e *CustomPackageEvidence) { e.AdapterDigest = "sha256:other" }, "adapter digest mismatch"},
		"role affinity":           {nil, func(e *CustomPackageEvidence) { e.Roles = []string{"archivist"} }, "does not declare Role"},
		"slot affinity":           {nil, func(e *CustomPackageEvidence) { e.Slots = []string{"refinement"} }, "does not declare slot"},
		"entrypoint":              {nil, func(e *CustomPackageEvidence) { e.Entrypoints = []string{"other"} }, "does not declare entrypoint"},
		"missing runtime":         {func(l *PluginLockFile) { l.Bindings[0].RuntimeKind = "" }, nil, "no runtime kind"},
		"missing connector":       {func(l *PluginLockFile) { l.Bindings[0].ConnectorID = "" }, nil, "no connector"},
		"missing generation":      {func(l *PluginLockFile) { l.Bindings[0].Generation = 0 }, nil, "generation"},
		"missing status":          {func(l *PluginLockFile) { l.Bindings[0].Status = "" }, nil, "no status"},
		"missing source digest":   {func(l *PluginLockFile) { l.Bindings[0].SourceDigest = "" }, nil, "no source digest"},
		"unknown role":            {func(l *PluginLockFile) { l.Bindings[0].Role = "bard" }, nil, "belongs to Role"},
		"tampered binding digest": {func(l *PluginLockFile) { l.Bindings[0].BindingDigest = "sha256:forged" }, nil, "tampered"},
		"tampered runtime":        {func(l *PluginLockFile) { l.Bindings[0].RuntimeKind = "executable" }, nil, "tampered"},
		"missing adapter node":    {func(l *PluginLockFile) { l.Lock.Nodes = l.Lock.Nodes[:1] }, nil, "no adapter evidence"},
		"adapter node mismatch":   {func(l *PluginLockFile) { l.Lock.Nodes[1].Digest = "sha256:other" }, nil, "differs between"},
		"missing role node":       {func(l *PluginLockFile) { l.Lock.Nodes = l.Lock.Nodes[:2] }, nil, "no role binding evidence"},
	}
	for name, tc := range cases {
		lock, evidence := reconciliationLock(t), reconciliationEvidence()
		if tc.mutateLock != nil {
			tc.mutateLock(&lock)
		}
		if tc.mutateEvidence != nil {
			tc.mutateEvidence(&evidence)
		}
		before := append([]SlotBinding(nil), lock.Bindings...)

		err := ReconcileCustomPackageBinding(lock, "ranger", "discovery", evidence)

		require.ErrorContains(t, err, tc.want, name)
		require.ErrorContains(t, err, CustomReinstallGuidance, name)
		require.Equal(t, before, lock.Bindings, "%s: the lock is never rewritten", name)
	}
}

func TestReconcileCustomPackageBindingRejectsRankedMirror(t *testing.T) {
	lock := reconciliationLock(t)
	lock.Bindings[0].Mode = SlotBindingModeRanked
	err := ReconcileCustomPackageBinding(lock, "ranger", "discovery", reconciliationEvidence())
	require.ErrorContains(t, err, "not custom")
}

func TestReconcileCustomPackageBindingRejectsAbsentBinding(t *testing.T) {
	lock := reconciliationLock(t)
	lock.Bindings = nil
	err := ReconcileCustomPackageBinding(lock, "ranger", "discovery", reconciliationEvidence())
	require.ErrorContains(t, err, "custom binding missing")
}

func TestNewCustomBindingEvidenceRejectsIncompleteOrInconsistentFacts(t *testing.T) {
	cases := map[string]func(*CustomPackageFacts){
		"version":         func(f *CustomPackageFacts) { f.PackageVersion = "" },
		"digest":          func(f *CustomPackageFacts) { f.PackageDigest = "" },
		"adapter digest":  func(f *CustomPackageFacts) { f.AdapterDigest = "" },
		"runtime":         func(f *CustomPackageFacts) { f.RuntimeKind = "" },
		"connector":       func(f *CustomPackageFacts) { f.ConnectorID = "" },
		"entrypoint":      func(f *CustomPackageFacts) { f.Entrypoint = "" },
		"role/slot owner": func(f *CustomPackageFacts) { f.Role = "archivist" },
		"versioned id":    func(f *CustomPackageFacts) { f.PackageID = "brainstorming@1.0.0" },
	}
	for name, mutate := range cases {
		facts := reconciliationFacts()
		mutate(&facts)

		_, err := NewCustomBindingEvidence(facts, 1, "active")

		require.ErrorContains(t, err, "custom_binding_invalid", name)
	}
	evidence, err := NewCustomBindingEvidence(reconciliationFacts(), 3, "active")
	require.NoError(t, err)
	require.Equal(t, "brainstorming@1.0.0", evidence.Binding.InstalledInstanceID)
	require.Equal(t, "brainstorming", evidence.Nodes[0].ID, "package and adapter nodes are keyed by the bare package id")
	require.Equal(t, "ranger:brainstorming@1.0.0", evidence.Nodes[2].ID)
	require.Equal(t, int64(3), evidence.Binding.Generation)
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
