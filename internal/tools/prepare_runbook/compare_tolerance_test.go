package preparerunbook

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/runbook"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeInventory(t *testing.T, root string, files map[string][]byte) {
	t.Helper()
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(root, name+".runbook.yaml"), content, 0o600))
	}
}

func TestCompareSkipsAnInvalidEntryAndStillClassifies(t *testing.T) {
	root := t.TempDir()
	candidate := runbook.Runbook{Objective: "recover the provider path", AppliesWhen: []string{"provider recovery"}}
	writeInventory(t, root, map[string][]byte{
		"a-broken": []byte("not: a runbook"),
		"b-valid":  validSidecar("valid"),
	})
	comparison, err := compare(Input{RunbookRoot: root}, candidate)
	require.NoError(t, err)
	assert.Equal(t, 1, comparison.SkippedEntries)
	assert.NotEqual(t, ComparisonInsufficient, comparison.Classification, "one invalid sidecar must not make the comparison insufficient")
	assert.Contains(t, comparison.Reason, "1 inventory entries could not be compared")
}

func TestCompareIsInsufficientOnlyWhenNoEntryCanBeCompared(t *testing.T) {
	root := t.TempDir()
	writeInventory(t, root, map[string][]byte{"a-broken": []byte("not: a runbook")})
	comparison, err := compare(Input{RunbookRoot: root}, runbook.Runbook{})
	require.NoError(t, err)
	assert.Equal(t, ComparisonInsufficient, comparison.Classification)
	assert.Equal(t, 1, comparison.SkippedEntries)
}

func TestCompareReportsTruncationAndCannotConcludeNovel(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{}
	for i := 0; i < 5; i++ {
		files[fmt.Sprintf("entry-%d", i)] = validSidecar(fmt.Sprintf("entry-%d", i))
	}
	writeInventory(t, root, files)
	candidate := runbook.Runbook{Objective: "something nobody wrote", AppliesWhen: []string{"zzz unrelated topic"}}
	comparison, err := compare(Input{RunbookRoot: root, MaxInventory: 2}, candidate)
	require.NoError(t, err)
	assert.True(t, comparison.Truncated)
	assert.Equal(t, 3, comparison.NotCompared)
	assert.Equal(t, ComparisonInsufficient, comparison.Classification, "a partial inventory cannot support a novel verdict")
	assert.Contains(t, comparison.Reason, "3 entries were not compared")
}

func TestCompareAppliesHintPriorityBeforeTheCap(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{}
	for _, name := range []string{"a-one", "b-two", "c-three", "z-hinted"} {
		files[name] = validSidecar(name)
	}
	writeInventory(t, root, files)
	comparison, err := compare(Input{RunbookRoot: root, MaxInventory: 1, SelectedRunbooksHint: []string{"z-hinted"}}, runbook.Runbook{Objective: "x", AppliesWhen: []string{"y"}})
	require.NoError(t, err)
	assert.True(t, comparison.Truncated)
	assert.Equal(t, 3, comparison.NotCompared)
	assert.Equal(t, []string{"z-hinted"}, comparison.SelectedHints)
}

func TestCompareWithAnInventoryUnderTheCapReportsNoTruncation(t *testing.T) {
	root := t.TempDir()
	writeInventory(t, root, map[string][]byte{"only": validSidecar("only")})
	comparison, err := compare(Input{RunbookRoot: root}, runbook.Runbook{Objective: "x", AppliesWhen: []string{"y"}})
	require.NoError(t, err)
	assert.False(t, comparison.Truncated)
	assert.Zero(t, comparison.NotCompared)
	assert.Equal(t, ComparisonNovel, comparison.Classification)
}
