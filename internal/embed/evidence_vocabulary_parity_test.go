package embed_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func readSchema(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := embed.Extractor{}.ReadFile(path)
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &doc))
	return doc
}

// dig walks nested mappings by key.
func dig(t *testing.T, doc map[string]any, keys ...string) any {
	t.Helper()
	var node any = doc
	for _, key := range keys {
		m, ok := node.(map[string]any)
		require.True(t, ok, "missing mapping before %q", key)
		node = m[key]
	}
	return node
}

func stringList(t *testing.T, value any) []string {
	t.Helper()
	items, ok := value.([]any)
	require.True(t, ok, "want a list, got %T", value)
	out := make([]string, len(items))
	for i, item := range items {
		out[i], ok = item.(string)
		require.True(t, ok)
	}
	return out
}

// Agents write evidence classes into the handoff packages, so the handoff
// schemas must declare the same vocabulary the confidence recorder accepts;
// without it agents invented `source`, which the recorder rejects.
func TestEvidenceClassVocabularyMatchesTheRecorder(t *testing.T) {
	want := domain.EvidenceClasses()
	evidence := readSchema(t, "schemas/evidence.schema.yaml")
	assert.Equal(t, want, stringList(t, dig(t, evidence, "fields", "class", "values")), "evidence.schema.yaml")

	for _, path := range []string{"schemas/handoff-ranger-to-archivist.schema.yaml", "schemas/handoff-archivist-to-sniper.schema.yaml"} {
		schema := readSchema(t, path)
		summary := []string{"required_fields", "confidence_summary", "fields"}
		assert.Equal(t, want, stringList(t, dig(t, schema, append(summary, "claims", "item_values", "evidence_classes")...)), "%s claims", path)
		assert.Equal(t, want, stringList(t, dig(t, schema, append(summary, "evidence", "item_values", "class")...)), "%s evidence", path)
	}
}
