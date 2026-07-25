package treasure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- LoadChestYAMLDocs ---

func TestLoadChestYAMLDocs_ActiveMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, _, _, err := LoadChestYAMLDocs(filepath.Join(dir, "active.yaml"), filepath.Join(dir, "gov.yaml"), filepath.Join(dir, "idx.yaml"))
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

func TestLoadChestYAMLDocs_GovernedMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	activePath := filepath.Join(dir, "active.yaml")
	require.NoError(t, os.WriteFile(activePath, []byte("mode: epic\n"), 0o644))
	_, _, _, err := LoadChestYAMLDocs(activePath, filepath.Join(dir, "gov.yaml"), filepath.Join(dir, "idx.yaml"))
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

func TestLoadChestYAMLDocs_IndexMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	activePath := filepath.Join(dir, "active.yaml")
	governedPath := filepath.Join(dir, "gov.yaml")
	require.NoError(t, os.WriteFile(activePath, []byte("mode: epic\n"), 0o644))
	require.NoError(t, os.WriteFile(governedPath, []byte("chests: []\n"), 0o644))
	_, _, _, err := LoadChestYAMLDocs(activePath, governedPath, filepath.Join(dir, "idx.yaml"))
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

func TestLoadChestYAMLDocs_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	activePath := filepath.Join(dir, "active.yaml")
	governedPath := filepath.Join(dir, "gov.yaml")
	indexPath := filepath.Join(dir, "idx.yaml")
	require.NoError(t, os.WriteFile(activePath, []byte("mode: epic\n"), 0o644))
	require.NoError(t, os.WriteFile(governedPath, []byte("chests: []\n"), 0o644))
	require.NoError(t, os.WriteFile(indexPath, []byte("sources: []\n"), 0o644))

	activeDoc, governedDoc, indexDoc, err := LoadChestYAMLDocs(activePath, governedPath, indexPath)
	require.NoError(t, err)
	assert.NotNil(t, activeDoc)
	assert.NotNil(t, governedDoc)
	assert.NotNil(t, indexDoc)
}

// --- ApplyAddMutations ---

func TestApplyAddMutations_ActiveMappingError(t *testing.T) {
	t.Parallel()
	activeDoc := mustParseDoc(t, "- a\n")
	governedDoc := mustParseDoc(t, "chests: []\n")
	indexDoc := mustParseDoc(t, "sources: []\n")

	err := ApplyAddMutations(activeDoc, governedDoc, indexDoc, "id", "path", "all", "T1", "human", []string{"all"})
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

func TestApplyAddMutations_GovernedMappingError(t *testing.T) {
	t.Parallel()
	activeDoc := mustParseDoc(t, "mode: epic\n")
	governedDoc := mustParseDoc(t, "- a\n")
	indexDoc := mustParseDoc(t, "sources: []\n")

	err := ApplyAddMutations(activeDoc, governedDoc, indexDoc, "id", "path", "all", "T1", "human", []string{"all"})
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

func TestApplyAddMutations_IndexMappingError(t *testing.T) {
	t.Parallel()
	activeDoc := mustParseDoc(t, "mode: epic\n")
	governedDoc := mustParseDoc(t, "chests: []\n")
	indexDoc := mustParseDoc(t, "- a\n")

	err := ApplyAddMutations(activeDoc, governedDoc, indexDoc, "id", "path", "all", "T1", "human", []string{"all"})
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

func TestApplyAddMutations_Success(t *testing.T) {
	t.Parallel()
	activeDoc := mustParseDoc(t, "mode: epic\n")
	governedDoc := mustParseDoc(t, "chests: []\n")
	indexDoc := mustParseDoc(t, "sources: []\n")

	err := ApplyAddMutations(activeDoc, governedDoc, indexDoc, "id", "path", "all", "T1", "human", []string{"all"})
	require.NoError(t, err)
}

// --- CheckChestIDAvailable ---

func TestCheckChestIDAvailable_LoadActiveChestsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(": not: valID: yaml:\n"), 0o644))
	err := CheckChestIDAvailable(dir, "any-id")
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

// --- ParseTagsFlag ---

func TestParseTagsFlag_AllPartsBlankAfterTrim(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"all"}, ParseTagsFlag(" , , "))
}

func TestParseTagsFlag(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []string{"all"}, ParseTagsFlag(""))
	assert.Equal(t, []string{"foo", "bar"}, ParseTagsFlag("foo, bar"))
}

// --- DeriveChestIDFromPath ---

func TestDeriveChestIDFromPath(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "docs", DeriveChestIDFromPath("/abs/path/to/docs"))
	assert.Equal(t, "docs", DeriveChestIDFromPath("/abs/path/to/docs/"))
	assert.Equal(t, "relative", DeriveChestIDFromPath("relative"))
}
