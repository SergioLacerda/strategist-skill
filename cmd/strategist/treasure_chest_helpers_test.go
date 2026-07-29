package main

import (
	"os"
	"runtime"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- telemetryRunFromCmd ---

func TestTelemetryRunFromCmd_NilCmd(t *testing.T) {
	t.Parallel()
	assert.Nil(t, telemetryRunFromCmd(nil))
}

func TestTelemetryRunFromCmd_NilContext(t *testing.T) {
	t.Parallel()
	// A cobra.Command with no context set → cmd.Context() returns nil.
	// Use the actual treasureChestCmd which starts without a context.
	assert.Nil(t, telemetryRunFromCmd(treasureChestCmd))
}

// --- treasureChestRootFromCmd / stringFlag / boolFlag ---

func TestTreasureChestRootFromCmd_NilCmd(t *testing.T) {
	t.Parallel()
	assert.Empty(t, treasureChestRootFromCmd(nil))
}

func TestTreasureChestRootFromCmd_NoFlagAnywhereReturnsEmpty(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{Use: "no-root-flag"}
	assert.Empty(t, treasureChestRootFromCmd(cmd))
}

func TestStringFlag_NilCmdReturnsFallback(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "fallback", stringFlag(nil, "missing", "fallback"))
}

func TestStringFlag_NoFlagAnywhereReturnsFallback(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{Use: "no-such-flag"}
	assert.Equal(t, "fallback", stringFlag(cmd, "missing", "fallback"))
}

func TestBoolFlag_NilCmdReturnsFallback(t *testing.T) {
	t.Parallel()
	assert.True(t, boolFlag(nil, "missing", true))
}

func TestBoolFlag_NoFlagAnywhereReturnsFallback(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{Use: "no-such-flag"}
	assert.True(t, boolFlag(cmd, "missing", true))
}

// --- resolveTreasureChestActionRoot ---

func TestResolveTreasureChestActionRoot_GetwdError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chdir-then-remove not reliable on windows")
	}
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(oldWd) })

	removed := t.TempDir()
	require.NoError(t, os.Chdir(removed))
	require.NoError(t, os.RemoveAll(removed))

	_, actionErr := resolveTreasureChestActionRoot(nil, "treasure-chest doctor")
	require.Error(t, actionErr)
	assert.Contains(t, actionErr.Error(), "get cwd")
}

func TestTelemetryRunFromCmd_WithNonNilContext(t *testing.T) {
	t.Parallel()
	// Use a private *cobra.Command, not a shared package-level command var — other
	// parallel tests read package-level commands' Context() concurrently, and
	// SetContext on a shared command races with those reads.
	cmd := &cobra.Command{}
	// Set a background context so cmd.Context() returns non-nil.
	cmd.SetContext(t.Context())
	result := telemetryRunFromCmd(cmd)
	// MissionRunFromContext returns nil when ctx has no embedded run.
	assert.Nil(t, result)
}
