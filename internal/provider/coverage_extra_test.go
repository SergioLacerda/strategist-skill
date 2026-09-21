package provider

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestValidateBoundaryReasonCodes(t *testing.T) {
	file := filepath.Join(t.TempDir(), "source")
	require.NoError(t, os.WriteFile(file, []byte("not a directory"), 0o644))
	cases := []struct {
		name, input, want string
	}{
		{name: "required", input: "", want: "source_required"},
		{name: "not directory", input: file, want: "source_not_directory"},
		{name: "missing manifest", input: t.TempDir(), want: "manifest_missing"},
		{name: "git source", input: "git@github.com:example/provider.git", want: "remote_source_deferred"},
		{name: "file uri input", input: "file:///tmp/provider", want: "source_unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			report, err := Validate(tc.input, "")
			require.Error(t, err)
			require.Contains(t, reasonCodes(report.Reasons), tc.want)
		})
	}
}

func TestValidateContractReasonCodes(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(string)
		want   string
	}{
		{name: "bad package yaml", mutate: func(dir string) { writeFixtureFile(t, dir, packageManifestName, "schema_version: [broken\n") }, want: "package_contract_invalid"},
		{name: "bad adapter yaml", mutate: func(dir string) { writeFixtureFile(t, dir, adapterManifestName, "schema_version: [broken\n") }, want: "adapter_contract_invalid"},
		{name: "identity mismatch", mutate: func(dir string) {
			replaceFixtureText(t, filepath.Join(dir, packageManifestName), "id: fixture-provider", "id: other-provider")
		}, want: "identity_mismatch"},
		{name: "provenance scheme", mutate: func(dir string) {
			replaceFixtureText(t, filepath.Join(dir, packageManifestName), "artifact_uri: local://minimal-provider", "artifact_uri: https://example.invalid/provider")
		}, want: "provenance_not_local"},
		{name: "slot unsupported", mutate: func(dir string) {
			replaceFixtureText(t, filepath.Join(dir, adapterManifestName), "  - refinement", "  - execution")
		}, want: "slot_unsupported"},
		{name: "role mismatch", mutate: func(dir string) {
			replaceFixtureText(t, filepath.Join(dir, adapterManifestName), "  - archivist", "  - ranger")
		}, want: "role_slot_mismatch"},
		{name: "legacy conflict", mutate: func(dir string) { writeFixtureFile(t, dir, legacyManifestName, "id: other-provider\n") }, want: "legacy_authority_conflict"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := copyFixture(t)
			tc.mutate(dir)
			report, err := Validate(dir, "refinement")
			require.Error(t, err)
			require.Contains(t, reasonCodes(report.Reasons), tc.want)
		})
	}
}

func TestArtifactAndStateHelpersCoverFailureBoundaries(t *testing.T) {
	require.NotEmpty(t, digestSourceFiles(map[string][]byte{"b": []byte("b"), "a": []byte("a")}))
	root := t.TempDir()
	lock, exists, err := readLock(root)
	require.NoError(t, err)
	require.False(t, exists)
	require.Equal(t, domain.PluginLockFileSchemaVersion, lock.SchemaVersion)
	require.NoError(t, writeLock(root, lock))
	_, exists, err = readLock(root)
	require.NoError(t, err)
	require.True(t, exists)
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("bad: ["), 0o644))
	_, _, err = readLock(root)
	require.Error(t, err)

	txFile, err := readTransactions(root)
	require.NoError(t, err)
	tx := domain.PluginTransaction{ID: "tx", State: "complete"}
	require.NoError(t, appendTransaction(root, txFile, tx))
	tx.State = "rolled_back"
	require.NoError(t, appendTransaction(root, txFile, tx))
	require.NoError(t, os.WriteFile(filepath.Join(root, transactionFileName), []byte("bad: ["), 0o644))
	_, err = readTransactions(root)
	require.Error(t, err)

	blocker := filepath.Join(root, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))
	require.Error(t, atomicWrite(filepath.Join(blocker, "child"), []byte("x")))
	require.Error(t, compileWorkspace(filepath.Join(root, "missing")))
	require.Error(t, validateProviderID("../bad"))
	require.Equal(t, "provider-add-refinement-id", transactionID("id", "refinement"))
}

func TestLifecycleAndLockHelpersCoverExistingState(t *testing.T) {
	oldInstance := domain.InstalledInstance{ID: "old", State: "active", LastKnownGood: true}
	old := domain.PluginLockFile{
		Inventory: domain.PluginInventory{Instances: []domain.InstalledInstance{oldInstance}},
		Bindings:  []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: "old", Generation: 4, Status: "active"}},
		Lock:      domain.PluginLock{Nodes: []domain.PluginLockNode{{ID: "fixture-provider", Kind: string(domain.PluginResourcePackage), Digest: "old"}}},
	}
	candidate := old
	candidate.Inventory.Instances = append(candidate.Inventory.Instances, domain.InstalledInstance{ID: "new"})
	candidate.Bindings = append([]domain.SlotBinding(nil), old.Bindings...)
	candidate.Bindings = replaceBinding(candidate.Bindings, domain.SlotBinding{Slot: "refinement", InstalledInstanceID: "new"})
	require.Equal(t, int64(4), currentGeneration(old.Bindings, "refinement"))
	require.Equal(t, int64(0), currentGeneration(old.Bindings, "execution"))
	inventory, bindings, err := activateThroughLifecycle(old, candidate, "new", "refinement")
	require.NoError(t, err)
	require.Equal(t, "new", bindings[0].InstalledInstanceID)
	require.Equal(t, int64(5), bindings[0].Generation)
	require.True(t, inventory.Instances[1].LastKnownGood)

	require.Equal(t, []domain.InstalledInstance{{ID: "new"}}, replaceInstance(nil, domain.InstalledInstance{ID: "new"}))
	require.Equal(t, "updated", replaceInstance([]domain.InstalledInstance{{ID: "new"}}, domain.InstalledInstance{ID: "new", State: "updated"})[0].State)
	lock := replaceLockNodes(old.Lock, "fixture-provider", "new-package", "new-adapter")
	require.Len(t, lock.Nodes, 2)
	require.Equal(t, int64(1), nextGeneration(domain.PluginLockFile{}, "refinement"))
	require.Equal(t, "new", findInstance(candidate.Inventory, "new").ID)
}

func TestCopyAndRollbackBoundaries(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	require.NoError(t, copySource(fixturePath(t), target))
	require.NoError(t, copySource(fixturePath(t), target))
	if err := os.Symlink(filepath.Join(target, packageManifestName), filepath.Join(dir, "symlink")); err == nil {
		require.Error(t, copySource(dir, filepath.Join(dir, "copy")))
	}
	root := t.TempDir()
	lock := domain.PluginLockFile{SchemaVersion: domain.PluginLockFileSchemaVersion}
	tx := domain.PluginTransaction{ID: "rollback", State: "staging"}
	result, err := rollbackAdd(root, lock, false, transactionFile{SchemaVersion: transactionSchemaVersion}, tx, "test", os.ErrInvalid)
	require.Error(t, err)
	require.Equal(t, "rolled_back", result.TransactionState)
	require.NoError(t, writeLock(root, lock))
	_, err = rollbackCandidate(root, "", false, lock, true, transactionFile{SchemaVersion: transactionSchemaVersion}, tx, "test", os.ErrInvalid)
	require.Error(t, err)
}

func TestAddUsesExistingLifecycleBinding(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, copySource(filepath.Join("..", "embed", "defaults"), root))
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("mode: epic\nbase_path: .analysis\nknowledge_index_path: knowledge.index.yaml\nslots:\n  discovery: ranger\n  refinement: archivist\n  execution: sniper\n"), 0o644))
	old := domain.PluginLockFile{
		Inventory: domain.PluginInventory{SchemaVersion: "strategist-plugin-inventory/v1", Instances: []domain.InstalledInstance{{ID: "old", State: "active", LastKnownGood: true}}},
		Bindings:  []domain.SlotBinding{{Slot: "refinement", InstalledInstanceID: "old", Generation: 7, Status: "active"}},
	}
	require.NoError(t, writeLock(root, old))
	result, err := Add(root, fixturePath(t), "refinement")
	require.NoError(t, err)
	require.Equal(t, int64(8), result.BindingGeneration)
}

func writeFixtureFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func replaceFixtureText(t *testing.T, path, old, replacement string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(raw), old)
	require.NoError(t, os.WriteFile(path, []byte(strings.Replace(string(raw), old, replacement, 1)), 0o644))
}
