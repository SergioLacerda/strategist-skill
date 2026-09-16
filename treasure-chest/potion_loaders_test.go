package treasure

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const onePotionYAML = `
schema_version: "1"
potions:
  - id: potion-1
    chest_id: runbooks
    runbook_ref: docs/runbooks/sample.md
    when_to_use: "When sample breaks."
    trust: T2
    status: proposed
    source_refs: ["docs/runbooks/sample.md"]
    reviewed_by: agent
`

func writePotionsFileT(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "potions.yaml"), []byte(content), 0o644))
}

// --- loading / validation ---

func TestLoadPotions_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, onePotionYAML)

	got, err := LoadPotions(dir, nil)
	require.NoError(t, err)
	require.Len(t, got["runbooks"], 1)
	assert.Equal(t, "potion-1", got["runbooks"][0].ID)
}

func TestLoadPotions_MissingFileIsNotError(t *testing.T) {
	t.Parallel()
	got, err := LoadPotions(t.TempDir(), nil)
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestLoadPotions_MissingRunbookRefErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, `
schema_version: "1"
potions:
  - id: potion-1
    chest_id: runbooks
    when_to_use: "When sample breaks."
    trust: T2
    status: proposed
    source_refs: ["docs/runbooks/sample.md"]
    reviewed_by: agent
`)

	_, err := LoadPotions(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing runbook_ref")
}

func TestLoadPotions_TrustExceedsChestTierErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, onePotionYAML)
	governed := map[string]GovernedChest{
		"runbooks": {ID: "runbooks", Trust: GovernedTrust{Tier: "T3"}},
	}

	_, err := LoadPotions(dir, governed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds parent chest's trust tier")
}

func TestLoadPotions_UnsupportedSchemaVersionErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, `
schema_version: "2"
potions: []
`)
	_, err := LoadPotions(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported schema_version")
}

func TestLoadPotions_MissingChestIDErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writePotionsFileT(t, dir, `
schema_version: "1"
potions:
  - id: potion-1
    runbook_ref: docs/runbooks/sample.md
    when_to_use: "When sample breaks."
    trust: T2
    status: proposed
    source_refs: ["docs/runbooks/sample.md"]
    reviewed_by: agent
`)
	_, err := LoadPotions(dir, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing chest_id")
}

func TestLoadPotions_PartitionedManifestUsed(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "potions"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "potions", "runbooks.yaml"), []byte(onePotionYAML), 0o644))

	got, err := LoadPotions(dir, nil)
	require.NoError(t, err)
	require.Len(t, got["runbooks"], 1)
}

func TestMonolithicPotionManifestPaths_StatError(t *testing.T) {
	skipIfPermissionTestUnsupported(t)
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission denial is not reliable on windows")
	}
	dir := t.TempDir()
	require.NoError(t, os.Chmod(dir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_, err := monolithicPotionManifestPaths(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stat potions.yaml")
}

func TestPartitionedPotionManifestPaths_ReadDirError(t *testing.T) {
	skipIfPermissionTestUnsupported(t)
	if runtime.GOOS == "windows" {
		t.Skip("chmod-based permission denial is not reliable on windows")
	}
	dir := t.TempDir()
	potionsDir := filepath.Join(dir, "potions")
	require.NoError(t, os.MkdirAll(potionsDir, 0o755))
	require.NoError(t, os.Chmod(potionsDir, 0o000))
	t.Cleanup(func() { _ = os.Chmod(potionsDir, 0o755) })

	_, err := partitionedPotionManifestPaths(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read potions/")
}

func TestPotionManifestLabel_RelError(t *testing.T) {
	t.Parallel()
	got := potionManifestLabel("relative/root", "/absolute/root/potions.yaml")
	assert.Equal(t, "/absolute/root/potions.yaml", got)
}
