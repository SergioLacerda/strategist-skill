package treasure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadChestDocumentsReportNonMissingReadErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	require.NoError(t, os.Mkdir(filepath.Join(dir, "active.yaml"), 0o755))
	_, err := LoadActiveChests(dir)
	require.ErrorContains(t, err, "read active.yaml")

	require.NoError(t, os.Mkdir(filepath.Join(dir, "knowledge.index.yaml"), 0o755))
	_, err = LoadIndexed(dir)
	require.ErrorContains(t, err, "read knowledge.index.yaml")

	require.NoError(t, os.Mkdir(filepath.Join(dir, "treasure-chests.yaml"), 0o755))
	_, err = LoadGoverned(dir)
	require.ErrorContains(t, err, "read treasure-chests.yaml")
}
