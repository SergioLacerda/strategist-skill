package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// doctorTestRoot builds a minimal .strategist/ tree with all three registry truth layers
// present and initially consistent (no chests declared anywhere).
func doctorTestRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	testutil.MinimalRoot(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
chests: []
`), 0o644))
	return dir
}

func TestTreasureChestDoctor_NoDriftReportsClean(t *testing.T) {
	dir := doctorTestRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: full
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
treasure_chests:
  - id: docs
    path: docs/
    scope: [all]
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
chests:
  - id: docs
    title: docs
    path: docs/
    trust:
      tier: T1
      reviewed_by: human
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "knowledge.index.yaml"), []byte(`
sources:
  - id: docs
    path: docs/
    tags: [all]
`), 0o644))
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	out := captureStdout(t, func() {
		require.NoError(t, treasureChestDoctorCmd.RunE(treasureChestDoctorCmd, nil))
	})
	assert.Contains(t, out, "no consistency drift found")
}

func TestTreasureChestDoctor_DetectsChestMissingFromIndex(t *testing.T) {
	dir := doctorTestRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: full
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
treasure_chests:
  - id: docs
    path: docs/
    scope: [all]
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "treasure-chests.yaml"), []byte(`
schema_version: "1"
chests:
  - id: docs
    title: docs
    path: docs/
    trust:
      tier: T1
      reviewed_by: human
`), 0o644))
	// knowledge.index.yaml intentionally omits the "docs" source — simulated divergence.
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	var err error
	errOut := captureStderr(t, func() {
		err = treasureChestDoctorCmd.RunE(treasureChestDoctorCmd, nil)
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "consistency drift in 1 chest")
	assert.Contains(t, errOut, "docs: present in active.yaml, treasure-chests.yaml; absent from knowledge.index.yaml")
}

func TestTreasureChestDoctor_IsReadOnly(t *testing.T) {
	dir := doctorTestRoot(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: full
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
treasure_chests:
  - id: orphan
    path: docs/
    scope: [all]
`), 0o644))
	resetTreasureChestFlags(t)
	setTreasureChestRoot(t, dir)

	beforeActive, err := os.ReadFile(filepath.Join(dir, "active.yaml"))
	require.NoError(t, err)
	beforeGoverned, err := os.ReadFile(filepath.Join(dir, "treasure-chests.yaml"))
	require.NoError(t, err)
	beforeIndex, err := os.ReadFile(filepath.Join(dir, "knowledge.index.yaml"))
	require.NoError(t, err)

	captureStderr(t, func() {
		_ = treasureChestDoctorCmd.RunE(treasureChestDoctorCmd, nil) //nolint:errcheck // error expected; only disk state matters here
	})

	afterActive, err := os.ReadFile(filepath.Join(dir, "active.yaml"))
	require.NoError(t, err)
	afterGoverned, err := os.ReadFile(filepath.Join(dir, "treasure-chests.yaml"))
	require.NoError(t, err)
	afterIndex, err := os.ReadFile(filepath.Join(dir, "knowledge.index.yaml"))
	require.NoError(t, err)

	assert.Equal(t, string(beforeActive), string(afterActive))
	assert.Equal(t, string(beforeGoverned), string(afterGoverned))
	assert.Equal(t, string(beforeIndex), string(afterIndex))
}
