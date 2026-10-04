package mission

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoveryArtifactPathsStayInsideWorkspace(t *testing.T) {
	workspace := t.TempDir()
	root := filepath.Join(workspace, ".strategist")
	rel, abs, err := DiscoveryArtifactPaths(root, filepath.Join(workspace, ".analysis"), "m1")
	require.NoError(t, err)
	assert.Equal(t, ".analysis/pending/m1-analysis.md", rel)
	assert.Equal(t, filepath.Join(workspace, ".analysis", "pending", "m1-analysis.md"), abs)
	_, _, err = DiscoveryArtifactPaths(root, t.TempDir(), "m1")
	require.ErrorContains(t, err, "escapes workspace")
}

func TestRequireArtifactTargetFreeRefusesUnownedArtifacts(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, RequireArtifactTargetFree(filepath.Join(dir, "absent.md"), "r1"))
	require.ErrorContains(t, RequireArtifactTargetFree(dir, "r1"), "inspect existing discovery artifact")

	plain := filepath.Join(dir, "plain.md")
	require.NoError(t, os.WriteFile(plain, []byte("no frontmatter"), 0o644))
	require.ErrorContains(t, RequireArtifactTargetFree(plain, "r1"), "invocation_artifact_exists")

	done := filepath.Join(dir, "done.md")
	require.NoError(t, os.WriteFile(done, []byte("---\nmission_status: gate_analysis_accepted\n---\nbody\n"), 0o644))
	require.ErrorContains(t, RequireArtifactTargetFree(done, "r1"), "invocation_artifact_exists")
}

func TestWriteDiscoveryArtifactCreatesAndPublishesAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new", "artifact.md")
	require.NoError(t, WriteDiscoveryArtifact(path, []byte("content")))
	assert.FileExists(t, path)

	blocker := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	require.ErrorContains(t, WriteDiscoveryArtifact(filepath.Join(blocker, "child", "artifact.md"), []byte("x")), "create directory")
	target := filepath.Join(dir, "target")
	require.NoError(t, os.MkdirAll(target, 0o755))
	require.ErrorContains(t, WriteDiscoveryArtifact(target, []byte("x")), "write normalized artifact")
}
