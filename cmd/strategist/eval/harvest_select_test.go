package eval

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/SergioLacerda/strategist-skill/treasure-chest"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scanTestRoot builds a project tree with a .strategist/ root and an .analysis/
// base_path, matching resolveDojoRoots' expected layout (base_path resolved
// relative to the .strategist root's parent directory). Duplicated from
// cmd/strategist's own eval_harvest_test.go (itself duplicated from
// internal/treasurecli's treasure_chest_scan_test.go) — Go test helpers
// aren't shareable across package boundaries.
func scanTestRoot(t *testing.T) (strategistDir, basePath string) {
	t.Helper()
	root := t.TempDir()
	strategistDir = filepath.Join(root, ".strategist")
	basePath = filepath.Join(root, ".analysis")
	require.NoError(t, os.MkdirAll(strategistDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(strategistDir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sniper
`), 0o644))
	return strategistDir, basePath
}

func TestParseHarvestInclude_Empty(t *testing.T) {
	out, err := ParseHarvestInclude("")
	require.NoError(t, err)
	assert.Nil(t, out)
}

func TestParseHarvestInclude_ValidList(t *testing.T) {
	out, err := ParseHarvestInclude("design, tasks,adr")
	require.NoError(t, err)
	assert.Equal(t, []string{"design", "tasks", "adr"}, out)
}

func TestParseHarvestInclude_UnknownValue(t *testing.T) {
	_, err := ParseHarvestInclude("bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
}

func TestParseHarvestInclude_SkipsBlankSegments(t *testing.T) {
	out, err := ParseHarvestInclude("design,,tasks")
	require.NoError(t, err)
	assert.Equal(t, []string{"design", "tasks"}, out)
}

func TestSelectHarvestMissionIDs_SingleMission(t *testing.T) {
	ids, warnings, err := SelectHarvestMissionIDs([]string{"m-1"}, HarvestOptions{}, "")
	require.NoError(t, err)
	assert.Equal(t, []string{"m-1"}, ids)
	assert.Nil(t, warnings)
}

func TestSelectHarvestMissionIDs_RequiresExactlyOneWithoutAll(t *testing.T) {
	_, _, err := SelectHarvestMissionIDs(nil, HarvestOptions{}, "")
	require.Error(t, err)

	_, _, err = SelectHarvestMissionIDs([]string{"a", "b"}, HarvestOptions{}, "")
	require.Error(t, err)
}

func TestSelectHarvestMissionIDs_AllRejectsMissionID(t *testing.T) {
	_, _, err := SelectHarvestMissionIDs([]string{"m-1"}, HarvestOptions{All: true}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestSelectHarvestMissionIDs_AllScansRefinedAndDone(t *testing.T) {
	_, base := scanTestRoot(t)
	writeMissionTasks(t, base, "refined", "m-refined", "## Task 1 — Example\n")
	writeMissionTasks(t, base, "done", "m-done", "## Task 1 — Example\n")

	ids, warnings, err := SelectHarvestMissionIDs(nil, HarvestOptions{All: true}, base)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"m-refined", "m-done"}, ids)
	assert.Empty(t, warnings)
}

// TestSelectHarvestMissionIDs_AllSkipsInconsistentMission reproduces the real
// failure behind .analysis/archived/20260804-treasure-scan-sq-block-bug-adr.md:
// a side_quests_approved: block using standard 2-space-indented list items
// (as opposed to the 0-indent style used elsewhere in this test file) trips
// scan_parse.go's normalizeLegacySideQuestFields, which strips the leading
// "- " from the indented first item, corrupting the YAML. --all must skip
// that mission with a warning instead of aborting the whole run.
func TestSelectHarvestMissionIDs_AllSkipsInconsistentMission(t *testing.T) {
	_, base := scanTestRoot(t)
	writeMissionTasks(t, base, "done", "good-mission", "## Task 1 — Example\n")
	writeMissionTasks(t, base, "done", "bad-mission",
		"side_quests_approved:\n  - id: SQ-001\n    description: indented list item\n    status: sq_backlog\n## Task 1 — Example\n")

	ids, warnings, err := SelectHarvestMissionIDs(nil, HarvestOptions{All: true}, base)

	require.NoError(t, err)
	assert.Equal(t, []string{"good-mission"}, ids)
	require.Len(t, warnings, 1)
	assert.Contains(t, warnings[0].Path, "bad-mission")
}

func TestSelectAllHarvestMissionIDs_ScanErrorPropagates(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ENOTDIR scan semantics differ on Windows")
	}
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o644))

	_, _, err := selectAllHarvestMissionIDs(blocker)
	require.ErrorContains(t, err, "eval harvest")
}

func TestPrintHarvestWarnings_PrintsEachWarning(t *testing.T) {
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetErr(&buf)

	PrintHarvestWarnings(cmd, []treasure.ScanWarning{{Path: "refined/bad-mission", Err: errors.New("unparseable")}})
	assert.Contains(t, buf.String(), "skipped inconsistent mission file")
}
