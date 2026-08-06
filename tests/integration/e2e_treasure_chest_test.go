//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file covers docs/integration-coverage-gaps.md item C3: an end-to-end
// `treasure-chest` scenario (scan → doctor → items list/show). Before this
// file, no tests/integration/*.go scenario ever invoked the treasure-chest
// command family, leaving internal/integrity (warning.go) and a slice of
// internal/domain's chest-grade path unreached under the `integration`
// build tag — see
// .analysis/refined/20260805-integration-coverage-mapping/analysis.md.

// installedTreasureChestWorkspace installs and compiles a workspace whose
// active.yaml declares a treasure chest (via writeHappyPathActiveYAML,
// chest id "source", path ".sdd/source") but does not write a matching
// treasure-chests.yaml — deliberately, so `doctor` below exercises its real
// divergence-detection path instead of a synthetic always-green fixture.
func installedTreasureChestWorkspace(t *testing.T) (workspace, strategistDir string) {
	t.Helper()

	workspace = t.TempDir()
	strategistDir = filepath.Join(workspace, ".strategist")

	install := runStrategistCLI(t, workspace, "install", "--target", workspace, "--silent")
	require.Equal(t, 0, install.exitCode, install.output())

	writeHappyPathActiveYAML(t, strategistDir)

	compile := runStrategistCLI(t, workspace, "compile", "--root", strategistDir)
	require.Equal(t, 0, compile.exitCode, compile.output())

	return workspace, strategistDir
}

// TestE2E_CLI_TreasureChestScan_NoMissions covers `treasure-chest scan
// --dry-run` against a workspace with no <base_path>/refined or
// <base_path>/done missions yet — internal/treasure.ScanMissions/
// BuildClusters/BuildGaps on empty input, via the real CLI binary.
func TestE2E_CLI_TreasureChestScan_NoMissions(t *testing.T) {
	t.Parallel()

	workspace, _ := installedTreasureChestWorkspace(t)

	result := runStrategistCLI(t, workspace, "treasure-chest", "scan", "--dry-run")
	require.Equal(t, 0, result.exitCode, result.output())
	assert.Contains(t, result.output(), "0 mission(s) scanned")
}

// TestE2E_CLI_TreasureChestDoctor_DetectsDivergence covers `doctor`'s
// consistency-drift path: active.yaml declares chest "source", but no
// treasure-chests.yaml (governed layer) or matching knowledge.index.yaml
// entry exists for it, so doctor must report a divergence and exit
// non-zero — exercising internal/treasure.LoadActiveChests/LoadGoverned/
// LoadIndexed/MergeChestRows together, none of which any prior
// tests/integration scenario reached.
func TestE2E_CLI_TreasureChestDoctor_DetectsDivergence(t *testing.T) {
	t.Parallel()

	workspace, _ := installedTreasureChestWorkspace(t)

	result := runStrategistCLI(t, workspace, "treasure-chest", "doctor")
	require.NotEqual(t, 0, result.exitCode, result.output())
	assert.Contains(t, result.output(), "source")
	assert.Contains(t, result.output(), "consistency drift")
}

// TestE2E_CLI_TreasureChestItems_ListAndShow seeds a jewels.yaml directly
// under .strategist/ (same fixture shape as
// internal/treasurecli/treasure_chest_items_test.go's oneProposedJewelYAML) and
// runs `items list` then `items show` through the real CLI binary —
// exercising internal/treasure's jewel loaders and
// internal/domain.ValidateJewelKind/Score/Status/Trust's call path, plus
// the items render layer, all previously unreached under the `integration`
// build tag.
func TestE2E_CLI_TreasureChestItems_ListAndShow(t *testing.T) {
	t.Parallel()

	workspace, strategistDir := installedTreasureChestWorkspace(t)

	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "jewels.yaml"), []byte(`
schema_version: "1"
jewels:
  - id: jewel-1
    chest_id: source
    kind: pattern
    statement: "Widgets require explicit teardown."
    source_refs: ["source#widgets"]
    trust: T1
    status: proposed
    reviewed_by: agent
    score:
      value: 55
      reasons: ["recurring across 2 missions"]
`), 0o644))

	list := runStrategistCLI(t, workspace, "treasure-chest", "items", "list")
	require.Equal(t, 0, list.exitCode, list.output())
	assert.Contains(t, list.output(), "jewel-1")

	show := runStrategistCLI(t, workspace, "treasure-chest", "items", "show", "jewel-1")
	require.Equal(t, 0, show.exitCode, show.output())
	assert.Contains(t, show.output(), "jewel-1")
	assert.Contains(t, show.output(), "Widgets require explicit teardown.")
}
