package eval

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/cliutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testResolveRoot is a hermetic stand-in for adapter_deps.go's
// resolveEvalActionRoot: it treats the explicit --root value as
// strategistRoot and its parent as projectRoot, without walking the real
// filesystem for an auto-discovered .strategist/.
func testResolveRoot(_ *cobra.Command, action, explicitRoot string) (strategistRoot, projectRoot string, err error) {
	if explicitRoot == "" {
		return "", "", fmt.Errorf("eval %s: root required in test", action)
	}
	return explicitRoot, filepath.Dir(explicitRoot), nil
}

func testHarvestDeps() HarvestDependencies {
	return HarvestDependencies{
		RootFlag:        "root",
		ResolveRoot:     testResolveRoot,
		ResolveBasePath: cliutil.ResolveActiveBasePath,
		Select:          SelectHarvestMissionIDs,
		PrintWarnings:   PrintHarvestWarnings,
		ParseInclude:    ParseHarvestInclude,
		Harvest:         HarvestMissions,
		SilenceRun:      noopSilenceRun,
	}
}

func setEvalHarvestFlags(t *testing.T, cmd *cobra.Command, root string, all bool, include string) {
	t.Helper()
	require.NoError(t, cmd.Flags().Set("root", root))
	require.NoError(t, cmd.Flags().Set("all", boolFlagString(all)))
	require.NoError(t, cmd.Flags().Set("include", include))
}

func boolFlagString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// TestRunHarvest_InvokesSilenceRun covers RunHarvest's own
// "if deps.SilenceRun != nil { deps.SilenceRun(cmd) }" branch. The telemetry
// MissionRun.SetSilent() logic itself lives in adapter_deps.go's SilenceRun
// closure (package main), outside this package's scope.
func TestRunHarvest_InvokesSilenceRun(t *testing.T) {
	dir, base := scanTestRoot(t)
	writeMissionTasks(t, base, "refined", "m-silent", "## Task 1 — Example\n")
	writeMissionAnalysis(t, base, "refined", "m-silent", "# Analysis\n")

	deps := testHarvestDeps()
	called := false
	deps.SilenceRun = func(*cobra.Command) { called = true }
	cmd := NewHarvest(deps)
	setEvalHarvestFlags(t, cmd, dir, false, "")

	require.NoError(t, cmd.RunE(cmd, []string{"m-silent"}))
	assert.True(t, called)
}

func TestHarvestCmd_ResolveBasePathErrorPropagates(t *testing.T) {
	root := filepath.Join(t.TempDir(), "no-active-yaml")
	require.NoError(t, os.MkdirAll(root, 0o755))

	cmd := NewHarvest(testHarvestDeps())
	setEvalHarvestFlags(t, cmd, root, false, "")

	err := cmd.RunE(cmd, []string{"m-1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "eval harvest")
}

func TestHarvestCmd_SelectMissionIDsErrorPropagates(t *testing.T) {
	dir, _ := scanTestRoot(t)

	cmd := NewHarvest(testHarvestDeps())
	setEvalHarvestFlags(t, cmd, dir, false, "")

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
}

func TestHarvestCmd_ParseIncludeErrorPropagates(t *testing.T) {
	dir, base := scanTestRoot(t)
	writeMissionTasks(t, base, "refined", "m-bogus", "## Task 1 — Example\n")
	writeMissionAnalysis(t, base, "refined", "m-bogus", "# Analysis\n")

	cmd := NewHarvest(testHarvestDeps())
	setEvalHarvestFlags(t, cmd, dir, false, "bogus")

	err := cmd.RunE(cmd, []string{"m-bogus"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
}

func TestHarvestCmd_HarvestMissionsErrorPropagates(t *testing.T) {
	dir, _ := scanTestRoot(t)

	cmd := NewHarvest(testHarvestDeps())
	setEvalHarvestFlags(t, cmd, dir, false, "")

	err := cmd.RunE(cmd, []string{"does-not-exist"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "eval harvest does-not-exist")
}

func TestHarvestCmd_EndToEnd(t *testing.T) {
	dir, base := scanTestRoot(t)
	writeMissionTasks(t, base, "refined", "20260804-example", "## Task 1 — Example\n")
	writeMissionAnalysis(t, base, "refined", "20260804-example", "# Analysis\n\nreal fixture content\n")

	cmd := NewHarvest(testHarvestDeps())
	var out bytes.Buffer
	cmd.SetOut(&out)
	setEvalHarvestFlags(t, cmd, dir, false, "")

	require.NoError(t, cmd.RunE(cmd, []string{"20260804-example"}))
	assert.Contains(t, out.String(), "1 mission(s), 1 fixture file(s) written")

	projectRoot := filepath.Dir(dir)
	got, err := os.ReadFile(filepath.Join(projectRoot, "tests", "evals", "regression", "20260804-example", "analysis.md"))
	require.NoError(t, err)
	assert.Equal(t, "# Analysis\n\nreal fixture content\n", string(got))
}
