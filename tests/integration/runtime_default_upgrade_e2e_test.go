//go:build integration

package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_CLI_RuntimeDefaultAutoUpgrade reproduces the 2026-07-19 client
// failure: a `.strategist/` runtime carries an older installed default for a
// normative file. `strategist check` must classify it as auto-repairable
// (not a generic runtime_stale), and a normal `strategist install` (no
// --force) must repair it so a following `strategist check` passes clean.
func TestE2E_CLI_RuntimeDefaultAutoUpgrade(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	strategistDir := filepath.Join(workspace, ".strategist")
	preflightPath := filepath.Join(strategistDir, "contracts", "machine", "preflight.yaml")

	install := runStrategistCLI(t, workspace, "install", "--target", workspace, "--silent")
	require.Equal(t, 0, install.exitCode, install.output())

	currentDefault, err := os.ReadFile(preflightPath)
	require.NoError(t, err)

	// Simulate a workspace installed by an older Strategist version: the
	// on-disk file and its recorded manifest hash both predate the current
	// embedded default.
	oldDefault := []byte("description: Active config from Bootstrap (mode, base_path, roles_config)\n")
	require.NoError(t, os.WriteFile(preflightPath, oldDefault, 0o644))
	rewriteManifestHash(t, strategistDir, "contracts/machine/preflight.yaml", domain.SHA256Hex(oldDefault))

	checkStale := runStrategistCLI(t, workspace, "check", "--root", strategistDir)
	require.NotEqual(t, 0, checkStale.exitCode, checkStale.output())
	assert.Contains(t, checkStale.stderr, "runtime_stale_auto_repairable")
	assert.Contains(t, checkStale.stderr, "contracts/machine/preflight.yaml")

	reinstall := runStrategistCLI(t, workspace, "install", "--target", workspace, "--silent")
	require.Equal(t, 0, reinstall.exitCode, reinstall.output())

	repaired, err := os.ReadFile(preflightPath)
	require.NoError(t, err)
	assert.Equal(t, string(currentDefault), string(repaired))

	checkClean := runStrategistCLI(t, workspace, "check", "--root", strategistDir)
	assert.Equal(t, 0, checkClean.exitCode, checkClean.output())
	assert.NotContains(t, checkClean.stderr, "runtime_stale")
}

func rewriteManifestHash(t *testing.T, strategistDir, relPath, hash string) {
	t.Helper()

	manifestPath := filepath.Join(strategistDir, domain.InstallManifestRelPath)
	data, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	var manifest domain.InstallManifest
	require.NoError(t, json.Unmarshal(data, &manifest))

	found := false
	for i := range manifest.Files {
		if manifest.Files[i].Path == relPath {
			manifest.Files[i].SHA256 = hash
			found = true
			break
		}
	}
	require.True(t, found, "manifest entry for %s not found", relPath)

	updated, err := json.MarshalIndent(manifest, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifestPath, append(updated, '\n'), 0o644))
}
