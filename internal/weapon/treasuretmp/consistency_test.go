package treasuretmp

import (
	"os"
	"path/filepath"
	"testing"

	treasure "github.com/SergioLacerda/strategist-skill/treasure-chest"
	"github.com/stretchr/testify/require"
)

func writeRunbookConsistencyFixture(t *testing.T, root, activePath, manifestPath, indexPath string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("runbooks:\n  canonical_path: "+activePath+"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "treasure-chests.yaml"), []byte("chests:\n  - id: runbooks\n    path: "+manifestPath+"\n    trust:\n      tier: T2\n      reviewed_by: human\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "knowledge.index.yaml"), []byte("sources:\n  - id: runbooks\n    path: "+indexPath+"\n"), 0o644))
}

func TestStatusRejectsRunbookLayerPathDrift(t *testing.T) {
	root := writeIndexFixture(t)
	writeRunbookConsistencyFixture(t, root, ".analysis/runbooks", "docs/runbooks", "docs/runbooks")

	status := Status(root)
	require.False(t, status.Healthy)
	require.Contains(t, status.Limitations, "unhealthy: treasure-chest-tmp: runbook source consistency: active.yaml=.analysis/runbooks, treasure-chests.yaml=docs/runbooks, knowledge.index.yaml=docs/runbooks")
}

func TestSearchRejectsIncompleteRunbookLayer(t *testing.T) {
	root := writeIndexFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "treasure-chests.yaml"), []byte("chests:\n  - id: runbooks\n    path: docs/runbooks\n"), 0o644))

	_, err := Search(Request{Root: root})
	require.ErrorContains(t, err, "runbook source consistency: active.yaml, treasure-chests.yaml, and knowledge.index.yaml must all declare runbooks")
}

func TestStatusAcceptsAlignedRunbookLayers(t *testing.T) {
	root := writeIndexFixture(t)
	writeRunbookConsistencyFixture(t, root, "docs/runbooks", "docs/runbooks", "docs/runbooks")

	status := Status(root)
	require.True(t, status.Healthy, status.Limitations)
}

func TestStatusRejectsMalformedRunbookLayers(t *testing.T) {
	t.Run("active", func(t *testing.T) {
		root := writeIndexFixture(t)
		writeRunbookConsistencyFixture(t, root, "docs/runbooks", "docs/runbooks", "docs/runbooks")
		require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("runbooks: [broken"), 0o644))

		status := Status(root)
		require.False(t, status.Healthy)
		require.Contains(t, status.Limitations[0], "parse active.yaml")
	})

	t.Run("knowledge index", func(t *testing.T) {
		root := writeIndexFixture(t)
		writeRunbookConsistencyFixture(t, root, "docs/runbooks", "docs/runbooks", "docs/runbooks")
		require.NoError(t, os.WriteFile(filepath.Join(root, "knowledge.index.yaml"), []byte("sources: [broken"), 0o644))

		status := Status(root)
		require.False(t, status.Healthy)
		require.Contains(t, status.Limitations[0], "parse knowledge.index.yaml")
	})
}

func TestStatusRejectsRunbookLayerWithoutIndexedDeclaration(t *testing.T) {
	root := writeIndexFixture(t)
	writeRunbookConsistencyFixture(t, root, "docs/runbooks", "docs/runbooks", "docs/runbooks")
	require.NoError(t, os.WriteFile(filepath.Join(root, "knowledge.index.yaml"), []byte("sources:\n  - id: other\n    path: docs/other\n"), 0o644))

	status := Status(root)
	require.False(t, status.Healthy)
	require.Contains(t, status.Limitations[0], "must all declare runbooks")
}

func TestRunbookSourceHelpersHandleDefaultsAndInactiveChest(t *testing.T) {
	require.Equal(t, defaultRunbookSourcePath, normalizeRunbookSourcePath("."))
	require.Equal(t, "docs/other", normalizeRunbookSourcePath("docs/other"))
	require.Equal(t, "docs/runbooks", firstIndexedRunbookPath(t))
	_, declared := governedRunbookPath(map[string]treasure.GovernedChest{
		"runbooks": {Path: "docs/runbooks", Status: "inactive"},
	})
	require.False(t, declared)
}

func firstIndexedRunbookPath(t *testing.T) string {
	t.Helper()
	path, ok := indexedRunbookPath([]indexedRunbookSource{{ID: "runbooks", Path: "docs/runbooks"}})
	require.True(t, ok)
	return path
}
