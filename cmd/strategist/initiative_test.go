package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/tabwriter"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/testutil"
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
