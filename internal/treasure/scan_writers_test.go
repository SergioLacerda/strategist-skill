package treasure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- RegenerateDir ---

func TestRegenerateDir_MkdirAllError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	parent := t.TempDir()
	require.NoError(t, os.Chmod(parent, 0o555))
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	err := RegenerateDir(filepath.Join(parent, "child"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create")
}

func TestRegenerateDir_RemoveAllError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	parent := t.TempDir()
	target := filepath.Join(parent, "child")
	require.NoError(t, os.MkdirAll(target, 0o755))
	require.NoError(t, os.Chmod(parent, 0o555))
	t.Cleanup(func() { _ = os.Chmod(parent, 0o755) })

	err := RegenerateDir(target)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remove")
}

func TestRegenerateDir_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "child")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stray"), []byte("x"), 0o644)) // unrelated file, unaffected
	require.NoError(t, RegenerateDir(target))
	info, err := os.Stat(target)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// --- WriteClusterFile / WriteGapFile: write errors ---

func TestWriteClusterFile_WriteError(t *testing.T) {
	t.Parallel()
	// Directory does not exist -> os.WriteFile fails.
	err := WriteClusterFile(filepath.Join(t.TempDir(), "nonexistent"), Cluster{ID: "cluster-x"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write")
}

func TestWriteGapFile_WriteError(t *testing.T) {
	t.Parallel()
	err := WriteGapFile(filepath.Join(t.TempDir(), "nonexistent"), Gap{ID: "sq-001"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write")
}

func TestWriteClusterFile_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, WriteClusterFile(dir, Cluster{ID: "cluster-1", CitedMissions: []string{"m1"}, Tags: []string{"t1"}}))
	raw, err := os.ReadFile(filepath.Join(dir, "cluster-1.md"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "id: cluster-1")
}

func TestWriteScanOutputs_ClusterFileWriteErrorAfterRegenerate(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	clustersDir := filepath.Join(parent, "clusters")
	gapsDir := filepath.Join(parent, "gaps")
	// A cluster ID containing a path separator makes the per-file WriteFile
	// fail (intermediate dir doesn't exist) even though RegenerateDir itself
	// succeeds — exercising WriteScanOutputs' propagation from
	// writeClusterFiles, not RegenerateDir's own error branch.
	badCluster := Cluster{ID: "sub/cluster-x"}

	err := WriteScanOutputs(clustersDir, []Cluster{badCluster}, gapsDir, nil)
	require.Error(t, err)
}

func TestWriteScanOutputs_GapFileWriteErrorAfterRegenerate(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	clustersDir := filepath.Join(parent, "clusters")
	gapsDir := filepath.Join(parent, "gaps")
	badGap := Gap{ID: "sub/gap-x", Status: "sq_pending"}

	err := WriteScanOutputs(clustersDir, nil, gapsDir, []Gap{badGap})
	require.Error(t, err)
}

func TestWriteGapFile_WithDependencies(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, WriteGapFile(dir, Gap{ID: "sq-001", Dependencies: []string{"sq-000"}, Status: "sq_pending"}))
	raw, err := os.ReadFile(filepath.Join(dir, "sq-001.md"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "dependencies: [sq-000]")
}

// --- WriteScanOutputs: regeneration/write failures ---

func TestWriteScanOutputs_GapsDirError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	parent := t.TempDir()
	clustersDir := filepath.Join(parent, "clusters")
	gapsDir := filepath.Join(parent, "locked", "gaps")
	require.NoError(t, os.MkdirAll(filepath.Join(parent, "locked"), 0o555))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(parent, "locked"), 0o755) })

	err := WriteScanOutputs(clustersDir, nil, gapsDir, []Gap{{ID: "sq-001", Status: "sq_pending"}})
	require.Error(t, err)
}

func TestWriteScanOutputs_ClusterWriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	parent := t.TempDir()
	clustersDir := filepath.Join(parent, "locked", "clusters")
	require.NoError(t, os.MkdirAll(filepath.Join(parent, "locked"), 0o555))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(parent, "locked"), 0o755) })

	err := WriteScanOutputs(clustersDir, []Cluster{{ID: "cluster-x"}}, filepath.Join(parent, "gaps"), nil)
	require.Error(t, err)
}
