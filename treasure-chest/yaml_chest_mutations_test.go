package treasure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- AppendActiveChestEntry / RemoveActiveChestEntry error branches ---

func TestAppendActiveChestEntry_RootMappingError(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "- a\n- b\n")
	err := AppendActiveChestEntry(doc, "id", "path", "all")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a mapping")
}

func TestRemoveActiveChestEntry_RootMappingError(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "- a\n- b\n")
	err := RemoveActiveChestEntry(doc, "id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a mapping")
}

func TestRemoveActiveChestEntry_NoSequenceDeclared(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "mode: epic\n")
	err := RemoveActiveChestEntry(doc, "id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no treasure_chests declared")
}

func TestRemoveActiveChestEntry_IDNotFound(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "treasure_chests:\n  - id: a\n    path: p\n    scope: all\n")
	err := RemoveActiveChestEntry(doc, "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in active.yaml")
}

func TestRemoveActiveChestEntry_Success(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "treasure_chests:\n  - id: a\n    path: p\n    scope: all\n  - id: b\n    path: q\n    scope: all\n")
	require.NoError(t, RemoveActiveChestEntry(doc, "a"))

	root, err := RootMapping(doc)
	require.NoError(t, err)
	seq := MappingValue(root, "treasure_chests")
	require.Len(t, seq.Content, 1)
	entry, idx := FindEntryByID(seq, "b")
	require.NotNil(t, entry)
	assert.Equal(t, 0, idx)
}

// --- AppendGovernedChestEntry / MarkGovernedChestInactive error branches ---

func TestAppendGovernedChestEntry_RootMappingError(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "- a\n")
	err := AppendGovernedChestEntry(doc, "id", "path", "T1", "human", []string{"all"})
	require.Error(t, err)
}

func TestMarkGovernedChestInactive_RootMappingError(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "- a\n")
	err := MarkGovernedChestInactive(doc, "id")
	require.Error(t, err)
}

func TestMarkGovernedChestInactive_NoChestsDeclared(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "schema_version: \"1\"\n")
	err := MarkGovernedChestInactive(doc, "id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no chests declared")
}

func TestMarkGovernedChestInactive_IDNotFound(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "chests:\n  - id: a\n")
	err := MarkGovernedChestInactive(doc, "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in treasure-chests.yaml")
}

// --- AppendIndexedSourceEntry / MarkIndexedSourceInactive error branches ---

func TestAppendIndexedSourceEntry_RootMappingError(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "- a\n")
	err := AppendIndexedSourceEntry(doc, "id", "path", []string{"all"})
	require.Error(t, err)
}

func TestMarkIndexedSourceInactive_RootMappingError(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "- a\n")
	err := MarkIndexedSourceInactive(doc, "id")
	require.Error(t, err)
}

func TestMarkIndexedSourceInactive_NoSourcesDeclared(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "schema_version: \"1\"\n")
	err := MarkIndexedSourceInactive(doc, "id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no sources declared")
}

func TestMarkIndexedSourceInactive_IDNotFound(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "sources:\n  - id: a\n")
	err := MarkIndexedSourceInactive(doc, "missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found in knowledge.index.yaml")
}
