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

// --- providers ---

func TestInitiativeCmd_ShowsAllSlots(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	orig := initiativeRoot
	t.Cleanup(func() { initiativeRoot = orig })
	initiativeRoot = dir

	out := captureStdout(t, func() {
		require.NoError(t, initiativeCmd.RunE(initiativeCmd, nil))
	})
	assert.Contains(t, out, "discovery")
	assert.Contains(t, out, "brainstorming")
	assert.Contains(t, out, "refinement")
	assert.Contains(t, out, "openspec-explore")
	assert.Contains(t, out, "execution")
	assert.Contains(t, out, "sdd-ask")
	// no manifests in minimal root → all show absent
	assert.Contains(t, out, "⚠ manifest ausente")
}

func TestInitiativeCmd_ShowsManifestOK(t *testing.T) {
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)

	// write a minimal provider manifest for brainstorming
	provDir := filepath.Join(dir, "skills", "brainstorming")
	require.NoError(t, os.MkdirAll(provDir, 0o755))
	manifest := []byte("id: brainstorming\nstatus: active\nrisk_score: write_analysis\nprovider_class: rankeado\nspecialization_taxonomy:\n  canonical_role: ranger\n  provider_class: rankeado\n")
	require.NoError(t, os.WriteFile(filepath.Join(provDir, "skill.yaml"), manifest, 0o644))

	orig := initiativeRoot
	t.Cleanup(func() { initiativeRoot = orig })
	initiativeRoot = dir

	out := captureStdout(t, func() {
		require.NoError(t, initiativeCmd.RunE(initiativeCmd, nil))
	})
	assert.Contains(t, out, "Ranger rankeado")
	assert.Contains(t, out, "✓ manifest OK")
}

func TestInitiativeCmd_MissingActiveYAML(t *testing.T) {
	dir := t.TempDir() // empty — no active.yaml

	orig := initiativeRoot
	t.Cleanup(func() { initiativeRoot = orig })
	initiativeRoot = dir

	err := initiativeCmd.RunE(initiativeCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active.yaml")
}

func TestInitiativeCmd_DefaultRootFallback(t *testing.T) {
	// When --root is empty, RunE auto-discovers via findStrategistRoot.
	// In a tmpdir with no .strategist/, it should return an error containing "not found".
	orig := initiativeRoot
	t.Cleanup(func() { initiativeRoot = orig })
	initiativeRoot = ""

	// Change CWD to an isolated temp dir so we don't accidentally pick up the real runtime.
	tmp := t.TempDir()
	origWd, _ := os.Getwd()
	require.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	err := initiativeCmd.RunE(initiativeCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProviderRow_FallbackRoles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir() // no manifests

	role, class, status := providerRow(dir, "discovery", "custom-ranger")
	assert.Equal(t, "Ranger", role)
	assert.Equal(t, "(base)", class)
	assert.Equal(t, "⚠ manifest ausente", status)

	role, _, _ = providerRow(dir, "refinement", "custom-arch")
	assert.Equal(t, "Archivist", role)

	role, _, _ = providerRow(dir, "execution", "custom-sniper")
	assert.Equal(t, "Sniper", role)

	role, _, _ = providerRow(dir, "custom-slot", "some-provider")
	assert.Equal(t, "Custom-slot", role) // unknown slot → title-case of slot name
}

func TestCanonicalRoleLabel(t *testing.T) {
	t.Parallel()
	cases := []struct{ input, want string }{
		{"ranger", "Ranger"},
		{"RANGER", "Ranger"},
		{"archivist", "Archivist"},
		{"Archivist", "Archivist"},
		{"sniper", "Sniper"},
		{"custom", "Custom"},
		{"", ""},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, canonicalRoleLabel(tc.input), "input=%q", tc.input)
	}
}
