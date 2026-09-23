package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMissionPersistenceRoundTripAndOutputModes(t *testing.T) {
	root := t.TempDir()
	engine, status, err := domain.StartMission(domain.MissionStartRequest{MissionID: "coverage-mission"})
	require.NoError(t, err)
	require.NotNil(t, engine)

	require.NoError(t, saveMission(root, status))
	loaded, loadedStatus, err := loadMission(root, status.MissionID)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, status, loadedStatus)

	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	require.NoError(t, writeMissionResult(cmd, false, loadedStatus))
	assert.Contains(t, out.String(), "mission_id=coverage-mission")

	out.Reset()
	require.NoError(t, writeMissionResult(cmd, true, loadedStatus))
	assert.Contains(t, out.String(), `"mission_id":"coverage-mission"`)

	out.Reset()
	require.NoError(t, writeMissionResult(cmd, false, map[string]string{"status": "ok"}))
	assert.Contains(t, out.String(), `"status":"ok"`)
}

func TestMissionPersistenceRejectsMissingAndInvalidState(t *testing.T) {
	root := t.TempDir()
	_, _, err := loadMission(root, "missing-mission")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	path := missionPath(root, "invalid-mission")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("{"), 0o644))
	_, _, err = loadMission(root, "invalid-mission")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid persisted state")

	require.NoError(t, os.WriteFile(path, []byte(`{"mission_id":"bad-mission"}`), 0o644))
	_, _, err = loadMission(root, "invalid-mission")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "restore mission state")
}

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

func TestWriteMissionResultPropagatesWriterErrors(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetOut(errorWriter{})
	status := domain.MissionEngineStatus{MissionID: "writer-error", Phase: "discovery", State: "active"}

	err := writeMissionResult(cmd, false, status)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write mission result")

	err = writeMissionResult(cmd, true, status)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "encode mission result")
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
