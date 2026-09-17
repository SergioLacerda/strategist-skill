package treasure

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadJewels_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, oneJewelYAML)

	got, err := LoadJewels(dir, nil)
	require.NoError(t, err)
	require.Len(t, got["source"], 1)
	assert.Equal(t, "jewel-1", got["source"][0].ID)
}

func TestLoadJewels_MissingFileIsNotError(t *testing.T) {
	t.Parallel()
	got, err := LoadJewels(t.TempDir(), nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestLoadJewels_MissingSourceRefsErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, `
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "x"
    trust: T1
    status: proposed
    reviewed_by: agent
`)
	_, err := LoadJewels(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing source_refs")
}

func TestLoadJewels_TrustExceedsChestTierErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, oneJewelYAML)
	governed := map[string]GovernedChest{
		"source": {ID: "source", Trust: GovernedTrust{Tier: "T2"}},
	}
	_, err := LoadJewels(dir, governed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds parent chest's trust tier")
}

func TestLoadJewels_UnsupportedSchemaVersionErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, `
schema_version: "2"
jewels: []
`)
	_, err := LoadJewels(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported schema_version")
}

func TestLoadJewels_MissingChestIDErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeJewelsFileT(t, dir, `
schema_version: "1"
jewels:
  - id: jewel-1
    kind: pattern
    statement: "x"
    source_refs: ["source#a"]
    trust: T1
    status: proposed
    reviewed_by: agent
`)
	_, err := LoadJewels(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing chest_id")
}

func TestLoadJewels_PartitionedManifestUsed(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "jewels"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "jewels", "source.yaml"), []byte(oneJewelYAML), 0o644))

	got, err := LoadJewels(dir, nil)
	require.NoError(t, err)
	require.Len(t, got["source"], 1)
}

func TestMonolithicJewelManifestPaths_StatError(t *testing.T) {
	skipIfPermissionTestUnsupported(t)
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission denial is not reliable on windows")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_, err := monolithicJewelManifestPaths(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stat jewels.yaml")
}

func TestPartitionedJewelManifestPaths_ReadDirError(t *testing.T) {
	skipIfPermissionTestUnsupported(t)
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission denial is not reliable on windows")
	}
	dir := t.TempDir()
	jewelsDir := filepath.Join(dir, "jewels")
	require.NoError(t, os.MkdirAll(jewelsDir, 0o755))
	require.NoError(t, os.Chmod(jewelsDir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(jewelsDir, 0o755) })

	_, err := partitionedJewelManifestPaths(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read jewels/")
}

func TestIsJewelPartitionFile_SuffixAndDirBranches(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.yaml"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.yml"), nil, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "c.txt"), nil, 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "d.yaml"), 0o755))

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	got := map[string]bool{}
	for _, e := range entries {
		got[e.Name()] = isJewelPartitionFile(e)
	}
	assert.True(t, got["a.yaml"])
	assert.True(t, got["b.yml"])
	assert.False(t, got["c.txt"])
	assert.False(t, got["d.yaml"]) // directory, even with .yaml suffix
}

func TestJewelManifestLabel_RelError(t *testing.T) {
	t.Parallel()
	// filepath.Rel requires both paths to be relative or both absolute;
	// mixing them forces the error branch, falling back to the raw path.
	got := jewelManifestLabel("relative/root", "/absolute/root/jewels.yaml")
	assert.Equal(t, "/absolute/root/jewels.yaml", got)
}

func TestNonDeprecatedJewelCount(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0, NonDeprecatedJewelCount(nil))

	jewels := []Jewel{
		{ID: "j1", Status: "proposed"},
		{ID: "j2", Status: "deprecated"},
		{ID: "j3", Status: "accepted"},
	}
	assert.Equal(t, 2, NonDeprecatedJewelCount(jewels))
}
