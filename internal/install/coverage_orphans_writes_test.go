package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrphanEntriesClassifyLegacyLayoutAndOrphans(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	embedded := map[string]string{"skills/brainstorming@1.0.0/SKILL.md": "h", "roles/default.yaml": "h"}

	legacyPath := "skills/brainstorming/SKILL.md"
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "skills", "brainstorming"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, filepath.FromSlash(legacyPath)), []byte("legacy"), 0o644))
	hash := domain.SHA256Hex([]byte("legacy"))

	manifest := domain.InstallManifest{Files: []domain.InstallManifestFile{
		{Path: "roles/default.yaml", SHA256: "x"},
		{Path: legacyPath, SHA256: hash},
		{Path: "skills/brainstorming/edited.md", SHA256: hash},
		{Path: "retired/file.md", SHA256: "x"},
	}}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "skills", "brainstorming", "edited.md"), []byte("user edit"), 0o644))

	none, err := orphanEntries(dir, manifest, false, embedded)
	require.NoError(t, err)
	assert.Nil(t, none)

	entries, err := orphanEntries(dir, manifest, true, embedded)
	require.NoError(t, err)
	states := map[string]domain.RuntimeFileUpgradeState{}
	for _, entry := range entries {
		states[entry.Path] = entry.State
	}
	assert.Equal(t, domain.UpgradeLegacyLayout, states[legacyPath], "an untouched legacy file is removable")
	assert.Equal(t, domain.UpgradeOrphaned, states["retired/file.md"])
	assert.NotContains(t, states, "roles/default.yaml")
	assert.NotEqual(t, domain.UpgradeLegacyLayout, states["skills/brainstorming/edited.md"], "a user-edited legacy file is kept")

	assert.False(t, isLegacyLayoutPath("skills/brainstorming@1.0.0/SKILL.md", map[string]bool{"brainstorming": true}))
	assert.False(t, isLegacyLayoutPath("skills/brainstorming/skill.yaml", map[string]bool{"brainstorming": true}))
	assert.False(t, isLegacyLayoutPath("short", nil))
}

func TestLockSlotsNeedingMigration(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	slots, err := lockSlotsNeedingMigration(dir)
	require.NoError(t, err)
	assert.Empty(t, slots)

	require.NoError(t, writePluginLockFile(dir, domain.PluginLockFile{Bindings: []domain.SlotBinding{
		{Slot: "discovery", Mode: domain.SlotBindingModeRanked},
		{Slot: "refinement", Mode: domain.SlotBindingModeRanked, WeaponVersion: "1.0.0"},
		{Slot: "execution"},
	}}))
	slots, err = lockSlotsNeedingMigration(dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"discovery"}, slots)

	broken := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(broken, pluginLockFileName), []byte("bindings: [unclosed"), 0o644))
	_, err = lockSlotsNeedingMigration(broken)
	require.ErrorContains(t, err, "upgrade: read lock")
}

func TestAtomicWriteFileFailures(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	blocker := filepath.Join(dir, "file")
	require.NoError(t, os.WriteFile(blocker, nil, 0o644))
	require.ErrorContains(t, atomicWriteFile(filepath.Join(blocker, "x.yaml"), []byte("x"), 0o644), "mkdir parent")

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "target", "child"), 0o755))
	require.ErrorContains(t, atomicWriteFile(filepath.Join(dir, "target"), []byte("x"), 0o644), "rename into place")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, entry := range entries {
		assert.NotContains(t, entry.Name(), ".tmp-", "the temporary file is cleaned up")
	}

	require.NoError(t, atomicWriteFile(filepath.Join(dir, "ok.yaml"), []byte("x"), 0o600))
}
