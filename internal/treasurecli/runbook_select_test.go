package treasurecli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixtureRunbookA = `schema_version: "1"
runbook_id: fix-timeout
runbook_type: analytical
source_doc: docs/runbooks/fix-timeout.md
applies_when:
  - timeout
  - connection refused
objective: diagnose timeout issues
checks:
  - id: check-logs
    level: mandatory
`

const fixtureRunbookB = `schema_version: "1"
runbook_id: fix-timeout-secondary
runbook_type: analytical
source_doc: docs/runbooks/fix-timeout-secondary.md
applies_when:
  - timeout
objective: secondary timeout checks
`

const fixtureRunbookUnrelated = `schema_version: "1"
runbook_id: unrelated-runbook
runbook_type: analytical
source_doc: docs/runbooks/unrelated-runbook.md
applies_when:
  - totally different topic
objective: something else entirely
`

const fixtureRunbookInvalidLevel = `schema_version: "1"
runbook_id: broken-runbook
runbook_type: analytical
source_doc: docs/runbooks/broken-runbook.md
applies_when:
  - timeout
objective: has an invalid check level
checks:
  - id: bad-check
    level: required
`

// runbookSelectTestRoot creates docs/runbooks/*.runbook.yaml fixtures under a
// fresh temp project root and returns the root's .strategist path — passed
// as --root, this resolves projectRoot back to the temp dir via
// resolveStrategistRoot's explicit branch (no .strategist/ directory needs
// to actually exist; that branch never stats it).
func runbookSelectTestRoot(t *testing.T, sidecars map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	runbooksDir := filepath.Join(dir, "docs", "runbooks")
	require.NoError(t, os.MkdirAll(runbooksDir, 0o755))
	for name, content := range sidecars {
		require.NoError(t, os.WriteFile(filepath.Join(runbooksDir, name+".runbook.yaml"), []byte(content), 0o644))
	}
	return filepath.Join(dir, ".strategist")
}

// resetRunbookSelectFlags resets --root and --format to their zero values.
// --signal is intentionally not touched here — every test calls
// setRunbookSignals explicitly, which replaces the underlying StringArray
// wholesale (Set on a StringArray appends rather than replaces, so an
// empty-string reset would leave a stray "" element instead of clearing it).
func resetRunbookSelectFlags(t *testing.T) {
	t.Helper()
	setRunbookRoot(t, "")
	require.NoError(t, runbookSelectCmd.Flags().Set(flagFormat, outputFormatTable))
}

// setRunbookRoot sets --root, which is registered on runbookCmd's
// PersistentFlags (not runbookSelectCmd's own Flags) — mirroring
// setTreasureChestRoot's identical pattern for treasureChestCmd.
func setRunbookRoot(t *testing.T, value string) {
	t.Helper()
	require.NoError(t, runbookCmd.PersistentFlags().Set(flagRoot, value))
}

// setRunbookSignals replaces --signal's full value list (StringArray's Set
// appends, so a direct Replace is needed for a clean per-test value).
func setRunbookSignals(t *testing.T, signals ...string) {
	t.Helper()
	sa := runbookSelectCmd.Flags().Lookup("signal")
	require.NotNil(t, sa)
	require.NoError(t, sa.Value.(interface{ Replace([]string) error }).Replace(signals))
}

func TestRunbookSelect_NoSignalErrors(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"fix-timeout": fixtureRunbookA})
	resetRunbookSelectFlags(t)
	setRunbookSignals(t)
	setRunbookRoot(t, root)
	attachMissionRun(t, runbookSelectCmd)

	err := runbookSelectCmd.RunE(runbookSelectCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one --signal is required")
}

func TestRunbookSelect_EmptyCorpusNoError(t *testing.T) {
	root := runbookSelectTestRoot(t, nil)
	resetRunbookSelectFlags(t)
	setRunbookSignals(t, "timeout")
	setRunbookRoot(t, root)
	attachMissionRun(t, runbookSelectCmd)

	out := captureStdout(t, func() {
		require.NoError(t, runbookSelectCmd.RunE(runbookSelectCmd, nil))
	})
	assert.Contains(t, out, "no docs/runbooks/*.runbook.yaml sidecars found")
}

func TestRunbookSelect_NoMatchNoError(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"unrelated-runbook": fixtureRunbookUnrelated})
	resetRunbookSelectFlags(t)
	setRunbookSignals(t, "totally unrelated xyz123 not present anywhere")
	setRunbookRoot(t, root)
	attachMissionRun(t, runbookSelectCmd)

	out := captureStdout(t, func() {
		require.NoError(t, runbookSelectCmd.RunE(runbookSelectCmd, nil))
	})
	assert.Contains(t, out, "no runbook matched the given signals")
}

func TestRunbookSelect_InvalidSidecarHardFails(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"broken-runbook": fixtureRunbookInvalidLevel})
	resetRunbookSelectFlags(t)
	setRunbookSignals(t, "timeout")
	setRunbookRoot(t, root)
	attachMissionRun(t, runbookSelectCmd)

	err := runbookSelectCmd.RunE(runbookSelectCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "broken-runbook.runbook.yaml")
	assert.Contains(t, err.Error(), "check_invalid")
}

func TestRunbookSelect_MatchAssignsPrimaryAndSupportingWithReason(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{
		"fix-timeout":           fixtureRunbookA,
		"fix-timeout-secondary": fixtureRunbookB,
		"unrelated-runbook":     fixtureRunbookUnrelated,
	})
	resetRunbookSelectFlags(t)
	setRunbookSignals(t, "timeout")
	setRunbookRoot(t, root)
	setCmdFlag(t, runbookSelectCmd, flagFormat, outputFormatJSON)
	attachMissionRun(t, runbookSelectCmd)

	out := captureStdout(t, func() {
		require.NoError(t, runbookSelectCmd.RunE(runbookSelectCmd, nil))
	})

	var rows []runbookSelectionRow
	require.NoError(t, json.Unmarshal([]byte(out), &rows))
	require.Len(t, rows, 2, "unrelated-runbook must not be selected — it has no matching applies_when entry")

	assert.Equal(t, "fix-timeout", rows[0].RunbookID)
	assert.Equal(t, "primary", rows[0].Role)
	assert.Equal(t, "runbooks", rows[0].ChestID)
	assert.Equal(t, "docs/runbooks/fix-timeout.md", rows[0].Ref)
	assert.NotEmpty(t, rows[0].Reason)

	assert.Equal(t, "fix-timeout-secondary", rows[1].RunbookID)
	assert.Equal(t, "supporting", rows[1].Role)
	assert.NotEmpty(t, rows[1].Reason)
}

func TestRunbookSelect_TableFormatNoError(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"fix-timeout": fixtureRunbookA})
	resetRunbookSelectFlags(t)
	setRunbookSignals(t, "timeout")
	setRunbookRoot(t, root)
	setCmdFlag(t, runbookSelectCmd, flagFormat, outputFormatTable)
	attachMissionRun(t, runbookSelectCmd)

	out := captureStdout(t, func() {
		require.NoError(t, runbookSelectCmd.RunE(runbookSelectCmd, nil))
	})
	assert.Contains(t, out, "RUNBOOK_ID")
	assert.Contains(t, out, "fix-timeout")
}

func TestRunbookSelectionRenderers_ClosedStdoutErrors(t *testing.T) {
	rows := []runbookSelectionRow{{RunbookID: "r-1", Role: "primary", ChestID: "runbooks", Ref: "docs/runbooks/r-1.md", Reason: "matched"}}

	withClosedStdout(t, func() {
		require.Error(t, renderRunbookSelectionTable(rows))
		require.Error(t, renderRunbookSelectionJSON(rows))
	})
}

func TestSortRunbookSelectionRows_TiesBrokenByRunbookID(t *testing.T) {
	rows := []runbookSelectionRow{
		{RunbookID: "z-supporting", Role: "supporting"},
		{RunbookID: "a-supporting", Role: "supporting"},
	}
	sortRunbookSelectionRows(rows)
	require.Equal(t, "a-supporting", rows[0].RunbookID)
	require.Equal(t, "z-supporting", rows[1].RunbookID)
}

func TestRunbookSelect_UnknownFormatErrors(t *testing.T) {
	root := runbookSelectTestRoot(t, map[string]string{"fix-timeout": fixtureRunbookA})
	resetRunbookSelectFlags(t)
	setRunbookSignals(t, "timeout")
	setRunbookRoot(t, root)
	setCmdFlag(t, runbookSelectCmd, flagFormat, "xml")
	attachMissionRun(t, runbookSelectCmd)

	err := runbookSelectCmd.RunE(runbookSelectCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown --format "xml"`)
}
