package treasure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanMissionsInDir_ReadDirPermissionError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("permission tests do not apply when running as root")
	}
	dir := t.TempDir()
	sub := filepath.Join(dir, "refined")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	require.NoError(t, os.Chmod(sub, 0o000))
	t.Cleanup(func() { _ = os.Chmod(sub, 0o755) })

	_, err := ScanMissionsInDir(sub)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read")
}

func TestScanMissionsInDir_PropagatesParseError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	missionDir := filepath.Join(dir, "mission-a")
	require.NoError(t, os.MkdirAll(missionDir, 0o755))
	// side_quests_approved: block with invalid YAML underneath.
	require.NoError(t, os.WriteFile(filepath.Join(missionDir, "tasks.md"),
		[]byte("side_quests_approved:\n  : not: valID:\n"), 0o644))

	_, err := ScanMissionsInDir(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse side_quests_approved")
}
