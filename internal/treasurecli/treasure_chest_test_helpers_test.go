package treasurecli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/telemetry"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// attachMissionRun wires a non-nil telemetry.MissionRun into cmd's context, so
// that RunE bodies gated on `telemetryRunFromCmd(cmd) != nil` (the SetSilent()
// branch present in most treasure-chest subcommands) get exercised. Returns
// the run so callers can inspect it (e.g. Snapshot()) if needed, and restores
// the command's original context on test cleanup. Duplicated from
// cmd/strategist's own cmd_test_helpers_test.go — see captureStdout's comment
// above for why.
func attachMissionRun(t *testing.T, cmd *cobra.Command) *telemetry.MissionRun {
	t.Helper()
	run := telemetry.NewMissionRun("test-mission")
	origCtx := cmd.Context()
	t.Cleanup(func() { cmd.SetContext(origCtx) })
	cmd.SetContext(telemetry.WithMissionRun(context.Background(), run))
	return run
}

// captureStdout replaces os.Stdout with a pipe and returns whatever was written.
// Duplicated from cmd/strategist's own cmd_test_helpers_test.go — Go test helpers
// aren't shareable across package boundaries, and this ~12-line helper isn't worth
// promoting to an exported package just for two callers.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stdout
	os.Stdout = w
	fn()
	require.NoError(t, w.Close())
	os.Stdout = old
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	return buf.String()
}

// chdirForTest changes the working directory to dir for the duration of the
// test, restoring the original on cleanup. Duplicated from cmd/strategist's
// own root_config_warning_test.go — see captureStdout's comment above.
func chdirForTest(t *testing.T, dir string) {
	t.Helper()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
}

// captureStderr replaces os.Stderr with a pipe and returns whatever was written.
// Duplicated from cmd/strategist's own cmd_test_helpers_test.go — see
// captureStdout's comment above.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stderr
	os.Stderr = w
	fn()
	require.NoError(t, w.Close())
	os.Stderr = old
	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	return buf.String()
}

// withClosedStdout runs fn with os.Stdout pointing at an already-closed file, so
// any write to stdout inside fn fails — used to exercise renderer error branches.
func withClosedStdout(t *testing.T, fn func()) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "closed-stdout-*")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	old := os.Stdout
	os.Stdout = f
	t.Cleanup(func() { os.Stdout = old })
	fn()
}

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
