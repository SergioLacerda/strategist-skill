package treasure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ApplyRemoveMutations ---

func TestApplyRemoveMutations_ActiveError(t *testing.T) {
	t.Parallel()
	docs := ChestDocSet{
		Active:   mustParseDoc(t, "mode: epic\n"), // no treasure_chests declared
		Governed: mustParseDoc(t, "chests: []\n"),
		Index:    mustParseDoc(t, "sources: []\n"),
	}
	err := ApplyRemoveMutations(docs, "missing")
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

func TestApplyRemoveMutations_GovernedError(t *testing.T) {
	t.Parallel()
	docs := ChestDocSet{
		Active:   mustParseDoc(t, "treasure_chests:\n  - id: a\n    path: p\n    scope: all\n"),
		Governed: mustParseDoc(t, "schema_version: \"1\"\n"), // no chests declared
		Index:    mustParseDoc(t, "sources: []\n"),
	}
	err := ApplyRemoveMutations(docs, "a")
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

func TestApplyRemoveMutations_IndexError(t *testing.T) {
	t.Parallel()
	docs := ChestDocSet{
		Active:   mustParseDoc(t, "treasure_chests:\n  - id: a\n    path: p\n    scope: all\n"),
		Governed: mustParseDoc(t, "chests:\n  - id: a\n"),
		Index:    mustParseDoc(t, "schema_version: \"1\"\n"), // no sources declared
	}
	err := ApplyRemoveMutations(docs, "a")
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

func TestApplyRemoveMutations_JewelsError(t *testing.T) {
	t.Parallel()
	docs := ChestDocSet{
		Active:   mustParseDoc(t, "treasure_chests:\n  - id: a\n    path: p\n    scope: all\n"),
		Governed: mustParseDoc(t, "chests:\n  - id: a\n"),
		Index:    mustParseDoc(t, "sources:\n  - id: a\n"),
		Jewels:   []YAMLWrite{{Path: "jewels.yaml", Doc: mustParseDoc(t, "- a\n")}}, // not a mapping -> rootMapping error
	}
	err := ApplyRemoveMutations(docs, "a")
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

func TestApplyRemoveMutations_Success(t *testing.T) {
	t.Parallel()
	docs := ChestDocSet{
		Active:   mustParseDoc(t, "treasure_chests:\n  - id: a\n    path: p\n    scope: all\n"),
		Governed: mustParseDoc(t, "chests:\n  - id: a\n"),
		Index:    mustParseDoc(t, "sources:\n  - id: a\n"),
	}
	err := ApplyRemoveMutations(docs, "a")
	require.NoError(t, err)
}

// --- LoadRemoveDocs ---

func TestLoadRemoveDocs_ActiveMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := NewChestPaths(dir)
	_, err := LoadRemoveDocs(p)
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

func TestLoadRemoveDocs_GovernedMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := NewChestPaths(dir)
	require.NoError(t, os.WriteFile(p.Active, []byte("mode: epic\n"), 0o644))
	_, err := LoadRemoveDocs(p)
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

func TestLoadRemoveDocs_IndexMissing(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := NewChestPaths(dir)
	require.NoError(t, os.WriteFile(p.Active, []byte("mode: epic\n"), 0o644))
	require.NoError(t, os.WriteFile(p.Governed, []byte("chests: []\n"), 0o644))
	_, err := LoadRemoveDocs(p)
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

func TestLoadRemoveDocs_JewelsPresent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := NewChestPaths(dir)
	require.NoError(t, os.WriteFile(p.Active, []byte("mode: epic\n"), 0o644))
	require.NoError(t, os.WriteFile(p.Governed, []byte("chests: []\n"), 0o644))
	require.NoError(t, os.WriteFile(p.Index, []byte("sources: []\n"), 0o644))
	require.NoError(t, os.WriteFile(p.Jewels, []byte("jewels: []\n"), 0o644))

	docs, err := LoadRemoveDocs(p)
	require.NoError(t, err)
	assert.NotEmpty(t, docs.Jewels)
}

func TestLoadRemoveDocs_JewelsAbsentIsNotError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := NewChestPaths(dir)
	require.NoError(t, os.WriteFile(p.Active, []byte("mode: epic\n"), 0o644))
	require.NoError(t, os.WriteFile(p.Governed, []byte("chests: []\n"), 0o644))
	require.NoError(t, os.WriteFile(p.Index, []byte("sources: []\n"), 0o644))

	docs, err := LoadRemoveDocs(p)
	require.NoError(t, err)
	assert.Empty(t, docs.Jewels)
}

// --- ResolveRemoveTarget ---

func TestResolveRemoveTarget_NoPathReturnsIDFlag(t *testing.T) {
	t.Parallel()
	id, err := ResolveRemoveTarget(t.TempDir(), "", "flag-id")
	require.NoError(t, err)
	assert.Equal(t, "flag-id", id)
}

func TestResolveRemoveTarget_LoadActiveChestsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(": not: valID: yaml:\n"), 0o644))
	_, err := ResolveRemoveTarget(dir, "some/path", "")
	require.Error(t, err)
	assert.NotEmpty(t, err.Error())
}

func TestResolveRemoveTarget_NoMatchesFallsBackToIDFlag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
treasure_chests:
  - id: source
    path: .sdd/source
    scope: all
`), 0o644))
	id, err := ResolveRemoveTarget(dir, "/no/such/path", "flag-id")
	require.NoError(t, err)
	assert.Equal(t, "flag-id", id)
}

func TestResolveRemoveTarget_NoMatchesNoIDFlagErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
treasure_chests:
  - id: source
    path: .sdd/source
    scope: all
`), 0o644))
	_, err := ResolveRemoveTarget(dir, "/no/such/path", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no chest registered")
}

func TestResolveRemoveTarget_MultipleMatchesIsAmbiguous(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "active.yaml"), []byte(`
mode: epic
base_path: .analysis
slots:
  discovery: brainstorming
  refinement: openspec-explore
  execution: sdd-ask
treasure_chests:
  - id: source-1
    path: .sdd/source
    scope: all
  - id: source-2
    path: .sdd/source
    scope: all
`), 0o644))

	_, err := ResolveRemoveTarget(dir, ".sdd/source", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous")
	assert.Contains(t, err.Error(), "source-1")
	assert.Contains(t, err.Error(), "source-2")
}
