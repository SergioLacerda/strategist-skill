package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/install"
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

func TestMissionLifecycleFlagHelpersAndValidation(t *testing.T) {
	require.Error(t, requireMissionID("Invalid Mission"))
	require.NoError(t, requireMissionID("valid-mission-123"))

	cmd := &cobra.Command{}
	cmd.Flags().StringSlice("refs", []string{"one"}, "")
	cmd.Flags().Int("limit", 7, "")
	assert.Equal(t, []string{"one"}, stringSliceFlag(cmd, "refs"))
	assert.Equal(t, 7, intFlag(cmd, "limit", 1))
	assert.Nil(t, stringSliceFlag(cmd, "missing"))
	assert.Equal(t, 1, intFlag(cmd, "missing", 1))

	flags := missionStartCmd.Flags()
	require.NoError(t, flags.Set("mission-id", "flag-mission"))
	require.NoError(t, flags.Set("json", "true"))
	t.Cleanup(func() {
		_ = flags.Set("mission-id", "")
		_ = flags.Set("json", "false")
	})
	opts := missionLifecycleOptionsFrom(missionStartCmd)
	assert.Equal(t, "flag-mission", opts.MissionID)
	assert.True(t, opts.JSON)
}

func TestNormalizeOpenSpecValidationAndDefaultPaths(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "")
	basePath, runtimeRoot, pending, err := resolveNormalizePaths(missionNormalizeOpenSpecOptions{
		Root: root, MissionID: "normalize-mission",
	})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(filepath.Dir(root), ".analysis"), basePath)
	assert.Equal(t, filepath.Join(root, "openspec"), runtimeRoot)
	assert.Equal(t, filepath.Join(basePath, "pending", "normalize-mission-analysis.md"), pending)

	err = runMissionNormalizeOpenSpec(&cobra.Command{}, missionNormalizeOpenSpecOptions{MissionID: "Invalid Mission"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lowercase letters")
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

func TestPrintUpgradePlanListsEveryUpgradeState(t *testing.T) {
	plan := install.UpgradePlan{Entries: []install.UpgradePlanEntry{
		{Path: "managed.md", State: domain.UpgradeManaged},
		{Path: "missing.md", State: domain.UpgradeMissing},
		{Path: "auto.md", State: domain.UpgradeAutoUpgrade},
		{Path: "custom.md", State: domain.UpgradeCustomized},
		{Path: "orphan.md", State: domain.UpgradeOrphaned},
	}}
	var out bytes.Buffer
	require.NoError(t, printUpgradePlan(&out, plan, false))
	assert.Contains(t, out.String(), "missing (will write): 1")
	assert.Contains(t, out.String(), "customized (preserved): 1")
	assert.Contains(t, out.String(), "orphaned (not deleted — review manually): 1")

	out.Reset()
	require.NoError(t, printUpgradePlan(&out, plan, true))
	assert.Contains(t, out.String(), "customized (will OVERWRITE — --force): 1")
}

func TestMissionStartAndStatusCommandsPersistState(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "")
	startFlags := missionStartCmd.Flags()
	statusFlags := missionStatusCmd.Flags()
	setFlags := func(flags interface {
		Set(string, string) error
	}) {
		require.NoError(t, flags.Set(flagRoot, root))
		require.NoError(t, flags.Set("mission-id", "command-mission"))
		require.NoError(t, flags.Set("json", "true"))
	}
	setFlags(startFlags)
	setFlags(statusFlags)
	t.Cleanup(func() {
		resetFlags := func(flags interface {
			Set(string, string) error
		}) {
			_ = flags.Set(flagRoot, "")
			_ = flags.Set("mission-id", "")
			_ = flags.Set("json", "false")
		}
		resetFlags(startFlags)
		resetFlags(statusFlags)
	})

	startOut := captureStdout(t, func() {
		require.NoError(t, missionStartCmd.RunE(missionStartCmd, nil))
	})
	assert.Contains(t, startOut, `"mission_id":"command-mission"`)

	statusOut := captureStdout(t, func() {
		require.NoError(t, missionStatusCmd.RunE(missionStatusCmd, nil))
	})
	assert.Contains(t, statusOut, `"mission_id":"command-mission"`)
}
