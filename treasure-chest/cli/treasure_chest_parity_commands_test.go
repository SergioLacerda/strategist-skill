package treasurecli

import (
	"encoding/json"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreasureChestCommandTreeParity(t *testing.T) {
	want := map[string]bool{
		"treasure-chest":                      false,
		"treasure-chest list":                 false,
		"treasure-chest add":                  false,
		"treasure-chest remove":               false,
		"treasure-chest doctor":               false,
		"treasure-chest index":                false,
		"treasure-chest scan":                 true,
		"treasure-chest items":                false,
		"treasure-chest items list":           false,
		"treasure-chest items show":           false,
		"treasure-chest items accept":         false,
		"treasure-chest items verify":         false,
		"treasure-chest items deprecate":      false,
		"treasure-chest items migrate-status": false,
		"treasure-chest items check-evidence": false,
	}

	got := make(map[string]bool)
	var walk func(*cobra.Command)
	walk = func(cmd *cobra.Command) {
		got[cmd.CommandPath()] = cmd.Hidden
		for _, child := range cmd.Commands() {
			walk(child)
		}
	}
	walk(treasureChestCmd)
	for path, hidden := range want {
		assert.Equal(t, hidden, got[path], path)
	}
	assert.Len(t, got, len(want))
}

func TestTreasureChestCommands_ReadOnlyJSONParity(t *testing.T) {
	root := parityFixture(t)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, root)
	setTreasureChestFormat(t, outputFormatJSON)
	before := paritySnapshot(t, root)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestCmd.RunE(treasureChestCmd, nil))
	})
	var status map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &status))
	assert.Contains(t, status, "chests")

	setTreasureChestFormat(t, outputFormatTable)
	out = captureStdout(t, func() {
		require.NoError(t, treasureChestListCmd.RunE(treasureChestListCmd, nil))
	})
	assert.Contains(t, out, "CHESTS")

	resetTreasureChestItemsFlags(t)
	setCmdFlag(t, treasureChestItemsListCmd, "format", outputFormatJSON)
	out = captureStdout(t, func() {
		require.NoError(t, treasureChestItemsListCmd.RunE(treasureChestItemsListCmd, nil))
	})
	var items []map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &items))
	assert.NotEmpty(t, items)
	assert.Equal(t, before, paritySnapshot(t, root), "read-only commands changed fixture")
}

func TestTreasureChestCommands_DoctorAndHiddenScanDryRun(t *testing.T) {
	root := parityScanFixture(t)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, root)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestDoctorCmd.RunE(treasureChestDoctorCmd, nil))
	})
	assert.Contains(t, out, "no consistency drift")

	before := paritySnapshot(t, root)
	setCmdFlag(t, treasureChestScanCmd, "dry-run", "true")
	out = captureStdout(t, func() {
		require.NoError(t, treasureChestScanCmd.RunE(treasureChestScanCmd, nil))
	})
	assert.Contains(t, out, "dry-run")
	assert.Equal(t, before, paritySnapshot(t, root))
}
