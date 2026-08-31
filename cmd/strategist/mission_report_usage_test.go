package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

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

func setMissionReportUsageFlags(t *testing.T, root, missionID string, tokensIn, tokensOut int64) {
	t.Helper()
	require.NoError(t, missionReportUsageCmd.Flags().Set(flagRoot, root))
	require.NoError(t, missionReportUsageCmd.Flags().Set("mission-id", missionID))
	require.NoError(t, missionReportUsageCmd.Flags().Set("tokens-in", strconv.FormatInt(tokensIn, 10)))
	require.NoError(t, missionReportUsageCmd.Flags().Set("tokens-out", strconv.FormatInt(tokensOut, 10)))
}

// resetMissionReportUsageFlags restores every flag to its zero/default
// state and clears Changed(), since missionReportUsageCmd is a package-level
// singleton shared across tests (same pattern as
// resetHandoffVerifyFlags/handoff_verify_test.go).
func resetMissionReportUsageFlags(t *testing.T) {
	t.Helper()
	require.NoError(t, missionReportUsageCmd.Flags().Set(flagRoot, ""))
	require.NoError(t, missionReportUsageCmd.Flags().Set("mission-id", ""))
	require.NoError(t, missionReportUsageCmd.Flags().Set("tokens-in", "0"))
	require.NoError(t, missionReportUsageCmd.Flags().Set("tokens-out", "0"))
	missionReportUsageCmd.Flags().Lookup(flagRoot).Changed = false
	missionReportUsageCmd.Flags().Lookup("mission-id").Changed = false
	missionReportUsageCmd.Flags().Lookup("tokens-in").Changed = false
	missionReportUsageCmd.Flags().Lookup("tokens-out").Changed = false
}

func TestMissionReportUsageCmd_RecordsRealTokenCounts(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-real-mission")
	setMissionReportUsageFlags(t, root, "20260830-real-mission", 4096, 2048)
	t.Cleanup(func() { resetMissionReportUsageFlags(t) })

	out := captureStdout(t, func() {
		require.NoError(t, missionReportUsageCmd.RunE(missionReportUsageCmd, nil))
	})
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
	setMissionReportUsageFlags(t, root, "20260830-does-not-exist", 10, 10)
	t.Cleanup(func() { resetMissionReportUsageFlags(t) })

	err := missionReportUsageCmd.RunE(missionReportUsageCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown mission_id")
}

func TestMissionReportUsageCmd_RejectsNegativeTokens(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-neg-mission")
	setMissionReportUsageFlags(t, root, "20260830-neg-mission", -1, 10)
	t.Cleanup(func() { resetMissionReportUsageFlags(t) })

	err := missionReportUsageCmd.RunE(missionReportUsageCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tokens-in")
}

func TestMissionReportUsageCmd_RejectsMalformedMissionID(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "")
	setMissionReportUsageFlags(t, root, "Not A Valid Id!!", 10, 10)
	t.Cleanup(func() { resetMissionReportUsageFlags(t) })

	err := missionReportUsageCmd.RunE(missionReportUsageCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "malformed")
}

func TestMissionReportUsageCmd_RequiresMissionID(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "")
	setMissionReportUsageFlags(t, root, "", 10, 10)
	t.Cleanup(func() { resetMissionReportUsageFlags(t) })

	err := missionReportUsageCmd.RunE(missionReportUsageCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--mission-id is required")
}

func TestMissionReportUsageCmd_RequiresTokensInFlag(t *testing.T) {
	root := setupMissionReportUsageRoot(t, "20260830-req-mission")
	t.Cleanup(func() { resetMissionReportUsageFlags(t) })
	require.NoError(t, missionReportUsageCmd.Flags().Set(flagRoot, root))
	require.NoError(t, missionReportUsageCmd.Flags().Set("mission-id", "20260830-req-mission"))
	require.NoError(t, missionReportUsageCmd.Flags().Set("tokens-out", "10"))
	// --tokens-in deliberately left unset (Changed()==false).

	err := missionReportUsageCmd.RunE(missionReportUsageCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--tokens-in is required")
}

func TestMissionIDKnown(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "refined", "m-known"), 0o755))

	assert.True(t, missionIDKnown(dir, "m-known"))
	assert.False(t, missionIDKnown(dir, "m-unknown"))
}

func TestMissionCmd_IsHumanStatusCommand(t *testing.T) {
	assert.False(t, isHumanStatusCommand(missionCmd))
}
