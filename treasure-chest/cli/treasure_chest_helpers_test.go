package treasurecli

import (
	"os"
	"runtime"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- treasureChestRootFromCmd ---

func TestTreasureChestRootFromCmd_NilCmd(t *testing.T) {
	t.Parallel()
	assert.Empty(t, treasureChestRootFromCmd(nil))
}

func TestTreasureChestRootFromCmd_NoFlagAnywhereReturnsEmpty(t *testing.T) {
	t.Parallel()
	cmd := &cobra.Command{Use: "no-root-flag"}
	assert.Empty(t, treasureChestRootFromCmd(cmd))
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
