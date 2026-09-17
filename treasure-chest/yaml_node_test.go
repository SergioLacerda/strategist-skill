package treasure

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Whitebox tests for yaml.Node-based read-modify-write helpers in yaml_node.go.
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
