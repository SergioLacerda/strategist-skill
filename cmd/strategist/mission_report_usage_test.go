package main

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	missionadapter "github.com/SergioLacerda/strategist-skill/cmd/strategist/mission"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMissionReportUsageRoot creates a .strategist/-like tree at
// <dir>/.strategist plus a known mission directory at
// <dir>/.analysis/refined/<missionID>, mirroring the layout
// cliutil.ResolveActiveBasePath expects: base_path (".analysis") is
// resolved relative to strategistRoot's parent.
func setupMissionReportUsageRoot(t *testing.T, missionID string) (strategistRoot string) {
	t.Helper()
	dir := t.TempDir()
	strategistRoot = filepath.Join(dir, ".strategist")
	require.NoError(t, os.MkdirAll(strategistRoot, 0o755))
	testutil.MinimalRoot(t, strategistRoot)
	if missionID != "" {
		require.NoError(t, os.MkdirAll(filepath.Join(dir, ".analysis", "refined", missionID), 0o755))
	}
	return strategistRoot
}

// reportUsageArgs builds `report-usage` args for executeMission.
func reportUsageArgs(root, missionID string, tokensIn, tokensOut int64) []string {
	return []string{"report-usage", "--root", root, "--mission-id", missionID,
		"--tokens-in", strconv.FormatInt(tokensIn, 10), "--tokens-out", strconv.FormatInt(tokensOut, 10)}
}

func TestMissionReportUsageCmd_RecordsRealTokenCounts(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-real-mission")
	out, err := executeMission(t, reportUsageArgs(root, "20260830-real-mission", 4096, 2048)...)
	require.NoError(t, err)
	assert.Contains(t, out, "mission_id=20260830-real-mission")
	assert.Contains(t, out, "tokens_in=4096")
	assert.Contains(t, out, "tokens_out=2048")

	data, err := os.ReadFile(filepath.Join(root, "memory", "mission-token-usage.jsonl"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `"mission_id":"20260830-real-mission"`)
	assert.Contains(t, string(data), `"tokens_in":4096`)
	assert.Contains(t, string(data), `"tokens_out":2048`)
	assert.Contains(t, string(data), `"source":"agent_report"`)
	assert.NotContains(t, string(data), `"tokens_in":0`)
}

func TestMissionReportUsageCmd_RejectsUnknownMissionID(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "") // no mission directory created
	_, err := executeMission(t, reportUsageArgs(root, "20260830-does-not-exist", 10, 10)...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown mission_id")
}

func TestMissionReportUsageCmd_RejectsNegativeTokens(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-neg-mission")
	_, err := executeMission(t, reportUsageArgs(root, "20260830-neg-mission", -1, 10)...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tokens-in")
}

func TestMissionReportUsageCmd_RejectsMalformedMissionID(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "")
	_, err := executeMission(t, reportUsageArgs(root, "Not A Valid Id!!", 10, 10)...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "malformed")
}

func TestMissionReportUsageCmd_RequiresMissionID(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "")
	_, err := executeMission(t, reportUsageArgs(root, "", 10, 10)...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--mission-id is required")
}

func TestMissionReportUsageCmd_RequiresTokensInFlag(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-req-mission")
	// --tokens-in deliberately left unset (Changed()==false).
	_, err := executeMission(t, "report-usage", "--root", root, "--mission-id", "20260830-req-mission", "--tokens-out", "10")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--tokens-in is required")
}

// TestRunMissionReportUsage_WithMissionRunSetsSilent covers the wired
// SilenceRun dependency: "if run := cliutil.TelemetryRunFromCmd(cmd); run != nil {
// run.SetSilent() }".
func TestRunMissionReportUsage_WithMissionRunSetsSilent(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-silent-mission")
	cmd := missionadapter.NewReportUsage(missionReportUsageDependencies())
	attachMissionRun(t, cmd)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(reportUsageArgs(root, "20260830-silent-mission", 1, 1)[1:])
	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "recorded")
}

// TestMissionReportUsageCmd_RootResolutionErrorPropagates covers
// runMissionReportUsage's "if err := cliutil.ResolveActiveBasePath(...); err
// != nil { return fmt.Errorf(...) }" branch: --root points at a directory
// with no active.yaml.
func TestMissionReportUsageCmd_RootResolutionErrorPropagates(t *testing.T) {
	_, err := executeMission(t, reportUsageArgs(t.TempDir(), "20260830-no-active-yaml", 1, 1)...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mission report-usage")
}

// TestMissionReportUsageCmd_AppendErrorPropagates covers runMissionReportUsage's
// "if err := telemetry.AppendMissionTokenUsage(...); err != nil { return
// fmt.Errorf(...) }" branch: root/memory exists as a regular file, so
// AppendMissionTokenUsage's MkdirAll for the memory directory fails.
func TestMissionReportUsageCmd_AppendErrorPropagates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ENOTDIR mkdir-error semantics differ on Windows")
	}
	root := setupMissionReportUsageRoot(t, "20260830-blocked-mission")
	require.NoError(t, os.WriteFile(filepath.Join(root, "memory"), []byte("x"), 0o644))
	_, err := executeMission(t, reportUsageArgs(root, "20260830-blocked-mission", 1, 1)...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mission report-usage")
}

// TestMissionReportUsageCmd_WriteOutputErrorPropagates covers
// runMissionReportUsage's final "if _, err := fmt.Fprintf(cmd.OutOrStdout(),
// ...); err != nil { return fmt.Errorf("mission report-usage: write output:
// %w", err) }" branch.
func TestMissionReportUsageCmd_WriteOutputErrorPropagates(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-write-err-mission")
	cmd := newWiredMissionCommand()
	cmd.SetOut(errorWriter{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(reportUsageArgs(root, "20260830-write-err-mission", 5, 5))

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write output")
}

func TestMissionIDKnown(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "refined", "m-known"), 0o755))

	assert.True(t, missionIDKnown(dir, "m-known"))
	assert.False(t, missionIDKnown(dir, "m-unknown"))
}
