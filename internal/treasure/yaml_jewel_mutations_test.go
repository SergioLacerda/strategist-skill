package treasure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

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
