package treasurecli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/tabwriter"

	"bytes"
	"errors"

	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreasureChestFlagHelpers_InheritedAndFallback(t *testing.T) {
	parent := &cobra.Command{Use: "parent"}
	parent.PersistentFlags().String("root", "from-parent", "")
	parent.PersistentFlags().Bool("enabled", true, "")
	child := &cobra.Command{Use: "child"}
	parent.AddCommand(child)

	assert.Equal(t, "from-parent", treasureChestRootFromCmd(child))
	assert.Equal(t, "from-parent", stringFlag(child, "root", "fallback"))
	assert.True(t, boolFlag(child, "enabled", false))
	assert.Equal(t, "fallback", stringFlag(child, "missing", "fallback"))
	assert.True(t, boolFlag(nil, "enabled", true))
	assert.False(t, boolFlag(child, "missing", false))
}

func TestRenderTreasureChestJSON_CorruptAndOKHealth(t *testing.T) {
	row := treasure.StatusRow{
		ID:          "source",
		Path:        ".sdd/source",
		Scope:       []string{"all"},
		TrustTier:   "T2",
		Freshness:   "unknown",
		Drift:       []string{"missing_index"},
		SourceGrade: "A",
		ReuseValue:  "high",
		OpenGaps:    []string{"gap-1"},
		JewelCount:  2,
	}

	tests := []struct {
		name        string
		compErr     error
		compiledAt  int64
		wantHealth  string
		wantWarning string
	}{
		{
			name:        "corrupt",
			compErr:     errors.New("bad gzip"),
			wantHealth:  "corrupt",
			wantWarning: ".compiled/.index.gz corrupt",
		},
		{
			name:       "ok",
			compiledAt: 1_700_000_000,
			wantHealth: "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "out.json")
			f, err := os.Create(path)
			require.NoError(t, err)
			require.NoError(t, renderTreasureChestJSON(f, "/tmp/root", []treasure.StatusRow{row}, tt.compErr, nil, nil, tt.compiledAt))
			require.NoError(t, f.Close())

			var out jsonTreasureChestOutput
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(raw, &out))
			require.Len(t, out.Chests, 1)
			assert.Equal(t, "source", out.Chests[0].ID)
			assert.Equal(t, tt.wantHealth, out.Index.Health)
			if tt.wantWarning != "" {
				assert.Contains(t, strings.Join(out.Warnings, "\n"), tt.wantWarning)
			}
		})
	}
}

func TestRenderTreasureChestSections_EdgeBranches(t *testing.T) {
	var chests bytes.Buffer
	cw := tabwriter.NewWriter(&chests, 0, 0, 3, ' ', 0)
	require.NoError(t, renderChestsSection(cw, []treasure.StatusRow{{
		ID:        "indexed-only",
		Drift:     []string{"unscoped"},
		Freshness: "unknown",
	}}))
	require.NoError(t, cw.Flush())
	assert.Contains(t, chests.String(), "indexed-only")
	assert.Contains(t, chests.String(), "unscoped")

	var index bytes.Buffer
	iw := tabwriter.NewWriter(&index, 0, 0, 3, ' ', 0)
	require.NoError(t, renderIndexSection(iw, "/tmp/root", 1_700_000_000, nil))
	require.NoError(t, iw.Flush())
	assert.Contains(t, index.String(), "ok")
	assert.Contains(t, index.String(), "2023-11-14")
}

func TestTreasureChestIndexCmd_MissingGovernance(t *testing.T) {
	root := t.TempDir()
	testutilMinimalRootWithoutGovernance(t, root)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, root)

	err := runTreasureChestIndexCmd(treasureChestIndexCmd, nil, treasureChestIndexOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest index")
}

func TestTreasureChestIndexCmd_InvalidScoringPolicy(t *testing.T) {
	root := minimalTreasureChestRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "treasure-chests.yaml"), []byte(`
schema_version: "1"
scoring_policy:
  max_score: 101
chests:
  - id: source
    title: Source
    path: .sdd/source
    trust:
      tier: T1
      reviewed_by: human
      last_reviewed: 2026-06-24
`), 0o644))
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, root)

	err := runTreasureChestIndexCmd(treasureChestIndexCmd, nil, treasureChestIndexOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scoring_policy")
}

func TestGovernedRows_CopiesTrustTiers(t *testing.T) {
	rows := governedRows(map[string]treasure.GovernedChest{
		"a": {Trust: treasure.GovernedTrust{Tier: "T1"}},
		"b": {Trust: treasure.GovernedTrust{Tier: "T3"}},
	})
	require.Len(t, rows, 2)
	assert.ElementsMatch(t, []string{"T1", "T3"}, []string{rows[0].TrustTier, rows[1].TrustTier})
}

func TestLoadJewelsForCmd_MissingGovernedIsBestEffort(t *testing.T) {
	root := t.TempDir()
	testutilMinimalRootWithoutGovernance(t, root)
	require.NoError(t, os.WriteFile(filepath.Join(root, "jewels.yaml"), []byte(oneProposedJewelYAML), 0o644))
	cmd := &cobra.Command{Use: "jewel-list"}
	cmd.Flags().String("root", root, "")

	jewelsByChest, err := loadJewelsForCmd(cmd, "test")
	require.NoError(t, err)
	require.Len(t, jewelsByChest["source"], 1)
}

func TestTreasureChestItemsCommands_UnknownFormat(t *testing.T) {
	root := itemsTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestItemsFlags(t)
	setTreasureChestRoot(t, root)
	setCmdFlag(t, treasureChestItemsListCmd, "format", "xml")

	err := runTreasureChestItemsList(treasureChestItemsListCmd, nil, treasureChestItemsListOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown --format")

	setCmdFlag(t, treasureChestItemsShowCmd, "format", "xml")
	err = runTreasureChestItemsShow(treasureChestItemsShowCmd, []string{"jewel-1"}, treasureChestItemsShowOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown --format")
}

func TestTreasureChestRenderers_ClosedStdoutErrors(t *testing.T) {
	j := treasure.Jewel{
		ID:         "jewel-1",
		ChestID:    "source",
		Kind:       "pattern",
		Statement:  "statement",
		SourceRefs: []string{"source#a"},
		Trust:      "T1",
		Status:     "proposed",
		Score:      treasure.JewelScore{Value: 10, Reasons: []string{"reason"}},
	}

	p := treasure.Potion{
		ID:         "potion-1",
		ChestID:    "runbooks",
		RunbookRef: "docs/runbooks/sample.md",
		WhenToUse:  "when sample breaks",
		Trust:      "T2",
		Status:     "proposed",
	}

	withClosedStdout(t, func() {
		require.Error(t, renderItemTable([]itemRow{jewelToItemRow(j)}, "empty", "treasure-chest items list"))
		require.Error(t, renderItemJSON([]itemRow{jewelToItemRow(j)}, "treasure-chest items list"))
		require.Error(t, renderJewelShowTable(j))
		require.Error(t, renderJewelShowJSON(j))
		require.Error(t, renderPotionShowTable(p))
		require.Error(t, renderPotionShowJSON(p))
	})
}

func TestTreasureChestTableRenderers_ClosedStdoutErrors(t *testing.T) {
	rows := []treasure.StatusRow{{ID: "c1", Path: "docs", Freshness: "unknown"}}

	withClosedStdout(t, func() {
		require.Error(t, renderTreasureChestTable("/tmp/root", rows, nil, nil, nil, 0))
	})
}

func TestTreasureChestItemsHelpers_ErrorAndNoopBranches(t *testing.T) {
	root := itemsTestRoot(t, oneProposedJewelYAML)

	err := runTreasureChestItemsPromote(root, nil, "accepted", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provide at least one item id")

	emptyRoot := itemsTestRoot(t, "schema_version: \"1\"\njewels: []\n")
	out := captureStdout(t, func() {
		require.NoError(t, runTreasureChestItemsMigrateStatus(emptyRoot))
	})
	assert.Contains(t, out, "nothing to migrate")
}

func TestLoadJewelsForCmd_InvalidJewelsErrors(t *testing.T) {
	root := itemsTestRoot(t, `
schema_version: "1"
jewels:
  - id: bad
    chest_id: source
    kind: unknown
    source_refs: ["source#a"]
    trust: T1
    status: proposed
    score: { value: 1 }
`)
	cmd := &cobra.Command{Use: "items-list"}
	cmd.Flags().String("root", root, "")

	_, err := loadJewelsForCmd(cmd, "treasure-chest items list")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest items list")
}

func TestRunTreasureChest_LoadJewelsError(t *testing.T) {
	root := itemsTestRoot(t, `
schema_version: "1"
jewels:
  - id: bad
    chest_id: source
    kind: unknown
    source_refs: ["source#a"]
    trust: T1
    status: proposed
    score: { value: 1 }
`)
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, root)

	err := treasureChestCmd.RunE(treasureChestCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest")
}

func TestLoadJewelsForCmd_LoadJewelsError(t *testing.T) {
	root := t.TempDir()
	testutilMinimalRootWithoutGovernance(t, root)
	require.NoError(t, os.WriteFile(filepath.Join(root, "jewels.yaml"), []byte(`
schema_version: "1"
jewels:
  - id: bad
    chest_id: source
    kind: invalid
    source_refs: ["source#a"]
    trust: T1
    status: proposed
    score: { value: 1 }
`), 0o644))
	cmd := &cobra.Command{Use: "jewel-list"}
	cmd.Flags().String("root", root, "")

	_, err := loadJewelsForCmd(cmd, "test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test:")
}

func testutilMinimalRootWithoutGovernance(t *testing.T, root string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
`), 0o644))
}
