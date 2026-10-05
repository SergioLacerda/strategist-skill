package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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

// reportUsageArgs builds `report-usage` args for executeMission, with the
// required phase and role and both counts observed.
func reportUsageArgs(root, missionID string, tokensIn, tokensOut int64) []string {
	return []string{"report-usage", "--root", root, "--mission-id", missionID,
		"--phase", "discovery", "--role", "ranger",
		"--tokens-in", strconv.FormatInt(tokensIn, 10), "--tokens-out", strconv.FormatInt(tokensOut, 10)}
}

func readUsageLedger(t *testing.T, root string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "memory", "mission-usage.jsonl"))
	require.NoError(t, err)
	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var rec map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &rec))
		out = append(out, rec)
	}
	return out
}

func TestMissionReportUsageCmd_RecordsAPhaseAttributedObservation(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-real-mission")
	args := append(reportUsageArgs(root, "20260830-real-mission", 4096, 2048), "--invocation-id", "inv_abc")
	out, err := executeMission(t, args...)
	require.NoError(t, err)
	assert.Contains(t, out, "mission_id=20260830-real-mission")
	assert.Contains(t, out, "phase=DISCOVERY")
	assert.Contains(t, out, "tokens_in=4096")
	assert.Contains(t, out, "quality=reported_unverified")
	assert.Contains(t, out, "recorded")

	recs := readUsageLedger(t, root)
	require.Len(t, recs, 1)
	assert.Equal(t, "DISCOVERY", recs[0]["phase"])
	assert.Equal(t, "ranger", recs[0]["role"])
	assert.Equal(t, "inv_abc", recs[0]["invocation_id"])
	assert.Equal(t, "incremental", recs[0]["record_kind"])
	assert.Equal(t, "reported_unverified", recs[0]["measurement_quality"])
	assert.EqualValues(t, 4096, recs[0]["tokens_input"])
	_, legacy := os.Stat(filepath.Join(root, "memory", "mission-token-usage.jsonl"))
	assert.True(t, os.IsNotExist(legacy), "the legacy ledger is no longer written")
}

func TestMissionReportUsageCmd_RepeatedDeliveryIsCountedOnce(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-dup-mission")
	args := append(reportUsageArgs(root, "20260830-dup-mission", 100, 10), "--invocation-id", "inv_dup")
	_, err := executeMission(t, args...)
	require.NoError(t, err)
	out, err := executeMission(t, args...)
	require.NoError(t, err)
	assert.Contains(t, out, "duplicate")
	assert.Len(t, readUsageLedger(t, root), 1)
}

func TestMissionReportUsageCmd_SameKeyWithDifferentCountsIsAConflict(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-conflict-mission")
	_, err := executeMission(t, append(reportUsageArgs(root, "20260830-conflict-mission", 100, 10), "--invocation-id", "inv_c")...)
	require.NoError(t, err)
	_, err = executeMission(t, append(reportUsageArgs(root, "20260830-conflict-mission", 999, 10), "--invocation-id", "inv_c")...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicts")
}

func TestMissionReportUsageCmd_OmittedCountIsNeverRecordedAsZero(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-partial-mission")
	out, err := executeMission(t, "report-usage", "--root", root, "--mission-id", "20260830-partial-mission",
		"--phase", "execution", "--role", "sniper", "--tokens-out", "40", "--invocation-id", "inv_p")
	require.NoError(t, err)
	assert.Contains(t, out, "tokens_in=unavailable")
	recs := readUsageLedger(t, root)
	assert.Nil(t, recs[0]["tokens_input"])
	assert.EqualValues(t, 40, recs[0]["tokens_output"])
}

func TestMissionReportUsageCmd_UnkeyedRecordIsFlagged(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-unkeyed-mission")
	out, err := executeMission(t, reportUsageArgs(root, "20260830-unkeyed-mission", 5, 5)...)
	require.NoError(t, err)
	assert.Contains(t, out, "dedup_unavailable")
	assert.Equal(t, true, readUsageLedger(t, root)[0]["dedup_unavailable"])
}

func TestMissionReportUsageCmd_CumulativeSnapshotNeedsASession(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-snap-mission")
	_, err := executeMission(t, append(reportUsageArgs(root, "20260830-snap-mission", 5, 5), "--record-kind", "cumulative_snapshot")...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--session-id")

	_, err = executeMission(t, append(reportUsageArgs(root, "20260830-snap-mission", 5, 5),
		"--record-kind", "cumulative_snapshot", "--session-id", "s1", "--scope", "session")...)
	require.NoError(t, err)
}

func TestMissionReportUsageCmd_RefusesAMeasuredClaim(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-fake-mission")
	_, err := executeMission(t, append(reportUsageArgs(root, "20260830-fake-mission", 5, 5), "--quality", "provider_measured")...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stronger")

	_, err = executeMission(t, append(reportUsageArgs(root, "20260830-fake-mission", 5, 5), "--source", "provider_response")...)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not accepted by the CLI")
}

func TestMissionReportUsageCmd_ApprovalGateCarriesNoTokens(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-gate-mission")
	_, err := executeMission(t, "report-usage", "--root", root, "--mission-id", "20260830-gate-mission",
		"--phase", "approval_gate", "--role", "shell", "--tokens-in", "1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Approval Gate carries no tokens")
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

func TestMissionReportUsageCmd_RequiresPhaseRoleAndAtLeastOneCount(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-req-mission")
	_, err := executeMission(t, "report-usage", "--root", root, "--mission-id", "20260830-req-mission", "--tokens-in", "1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--phase is required")
	assert.Contains(t, err.Error(), "--role is required")

	_, err = executeMission(t, "report-usage", "--root", root, "--mission-id", "20260830-req-mission", "--phase", "discovery", "--role", "ranger")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one of --tokens-in or --tokens-out")
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
