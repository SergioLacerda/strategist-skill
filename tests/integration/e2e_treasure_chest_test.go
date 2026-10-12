//go:build integration

package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_CLI_TreasureChestNamespaceRemoved protects the Sprint 1 boundary:
// Strategist no longer exposes the former internal Treasure Chest command
// family. The retained implementation is consumed through the admitted
// TREASURE-CHEST-TMP Weapon and JEWELER instead.
func TestE2E_CLI_TreasureChestNamespaceRemoved(t *testing.T) {
	t.Parallel()

	workspace, _ := installedTelemetryWorkspace(t)
	result := runStrategistCLI(t, workspace, "treasure-chest", "--help")
	require.Equal(t, 1, result.exitCode, result.output())
	assert.Contains(t, result.output(), `unknown command "treasure-chest"`)
}
