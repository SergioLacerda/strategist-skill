package main

// Whitebox tests for low-coverage helpers in treasure_chest.go and initiative.go.

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/SergioLacerda/strategist-skill/internal/treasure"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- readLastMissionID ---

func TestReadLastMissionID_FileAbsent(t *testing.T) {
	t.Parallel()
	result := readLastMissionID(filepath.Join(t.TempDir(), "nonexistent.jsonl"))
	assert.Equal(t, "—", result)
}

func TestReadLastMissionID_WithMissionID(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "outcomes.jsonl")
	line := `{"mission_id":"m-001","status":"done"}`
	require.NoError(t, os.WriteFile(path, []byte(line+"\n"), 0o644))
	assert.Equal(t, "m-001", readLastMissionID(path))
}

func TestReadLastMissionID_LastLineUsed(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "outcomes.jsonl")
	content := `{"mission_id":"m-001"}` + "\n" + `{"mission_id":"m-002"}` + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	assert.Equal(t, "m-002", readLastMissionID(path))
}

func TestReadLastMissionID_EmptyLines(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "outcomes.jsonl")
	require.NoError(t, os.WriteFile(path, []byte("\n\n"), 0o644))
	assert.Equal(t, "—", readLastMissionID(path))
}

func TestReadLastMissionID_InvalidJSON(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "outcomes.jsonl")
	require.NoError(t, os.WriteFile(path, []byte("not-json\n"), 0o644))
	assert.Equal(t, "—", readLastMissionID(path))
}

func TestReadLastMissionID_NoMissionIDField(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "outcomes.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"status":"done"}`+"\n"), 0o644))
	assert.Equal(t, "—", readLastMissionID(path))
}

func TestReadLastMissionID_EmptyMissionID(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "outcomes.jsonl")
	require.NoError(t, os.WriteFile(path, []byte(`{"mission_id":""}`+"\n"), 0o644))
	assert.Equal(t, "—", readLastMissionID(path))
}

// --- treasure.LoadCompiledIndex ---

func writeIndexGz(t *testing.T, path string, compiledAt int64, sourceIDs ...string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	f, err := os.Create(path)
	require.NoError(t, err)
	gz := gzip.NewWriter(f)
	meta := make(map[string]any)
	for _, id := range sourceIDs {
		meta[id] = map[string]any{"id": id}
	}
	require.NoError(t, json.NewEncoder(gz).Encode(map[string]any{
		"compiled_at": compiledAt,
		"source_meta": meta,
	}))
	require.NoError(t, gz.Close())
	require.NoError(t, f.Close())
}

func TestLoadCompiledIndex_NotExist(t *testing.T) {
	t.Parallel()
	ids, ts, err := treasure.LoadCompiledIndex(t.TempDir())
	require.NoError(t, err)
	assert.Nil(t, ids)
	assert.Zero(t, ts)
}

func TestLoadCompiledIndex_CorruptGzip(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".compiled", ".index.gz")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("not gzip"), 0o644))

	_, _, err := treasure.LoadCompiledIndex(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decompress")
}

func TestLoadCompiledIndex_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, ".compiled", ".index.gz")
	writeIndexGz(t, path, 1700000000, "chest-a", "chest-b")

	ids, ts, err := treasure.LoadCompiledIndex(dir)
	require.NoError(t, err)
	assert.EqualValues(t, 1700000000, ts)
	assert.True(t, ids["chest-a"])
	assert.True(t, ids["chest-b"])
	assert.False(t, ids["chest-c"])
}

// --- treasure.LoadIndexed ---

func TestLoadIndexed_NotExist(t *testing.T) {
	t.Parallel()
	result, err := treasure.LoadIndexed(t.TempDir())
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestLoadIndexed_CorruptYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"),
		[]byte(": not: valID:\n"), 0o644))

	_, err := treasure.LoadIndexed(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "knowledge.index.yaml")
}

func TestLoadIndexed_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"),
		[]byte("sources:\n  - id: chest-x\n  - id: chest-y\n"), 0o644))

	result, err := treasure.LoadIndexed(dir)
	require.NoError(t, err)
	assert.True(t, result["chest-x"])
	assert.True(t, result["chest-y"])
}

// --- mergeChestRows ---

func TestMergeChestRows_GovernedNotInActive(t *testing.T) {
	t.Parallel()
	governed := map[string]treasure.GovernedChest{
		"gov-only": {ID: "gov-only", Path: "/some/path", Trust: treasure.GovernedTrust{Tier: "T1"}},
	}
	rows := treasure.MergeChestRows(nil, governed, nil, nil, nil)
	require.Len(t, rows, 1)
	r := rows[0]
	assert.Equal(t, "gov-only", r.ID)
	assert.True(t, r.Governed)
	assert.False(t, r.Configured)
	assert.Contains(t, r.Drift, "unscoped")
}

func TestMergeChestRows_IndexedNotDeclared(t *testing.T) {
	t.Parallel()
	indexed := map[string]bool{"idx-only": true}
	rows := treasure.MergeChestRows(nil, nil, indexed, nil, nil)
	require.Len(t, rows, 1)
	r := rows[0]
	assert.Equal(t, "idx-only", r.ID)
	assert.True(t, r.Indexed)
	assert.False(t, r.Governed)
	assert.False(t, r.Configured)
	assert.Equal(t, "unknown", r.Freshness)
	assert.Contains(t, r.Drift, "unscoped")
}

func TestMergeChestRows_FullMerge(t *testing.T) {
	t.Parallel()
	active := []treasure.ActiveChest{
		{ID: "chest-a", Path: "/a", Scope: []string{"discovery"}},
	}
	governed := map[string]treasure.GovernedChest{
		"chest-a":  {ID: "chest-a", Trust: treasure.GovernedTrust{Tier: "T1", LastReviewed: "2026-01-01"}},
		"gov-only": {ID: "gov-only", Path: "/b", Trust: treasure.GovernedTrust{Tier: "T2"}},
	}
	indexed := map[string]bool{"chest-a": true, "idx-only": true}
	compiled := map[string]bool{"chest-a": true}

	rows := treasure.MergeChestRows(active, governed, indexed, compiled, nil)
	assert.Len(t, rows, 3)

	byID := make(map[string]treasure.StatusRow)
	for _, r := range rows {
		byID[r.ID] = r
	}

	a := byID["chest-a"]
	assert.True(t, a.Configured)
	assert.True(t, a.Governed)
	assert.True(t, a.Indexed)
	assert.True(t, a.Compiled)
	assert.Equal(t, "fresh", a.Freshness)
	assert.Empty(t, a.Drift)

	govOnly := byID["gov-only"]
	assert.False(t, govOnly.Configured)
	assert.True(t, govOnly.Governed)
	assert.Contains(t, govOnly.Drift, "unscoped")

	idxOnly := byID["idx-only"]
	assert.False(t, idxOnly.Configured)
	assert.True(t, idxOnly.Indexed)
}

// --- renderIndexSection ---

func TestRenderIndexSection_CompError(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	err := renderIndexSection(w, t.TempDir(), 0, errors.New("decompress failed"))
	require.NoError(t, err)
	require.NoError(t, w.Flush())
	out := buf.String()
	assert.Contains(t, out, "corrupt")
	assert.Contains(t, out, "—")
}

func TestRenderIndexSection_Absent(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	err := renderIndexSection(w, t.TempDir(), 0, nil)
	require.NoError(t, err)
	require.NoError(t, w.Flush())
	out := buf.String()
	assert.Contains(t, out, "absent")
	assert.Contains(t, out, "—")
}

func TestRenderIndexSection_OK(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	err := renderIndexSection(w, t.TempDir(), 1700000000, nil)
	require.NoError(t, err)
	require.NoError(t, w.Flush())
	out := buf.String()
	assert.Contains(t, out, "ok")
	assert.Contains(t, out, "UTC")
}

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

// --- treasure-chest integration: treasure.LoadGoverned ---

func TestLoadGoverned_NotExist(t *testing.T) {
	t.Parallel()
	result, err := treasure.LoadGoverned(t.TempDir())
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestLoadGoverned_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := "chests:\n  - id: chest-one\n    title: Chest One\n    path: /some/path\n    trust:\n      tier: T1\n      reviewed_by: user@example.com\n      last_reviewed: '2026-01-01'\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(content), 0o644))

	result, err := treasure.LoadGoverned(dir)
	require.NoError(t, err)
	require.Contains(t, result, "chest-one")
	assert.Equal(t, "T1", result["chest-one"].Trust.Tier)
}

func TestLoadGoverned_WithValidGrade(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := "chests:\n  - id: chest-one\n    path: /some/path\n    trust:\n      tier: T1\n    grade:\n      source_grade: A\n      reuse_value: high\n      implementation_status: implemented\n    open_gaps: [\"missing tests\"]\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(content), 0o644))

	result, err := treasure.LoadGoverned(dir)
	require.NoError(t, err)
	require.Contains(t, result, "chest-one")
	assert.Equal(t, "A", result["chest-one"].Grade.SourceGrade)
	assert.Equal(t, "high", result["chest-one"].Grade.ReuseValue)
	assert.Equal(t, []string{"missing tests"}, result["chest-one"].OpenGaps)
}

func TestLoadGoverned_WithInvalidGrade(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := "chests:\n  - id: chest-one\n    path: /some/path\n    trust:\n      tier: T1\n    grade:\n      source_grade: Z\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(content), 0o644))

	_, err := treasure.LoadGoverned(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "source_grade")
}

// --- initiative: readLastMissionID via writeWorkspaceSection ---

func TestInitiativeCmd_WithOutcomesFile(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	// Write a memory/outcomes.jsonl file with a known mission_id.
	memDir := filepath.Join(dir, "memory")
	require.NoError(t, os.MkdirAll(memDir, 0o755))
	line := `{"mission_id":"m-test-123","status":"done"}`
	require.NoError(t, os.WriteFile(filepath.Join(memDir, "outcomes.jsonl"), []byte(line+"\n"), 0o644))

	orig := initiativeRoot
	t.Cleanup(func() { initiativeRoot = orig })
	initiativeRoot = dir

	out := captureStdout(t, func() {
		require.NoError(t, initiativeCmd.RunE(initiativeCmd, nil))
	})
	assert.Contains(t, out, "m-test-123")
}

// --- treasure.LoadGoverned ---

func TestLoadGoverned_CorruptYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"),
		[]byte(": not: valID: yaml:\n"), 0o644))

	_, err := treasure.LoadGoverned(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "treasure-chests.yaml")
}

// --- collectWarnings ---

func TestCollectWarnings_AllWarnings(t *testing.T) {
	t.Parallel()
	rows := []treasure.StatusRow{
		{ID: "drift-chest", Configured: true, Governed: false, Indexed: false,
			Drift: []string{"missing_governance", "missing_index"}},
		{ID: "historical-chest", TrustTier: "T2", LastReviewed: ""},
	}
	warnings := collectWarnings(rows,
		errors.New("gov load error"),
		errors.New("idx load error"),
		errors.New("corrupt"),
		0,
	)
	assert.Contains(t, strings.Join(warnings, "\n"), "treasure-chests.yaml unavailable")
	assert.Contains(t, strings.Join(warnings, "\n"), "knowledge.index.yaml unavailable")
	assert.Contains(t, strings.Join(warnings, "\n"), "corrupt")
	assert.Contains(t, strings.Join(warnings, "\n"), "drift detected")
	assert.Contains(t, strings.Join(warnings, "\n"), "historical")
}

func TestCollectWarnings_AbsentIndex(t *testing.T) {
	t.Parallel()
	warnings := collectWarnings(nil, nil, nil, nil, 0)
	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "absent")
}

func TestCollectWarnings_DriftWithCompiledAt(t *testing.T) {
	t.Parallel()
	rows := []treasure.StatusRow{
		{ID: "chest-a", Configured: true, Governed: false,
			Drift: []string{"missing_governance"}},
	}
	warnings := collectWarnings(rows, nil, nil, nil, 1700000000)
	combined := strings.Join(warnings, "\n")
	assert.Contains(t, combined, "drift detected")
	assert.Contains(t, combined, "→ run: strategist treasure-chest --index to refresh")
}

// --- renderWarningsSection ---

func TestRenderWarningsSection_Empty(t *testing.T) {
	out := captureStdout(t, func() {
		renderWarningsSection(nil)
	})
	assert.Empty(t, out)
}

func TestRenderWarningsSection_WithWarnings(t *testing.T) {
	out := captureStdout(t, func() {
		renderWarningsSection([]string{"⚠ first warning", "⚠ second warning"})
	})
	assert.Contains(t, out, "WARNINGS")
	assert.Contains(t, out, "first warning")
	assert.Contains(t, out, "second warning")
}

// --- renderChestsSection ---

func TestRenderChestsSection_WithRows(t *testing.T) {
	t.Parallel()
	rows := []treasure.StatusRow{
		{ID: "chest-a", Path: "/path/a", Scope: []string{"discovery"}, TrustTier: "T1",
			Freshness: "fresh", Drift: nil},
		{ID: "chest-b", Scope: nil, TrustTier: "", Freshness: "unknown",
			Drift: []string{"missing_governance"}},
	}
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)
	err := renderChestsSection(w, rows)
	require.NoError(t, err)
	require.NoError(t, w.Flush())
	out := buf.String()
	assert.Contains(t, out, "chest-a")
	assert.Contains(t, out, "chest-b")
	assert.Contains(t, out, "T1")
	assert.Contains(t, out, "missing_governance")
	assert.Contains(t, out, "none") // chest-a has no drift
}

// --- writeWorkspaceSection with empty base_path ---

func TestWriteWorkspaceSection_EmptyBasePath(t *testing.T) {
	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)

	dir := t.TempDir()
	cfg := domain.ActiveConfig{
		Mode:     "full",
		BasePath: "", // triggers the defaulting branch
	}
	err := writeWorkspaceSection(w, cfg, dir, dir)
	require.NoError(t, err)
	require.NoError(t, w.Flush())
	out := buf.String()
	assert.Contains(t, out, ".analysis") // default base_path
}

// --- telemetryRunFromCmd with non-nil context ---

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

// --- treasure.LoadActiveChests with corrupt YAML ---

func TestLoadActiveChests_CorruptYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"),
		[]byte(": not: valID: yaml:\n"), 0o644))

	_, err := treasure.LoadActiveChests(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active.yaml")
}

// --- formatCount ---

func TestFormatCount_NegativeReturnsEmpty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "—", formatCount(-1, "card"))
}

func TestFormatCount_One(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1 card", formatCount(1, "card"))
	assert.Equal(t, "1 missão", formatCount(1, "missão"))
}

func TestFormatCount_Many(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "3 cards", formatCount(3, "card"))
	assert.Equal(t, "2 missões", formatCount(2, "missão"))
}
