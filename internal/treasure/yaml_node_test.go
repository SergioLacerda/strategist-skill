package treasure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Whitebox tests for yaml.Node-based read-modify-write helpers in treasure_chest_yaml_node.go.
// These exercise the pure functions directly with hand-built yaml.Node trees, hitting the
// error branches that the higher-level add/remove command tests don't reach (malformed
// documents, missing sequences/keys, non-mapping items, etc).

func mustParseDoc(t *testing.T, content string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte(content), &doc))
	return &doc
}

// --- ReadYAMLNode ---

func TestReadYAMLNode_MissingFile(t *testing.T) {
	t.Parallel()
	_, err := ReadYAMLNode(filepath.Join(t.TempDir(), "nope.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read")
}

func TestReadYAMLNode_InvalidYAML(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "bad.yaml")
	require.NoError(t, os.WriteFile(path, []byte(": not: valid: yaml:\n"), 0o644))
	_, err := ReadYAMLNode(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

// --- WriteYAMLNodes ---

func TestWriteYAMLNodes_WriteFailure(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "a: 1\n")
	// Directory instead of file path — os.WriteFile will fail.
	dir := t.TempDir()
	written, err := WriteYAMLNodes(YAMLWrite{Path: dir, Doc: doc})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write")
	assert.Empty(t, written)
}

func TestWriteYAMLNodes_PartialWriteReturnsWrittenPaths(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	goodPath := filepath.Join(dir, "good.yaml")
	doc := mustParseDoc(t, "a: 1\n")

	written, err := WriteYAMLNodes(
		YAMLWrite{Path: goodPath, Doc: doc},
		YAMLWrite{Path: dir, Doc: doc}, // second write fails: dir is not a regular file
	)
	require.Error(t, err)
	assert.Equal(t, []string{goodPath}, written)
}

func TestWriteYAMLNodes_Success(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	doc := mustParseDoc(t, "a: 1\n")

	written, err := WriteYAMLNodes(YAMLWrite{Path: path, Doc: doc})
	require.NoError(t, err)
	assert.Equal(t, []string{path}, written)
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "a: 1")
}

// --- RootMapping ---

func TestRootMapping_EmptyDocument(t *testing.T) {
	t.Parallel()
	doc := &yaml.Node{Kind: yaml.DocumentNode}
	_, err := RootMapping(doc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no content")
}

func TestRootMapping_NotDocumentNode(t *testing.T) {
	t.Parallel()
	doc := &yaml.Node{Kind: yaml.ScalarNode, Value: "x"}
	_, err := RootMapping(doc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no content")
}

func TestRootMapping_RootNotMapping(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "- a\n- b\n")
	_, err := RootMapping(doc)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a mapping")
}

func TestRootMapping_Success(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "a: 1\n")
	root, err := RootMapping(doc)
	require.NoError(t, err)
	assert.Equal(t, yaml.MappingNode, root.Kind)
}

// --- MappingValue ---

func TestMappingValue_KeyNotFound(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "a: 1\n")
	root, err := RootMapping(doc)
	require.NoError(t, err)
	assert.Nil(t, MappingValue(root, "missing"))
}

func TestMappingValue_KeyFound(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "a: 1\nb: 2\n")
	root, err := RootMapping(doc)
	require.NoError(t, err)
	v := MappingValue(root, "b")
	require.NotNil(t, v)
	assert.Equal(t, "2", v.Value)
}

// --- FindOrCreateSequence ---

func TestFindOrCreateSequence_CreatesWhenMissing(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "a: 1\n")
	root, err := RootMapping(doc)
	require.NoError(t, err)

	seq := FindOrCreateSequence(root, "items")
	require.NotNil(t, seq)
	assert.Equal(t, yaml.SequenceNode, seq.Kind)
	assert.Empty(t, seq.Content)

	// key was appended to the mapping
	assert.NotNil(t, MappingValue(root, "items"))
}

func TestFindOrCreateSequence_ReturnsExisting(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "items:\n  - x\n  - y\n")
	root, err := RootMapping(doc)
	require.NoError(t, err)

	seq := FindOrCreateSequence(root, "items")
	require.NotNil(t, seq)
	assert.Len(t, seq.Content, 2)
}

// --- FindEntryByID ---

func TestFindEntryByID_Found(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "items:\n  - id: a\n  - id: b\n")
	root, err := RootMapping(doc)
	require.NoError(t, err)
	seq := MappingValue(root, "items")

	entry, idx := FindEntryByID(seq, "b")
	require.NotNil(t, entry)
	assert.Equal(t, 1, idx)
}

func TestFindEntryByID_NotFound(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "items:\n  - id: a\n")
	root, err := RootMapping(doc)
	require.NoError(t, err)
	seq := MappingValue(root, "items")

	entry, idx := FindEntryByID(seq, "missing")
	assert.Nil(t, entry)
	assert.Equal(t, -1, idx)
}

func TestFindEntryByID_SkipsNonMappingItems(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "items:\n  - plain-scalar\n  - id: b\n")
	root, err := RootMapping(doc)
	require.NoError(t, err)
	seq := MappingValue(root, "items")

	entry, idx := FindEntryByID(seq, "b")
	require.NotNil(t, entry)
	assert.Equal(t, 1, idx)
}

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

// --- MarkJewelsDeprecatedForChest ---

func TestMarkJewelsDeprecatedForChest_RootMappingError(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "- a\n")
	err := MarkJewelsDeprecatedForChest(doc, "chest-id")
	require.Error(t, err)
}

func TestMarkJewelsDeprecatedForChest_NoJewelsKeyIsNoop(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "schema_version: \"1\"\n")
	require.NoError(t, MarkJewelsDeprecatedForChest(doc, "chest-id"))
}

func TestMarkJewelsDeprecatedForChest_SkipsNonMatchingAndNonMappingEntries(t *testing.T) {
	t.Parallel()
	doc := mustParseDoc(t, "jewels:\n  - plain-scalar\n  - id: j1\n    chest_id: other\n  - id: j2\n    chest_id: target\n")
	require.NoError(t, MarkJewelsDeprecatedForChest(doc, "target"))

	root, err := RootMapping(doc)
	require.NoError(t, err)
	seq := MappingValue(root, "jewels")
	other, _ := FindEntryByID(seq, "j1")
	require.NotNil(t, other)
	assert.Nil(t, MappingValue(other, "status"))

	target, _ := FindEntryByID(seq, "j2")
	require.NotNil(t, target)
	status := MappingValue(target, "status")
	require.NotNil(t, status)
	assert.Equal(t, "deprecated", status.Value)
	history := MappingValue(target, "history")
	require.NotNil(t, history)
	assert.Equal(t, yaml.SequenceNode, history.Kind)
	require.Len(t, history.Content, 1)
	assert.Equal(t, "deprecated", MappingValue(history.Content[0], "status").Value)
}
