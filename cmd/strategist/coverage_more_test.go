package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
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

func TestSimulateReport_ErrorAndWriterFailure(t *testing.T) {
	okOut := captureStdout(t, func() {
		require.NoError(t, printSimulateReport("/tmp/root", map[string]string{
			"discovery":  "ranger",
			"refinement": "archivist",
			"execution":  "sniper",
		}, "epic", "test", nil))
	})
	assert.Contains(t, okOut, "status=ready")

	out := captureStdout(t, func() {
		err := printSimulateReport("/tmp/root", map[string]string{
			"discovery": "ranger",
			"execution": "sniper",
		}, "epic", "test", []string{"missing refinement"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "errors=1")
	})
	assert.Contains(t, out, "missing_provider")
	assert.Contains(t, out, "missing refinement")

	sw := &simReportWriter{w: tabwriter.NewWriter(errorWriter{}, 0, 0, 3, ' ', 0)}
	sw.line("boom\n")
	require.Error(t, sw.err)
	sw.line("ignored\n")
	assert.Contains(t, sw.err.Error(), "write output")
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) {
	return 0, errors.New("forced write failure")
}

func TestValidateHelpers_DirectErrorBranches(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: experimental
roles_config: roles/default.yaml
`), 0o644))
	err := validateActiveYAML(filepath.Join(dir, "active.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mode")

	errs, checks := validatePersonasDir(filepath.Join(dir, "missing-personas"))
	assert.Zero(t, checks)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "personas/")

	errs, checks = validateRolesDir(filepath.Join(dir, "missing-roles"))
	assert.Zero(t, checks)
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0], "roles/")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte("not: [yaml"), 0o644))
	err = validateYAMLFile(filepath.Join(dir, "bad.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid YAML")
}

func TestValidateRoleFile_RoleDefinitionAndSlotMapErrors(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "role.yaml"), []byte(`
role: scout
specialization:
  slot: invalid
`), 0o644))
	errs := validateRoleFile(dir, "role.yaml")
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "roles/role.yaml")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "slots.yaml"), []byte(`
discovery: scout
`), 0o644))
	errs = validateRoleFile(dir, "slots.yaml")
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "roles/slots.yaml")
}

func TestInitiativeWorkspaceSection_DefaultBaseAndWriteError(t *testing.T) {
	root := t.TempDir()
	project := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(project, ".analysis", "pending"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "memory"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "memory", "ignored"), nil, 0o644))

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	err := writeWorkspaceSection(w, domain.ActiveConfig{Mode: "epic"}, root, project)
	require.NoError(t, err)
	require.NoError(t, w.Flush())
	assert.Contains(t, buf.String(), "base_path")
	assert.Contains(t, buf.String(), ".analysis")

	bad := tabwriter.NewWriter(errorWriter{}, 0, 0, 3, ' ', 0)
	require.NoError(t, writeWorkspaceSection(bad, domain.ActiveConfig{Mode: "epic"}, root, project))
	assert.Error(t, bad.Flush())
}

func TestRootHelpers_HumanStatusAndPreRun(t *testing.T) {
	assert.False(t, isHumanStatusCommand(nil))
	parent := &cobra.Command{Use: "treasure-chest"}
	child := &cobra.Command{Use: "jewel"}
	parent.AddCommand(child)
	assert.True(t, isHumanStatusCommand(child))

	cmd := &cobra.Command{Use: "custom"}
	require.NoError(t, rootCmd.PersistentPreRunE(cmd, nil))
	assert.NotNil(t, cmd.Context())
	require.NoError(t, rootCmd.PersistentPostRunE(cmd, nil))
}

func TestValidateRuntimeDefaultParity_DetectsRuntimeDrift(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "SKILL.md"), []byte("stale runtime copy"), 0o644))

	errs := validateRuntimeDefaultParity(root)
	require.NotEmpty(t, errs)
	assert.Contains(t, errs[0], `normative file "SKILL.md" differs`)
}

func TestRunCompile_CompileAllError(t *testing.T) {
	orig := compileRoot
	t.Cleanup(func() { compileRoot = orig })
	compileRoot = t.TempDir()

	err := runCompile(&cobra.Command{Use: "compile"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "compile: compile all")
}

func TestRunInstall_RejectsConflictingShimFlags(t *testing.T) {
	origNoShim := installNoShim
	origShimPath := installShimPath
	t.Cleanup(func() {
		installNoShim = origNoShim
		installShimPath = origShimPath
	})
	installNoShim = true
	installShimPath = filepath.Join(t.TempDir(), "SKILL.md")

	err := runInstall(&cobra.Command{Use: "install"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mutually exclusive")
}

func TestRunInitiative_InvalidActiveYAML(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "active.yaml"), []byte("mode: ["), 0o644))

	orig := initiativeRoot
	t.Cleanup(func() { initiativeRoot = orig })
	initiativeRoot = root

	err := runInitiative(&cobra.Command{Use: "initiative"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse active.yaml")
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

func TestTreasureChestJewelCommands_UnknownFormat(t *testing.T) {
	root := mineTestRoot(t, oneProposedJewelYAML)
	resetTreasureChestFlags(t)
	resetTreasureChestJewelFlags(t)
	setTreasureChestRoot(t, root)
	setCmdFlag(t, treasureChestJewelListCmd, "format", "xml")

	err := runTreasureChestJewelList(treasureChestJewelListCmd, nil, treasureChestJewelListOptions{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown --format")

	setCmdFlag(t, treasureChestJewelShowCmd, "format", "xml")
	err = runTreasureChestJewelShow(treasureChestJewelShowCmd, []string{"jewel-1"}, treasureChestJewelShowOptions{})
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

	withClosedStdout(t, func() {
		require.Error(t, renderJewelListTable([]treasure.Jewel{j}))
		require.Error(t, renderJewelListJSON([]treasure.Jewel{j}))
		require.Error(t, renderJewelShowTable(j))
		require.Error(t, renderJewelShowJSON(j))
		require.Error(t, renderMineTable([]treasure.Jewel{j}))
		require.Error(t, renderMineJSON([]treasure.Jewel{j}))
	})
}

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

func TestTreasureChestMineHelpers_ErrorAndNoopBranches(t *testing.T) {
	root := mineTestRoot(t, oneProposedJewelYAML)

	err := runTreasureChestMineList(root, "xml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown --format")

	err = runTreasureChestMinePromote(root, nil, "accepted", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provide at least one jewel id")

	emptyRoot := mineTestRoot(t, "schema_version: \"1\"\njewels: []\n")
	out := captureStdout(t, func() {
		require.NoError(t, runTreasureChestMineMigrateStatus(emptyRoot))
	})
	assert.Contains(t, out, "nothing to migrate")
}

func TestTreasureChestMineList_InvalidJewelsErrors(t *testing.T) {
	root := mineTestRoot(t, `
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

	err := runTreasureChestMineList(root, "table")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chest mine")
}

func TestRunTreasureChest_LoadJewelsError(t *testing.T) {
	root := mineTestRoot(t, `
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
roles_config: roles/default.yaml
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
`), 0o644))
}
