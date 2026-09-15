package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writePluginLockFixture(t *testing.T, root, refinementID string) {
	t.Helper()
	content := "schema_version: strategist-plugin-lock-file/v1\n" +
		"bindings:\n" +
		"  - slot: discovery\n" +
		"    installed_instance_id: ranger\n" +
		"  - slot: refinement\n" +
		"    installed_instance_id: " + refinementID + "\n"
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte(content), 0o644))
}

func TestCheckPluginLockParityReportsDivergence(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writePluginLockFixture(t, root, "archivist")

	errs := checkPluginLockParity(root, map[string]string{
		"discovery":  "ranger",
		"refinement": "openspec-propose",
		"execution":  "sniper",
	})

	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "slot refinement")
	assert.Contains(t, errs[0], "openspec-propose")
	assert.Contains(t, errs[0], "archivist")
}

func TestCheckPluginLockParityAcceptsMatchingBindings(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writePluginLockFixture(t, root, "archivist")

	errs := checkPluginLockParity(root, map[string]string{
		"discovery":  "ranger",
		"refinement": "archivist",
		"execution":  "sniper",
	})

	assert.Empty(t, errs)
}
