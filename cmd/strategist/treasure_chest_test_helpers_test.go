package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// minimalTreasureChestRoot builds a .strategist/-like tree for treasure-chest command tests.
func minimalTreasureChestRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
treasure_chests:
  - id: source
    path: .sdd/source
    scope: all
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
chests:
  - id: source
    title: Source
    path: .sdd/source
    trust:
      tier: T1
      reviewed_by: human
      last_reviewed: 2026-06-24
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"), []byte(`
sources:
  - id: source
    path: .sdd/source
    tags: [all]
`), 0o644))

	return dir
}

// resetTreasureChestFlags saves and restores all treasure-chest command flags.
func resetTreasureChestFlags(t *testing.T) {
	t.Helper()
	origRoot, err := treasureChestCmd.PersistentFlags().GetString("root")
	require.NoError(t, err)
	origIndex, err := treasureChestCmd.Flags().GetBool("index")
	require.NoError(t, err)
	origHist, err := treasureChestCmd.Flags().GetBool("include-historical")
	require.NoError(t, err)
	origFmt, err := treasureChestCmd.Flags().GetString("format")
	require.NoError(t, err)
	origScope, err := treasureChestCmd.Flags().GetString("scope")
	require.NoError(t, err)
	t.Cleanup(func() {
		setTreasureChestRoot(t, origRoot)
		setTreasureChestDoIndex(t, origIndex)
		setTreasureChestIncludeHistorical(t, origHist)
		setTreasureChestFormat(t, origFmt)
		setTreasureChestScope(t, origScope)
	})
	setTreasureChestRoot(t, "")
	setTreasureChestDoIndex(t, false)
	setTreasureChestIncludeHistorical(t, false)
	setTreasureChestFormat(t, "table")
	setTreasureChestScope(t, "")
	setCmdFlag(t, treasureChestScanCmd, "dry-run", "false")
}

func setTreasureChestRoot(t *testing.T, value string) {
	t.Helper()
	require.NoError(t, treasureChestCmd.PersistentFlags().Set("root", value))
}

func setTreasureChestDoIndex(t *testing.T, value bool) {
	t.Helper()
	require.NoError(t, treasureChestCmd.Flags().Set("index", fmt.Sprint(value)))
}

func setTreasureChestIncludeHistorical(t *testing.T, value bool) {
	t.Helper()
	require.NoError(t, treasureChestCmd.Flags().Set("include-historical", fmt.Sprint(value)))
}

func setTreasureChestFormat(t *testing.T, value string) {
	t.Helper()
	require.NoError(t, treasureChestCmd.Flags().Set("format", value))
}

func setTreasureChestScope(t *testing.T, value string) {
	t.Helper()
	require.NoError(t, treasureChestCmd.Flags().Set("scope", value))
}

func setCmdFlag(t *testing.T, cmd *cobra.Command, name, value string) {
	t.Helper()
	require.NoError(t, cmd.Flags().Set(name, value))
}

func cmdFlagString(t *testing.T, cmd *cobra.Command, name string) string {
	t.Helper()
	value, err := cmd.Flags().GetString(name)
	require.NoError(t, err)
	return value
}

func cmdFlagBool(t *testing.T, cmd *cobra.Command, name string) bool {
	t.Helper()
	value, err := cmd.Flags().GetBool(name)
	require.NoError(t, err)
	return value
}
