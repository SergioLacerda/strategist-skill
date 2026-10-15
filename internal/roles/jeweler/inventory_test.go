package jeweler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildInventoryIsStableAndReadOnly(t *testing.T) {
	root := inventoryTempDir(t)
	writeInventoryFixture(t, root, "legacy/jewels.yaml", "jewel-one\n")
	writeInventoryFixture(t, root, "legacy/index.gz", "index-one\n")
	sources := []InventorySource{
		{Key: "index", Kind: "compiled_index", SourcePath: "legacy/index.gz", Status: "captured", Provenance: "legacy:index"},
		{Key: "jewel", Kind: "jewel", SourcePath: "legacy/jewels.yaml", Status: "accepted", Provenance: "legacy:jewels"},
	}

	first, err := BuildInventory(root, sources)
	require.NoError(t, err)
	second, err := BuildInventory(root, sources)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, []string{"index", "jewel"}, []string{first[0].Key, first[1].Key})
	firstDigest, err := InventoryDigest(first)
	require.NoError(t, err)
	secondDigest, err := InventoryDigest(second)
	require.NoError(t, err)
	require.Equal(t, firstDigest, secondDigest)

	data, err := os.ReadFile(filepath.Join(root, "legacy", "jewels.yaml"))
	require.NoError(t, err)
	require.Equal(t, "jewel-one\n", string(data))
}

func TestBuildInventoryChangesDigestWhenSourceChanges(t *testing.T) {
	root := inventoryTempDir(t)
	path := filepath.Join(root, "jewels.yaml")
	require.NoError(t, os.WriteFile(path, []byte("one\n"), 0o600))
	source := []InventorySource{{Key: "jewels", Kind: "jewels", SourcePath: "jewels.yaml", Status: "captured", Provenance: "legacy:jewels"}}

	first, err := BuildInventory(root, source)
	require.NoError(t, err)
	firstDigest, err := InventoryDigest(first)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte("two\n"), 0o600))
	second, err := BuildInventory(root, source)
	require.NoError(t, err)
	secondDigest, err := InventoryDigest(second)
	require.NoError(t, err)
	require.NotEqual(t, first[0].Digest, second[0].Digest)
	require.NotEqual(t, firstDigest, secondDigest)
}

func TestBuildWorkspaceInventoryCapturesLegacyProjectionInputs(t *testing.T) {
	root := inventoryTempDir(t)
	for _, path := range []string{
		".strategist/treasure-chests.yaml",
		".strategist/jewels.yaml",
		".strategist/potions.yaml",
		".strategist/knowledge.index.yaml",
		".strategist/.compiled/.index.gz",
	} {
		writeInventoryFixture(t, root, path, path+"\n")
	}

	entries, err := BuildWorkspaceInventory(root)
	require.NoError(t, err)
	require.Len(t, entries, 5)
	require.Equal(t, "compiled-index", entries[0].Key)
	require.Equal(t, "treasure-chests", entries[4].Key)
	for _, entry := range entries {
		require.Equal(t, "captured", entry.Status)
		require.Equal(t, "workspace:legacy", entry.Provenance)
		require.Contains(t, entry.Digest, "sha256:")
	}
}

func TestBuildInventoryRejectsUnsafeAndDuplicateSources(t *testing.T) {
	root := inventoryTempDir(t)
	writeInventoryFixture(t, root, "safe.yaml", "safe\n")
	base := InventorySource{Key: "safe", Kind: "jewel", SourcePath: "safe.yaml", Status: "captured", Provenance: "legacy"}

	_, err := BuildInventory(root, []InventorySource{base, base})
	require.ErrorContains(t, err, "duplicate inventory source")
	unsafe := base
	unsafe.SourcePath = "../outside.yaml"
	_, err = BuildInventory(root, []InventorySource{unsafe})
	require.ErrorContains(t, err, "must stay within the workspace")

	link := filepath.Join(root, "link.yaml")
	if err := os.Symlink(filepath.Join(root, "safe.yaml"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	unsafe = base
	unsafe.SourcePath = "link.yaml"
	_, err = BuildInventory(root, []InventorySource{unsafe})
	require.ErrorContains(t, err, "must not be a symlink")
}

func writeInventoryFixture(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

func inventoryTempDir(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp(".", ".jeweler-inventory-")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	return root
}
