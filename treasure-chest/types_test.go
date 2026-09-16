package treasure

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestScope_UnmarshalYAML_Scalar(t *testing.T) {
	t.Parallel()
	var s Scope
	require.NoError(t, yaml.Unmarshal([]byte("all"), &s))
	assert.Equal(t, Scope{"all"}, s)
}

func TestScope_UnmarshalYAML_Sequence(t *testing.T) {
	t.Parallel()
	var s Scope
	require.NoError(t, yaml.Unmarshal([]byte("[discovery, refinement]"), &s))
	assert.Equal(t, Scope{"discovery", "refinement"}, s)
}

func TestScope_UnmarshalYAML_DecodeErrorPropagates(t *testing.T) {
	t.Parallel()
	var s Scope
	// A mapping node is neither a scalar nor a []string-decodable sequence.
	err := yaml.Unmarshal([]byte("discovery: true\n"), &s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode scope")
}
