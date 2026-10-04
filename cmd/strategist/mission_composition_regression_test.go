package main

// Composition-boundary regression coverage.

import (
	"path/filepath"
	"testing"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeOpenSpecValidationAndDefaultPaths(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "")
	basePath, runtimeRoot, pending, err := resolveNormalizePaths(missionadapter.NormalizeOptions{
		Root: root, MissionID: "normalize-mission",
	})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(filepath.Dir(root), ".analysis"), basePath)
	assert.Equal(t, filepath.Join(root, "openspec"), runtimeRoot)
	assert.Equal(t, filepath.Join(basePath, "pending", "normalize-mission-analysis.md"), pending)

	err = missionadapter.RunNormalizeOpenSpec(&cobra.Command{}, missionNormalizeDependencies(), missionadapter.NormalizeOptions{MissionID: "Invalid Mission"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mission normalize-openspec: --mission-id \"Invalid Mission\" is malformed")
}

func TestMissionStartAndStatusCommandsPersistState(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "")

	startOut, err := executeMission(t, "start", "--root", root, "--mission-id", "command-mission", "--json")
	require.NoError(t, err)
	assert.Contains(t, startOut, `"mission_id":"command-mission"`)

	statusOut, err := executeMission(t, "status", "--root", root, "--mission-id", "command-mission", "--json")
	require.NoError(t, err)
	assert.Contains(t, statusOut, `"mission_id":"command-mission"`)

	_, err = executeMission(t, "start", "--root", root, "--mission-id", "command-mission")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}
