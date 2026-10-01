package provider

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func addWorkspace(t *testing.T) string {
	t.Helper()
	return stockRankedWorkspace(t)
}

func readLockFile(t *testing.T, root string) domain.PluginLockFile {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "plugins.lock"))
	require.NoError(t, err)
	var lock domain.PluginLockFile
	require.NoError(t, yaml.Unmarshal(raw, &lock))
	return lock
}

func TestAddPersistsACompleteCustomBindingWithMatchingEvidence(t *testing.T) {
	root := addWorkspace(t)

	result, err := Add(root, analysisFixturePath(t), "refinement")

	require.NoError(t, err)
	lock := readLockFile(t, root)
	binding, err := domain.SingleLockBindingForSlot(lock, "refinement")
	require.NoError(t, err)
	require.Equal(t, result.InstanceID, binding.InstalledInstanceID)
	require.Equal(t, "archivist", binding.Role)
	require.Equal(t, "1.0.0", binding.WeaponVersion)
	require.Equal(t, result.Report.AdapterDigest, binding.WeaponDigest)
	require.Equal(t, result.Report.PackageDigest, binding.SourceDigest)
	require.Equal(t, domain.RankedRuntimeHost, binding.RuntimeKind)
	require.Equal(t, "local_path", binding.ConnectorID)
	require.Equal(t, "host.prompt", binding.Entrypoint)
	require.NoError(t, domain.ValidateCustomBinding(lock, binding, "archivist", "refinement"))
	require.NotEmpty(t, lock.NodeDigest("archivist:"+result.InstanceID, domain.CustomRoleBindingNodeKind))
}

// A source that cannot yield a complete binding fails the whole transaction:
// the previous package directory, lock and transaction journal are restored
// byte for byte and no instance directory is left behind.
func TestAddRollsBackWhenACompleteBindingCannotBeDerived(t *testing.T) {
	root := addWorkspace(t)
	_, err := Add(root, analysisFixturePath(t), "refinement")
	require.NoError(t, err)
	lockBefore, err := os.ReadFile(filepath.Join(root, "plugins.lock"))
	require.NoError(t, err)
	txBefore, err := os.ReadFile(filepath.Join(root, transactionFileName))
	require.NoError(t, err)

	source, reasons := loadSource(analysisFixturePath(t))
	require.Empty(t, reasons)
	source.Package.Version = "2.0.0"
	source.Adapter.Entrypoints = nil // no entrypoint: the binding cannot be derived
	report := Report{Validated: true, AdapterDigest: "sha256:adapter"}
	txn, err := newAddTxn(root, source, report, "refinement")
	require.NoError(t, err)

	_, err = txn.run()

	require.ErrorContains(t, err, "custom_binding_invalid")
	lockAfter, readErr := os.ReadFile(filepath.Join(root, "plugins.lock"))
	require.NoError(t, readErr)
	require.Equal(t, string(lockBefore), string(lockAfter))
	require.NoDirExists(t, filepath.Join(root, providerDirName, "fixture-provider@2.0.0"))
	txAfter, readErr := os.ReadFile(filepath.Join(root, transactionFileName))
	require.NoError(t, readErr)
	require.NotEqual(t, string(txBefore), string(txAfter), "the failed attempt is journaled")
	require.Contains(t, string(txAfter), "state: rolled_back")
}

// A binding written before the complete contract existed is not "already
// bound": adding the same package again replaces it with the complete one.
func TestReAddingTheSamePackageReplacesAnIncompleteBinding(t *testing.T) {
	root := addWorkspace(t)
	result, err := Add(root, analysisFixturePath(t), "refinement")
	require.NoError(t, err)
	lock := readLockFile(t, root)
	for i := range lock.Bindings {
		if lock.Bindings[i].Slot == "refinement" {
			lock.Bindings[i] = domain.SlotBinding{SchemaVersion: "strategist-plugin-binding/v1", Slot: "refinement", InstalledInstanceID: result.InstanceID, Generation: 1, Status: "active", Mode: domain.SlotBindingModeCustom}
		}
	}
	raw, err := yaml.Marshal(lock)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), raw, 0o644))

	_, err = Add(root, analysisFixturePath(t), "refinement")

	require.NoError(t, err)
	repaired := readLockFile(t, root)
	binding, bindErr := domain.SingleLockBindingForSlot(repaired, "refinement")
	require.NoError(t, bindErr)
	require.NoError(t, domain.ValidateCustomBinding(repaired, binding, "archivist", "refinement"))
}

func TestAddingTheSameCompletePackageTwiceIsANoOp(t *testing.T) {
	root := addWorkspace(t)
	first, err := Add(root, analysisFixturePath(t), "refinement")
	require.NoError(t, err)
	before, err := os.ReadFile(filepath.Join(root, "plugins.lock"))
	require.NoError(t, err)

	second, err := Add(root, analysisFixturePath(t), "refinement")

	require.NoError(t, err)
	require.Equal(t, first.BindingGeneration, second.BindingGeneration)
	after, readErr := os.ReadFile(filepath.Join(root, "plugins.lock"))
	require.NoError(t, readErr)
	require.Equal(t, string(before), string(after))
}
