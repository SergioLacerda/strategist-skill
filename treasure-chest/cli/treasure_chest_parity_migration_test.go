package treasurecli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreasureChestBinary_AddReplayIsStableAndPreservesComments(t *testing.T) {
	bin := buildStrategistForParityTest(t)
	root := parityFixture(t)
	args := []string{"treasure-chest", "add", "docs/new-source", "--root", root, "--id", "extra"}

	first := runParityProcess(t, bin, args...)
	assertParitySuccess(t, first)
	afterFirst := paritySnapshot(t, root)
	assert.Contains(t, readParityFile(t, root, "active.yaml"), "preserved active comment")
	assert.Contains(t, readParityFile(t, root, "treasure-chests.yaml"), "preserved governance comment")

	second := runParityProcess(t, bin, args...)
	assert.Equal(t, 1, second.exitCode)
	assert.Equal(t, afterFirst, paritySnapshot(t, root), "duplicate add changed existing state")
}

func TestTreasureChestBinary_MigrationReplayPreservesIdentityAndProvenance(t *testing.T) {
	bin := buildStrategistForParityTest(t)
	root := parityMigrationFixture(t)
	args := []string{"treasure-chest", "items", "migrate-status", "--root", root}

	first := runParityProcess(t, bin, args...)
	assertParitySuccess(t, first)
	content := readParityFile(t, root, "jewels.yaml")
	assert.Contains(t, content, "id: legacy-jewel")
	assert.Contains(t, content, "source#legacy")
	assert.Contains(t, content, "status: accepted")

	second := runParityProcess(t, bin, args...)
	assertParitySuccess(t, second)
	assert.Contains(t, second.stdout, "nothing to migrate")
	assert.Equal(t, content, readParityFile(t, root, "jewels.yaml"))
}

func TestTreasureChestBinary_RemoveFailurePreservesLegacyData(t *testing.T) {
	bin := buildStrategistForParityTest(t)
	root := parityFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(root, "jewels.yaml"), []byte("jewels: [unterminated\n"), 0o644))
	before := paritySnapshot(t, root)

	result := runParityProcess(t, bin, "treasure-chest", "remove", "--root", root, "--id", "source")
	assert.Equal(t, 1, result.exitCode)
	assert.Equal(t, before, paritySnapshot(t, root), "failed remove partially mutated the fixture")
}
