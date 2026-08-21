package embed_test

import (
	"testing"

	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestExtractor_ReadsPluginResourceSchemas(t *testing.T) {
	t.Parallel()

	schemas := []string{
		"plugins/schemas/package.schema.yaml",
		"plugins/schemas/adapter.schema.yaml",
		"plugins/schemas/trust-policy.schema.yaml",
		"plugins/schemas/inventory.schema.yaml",
		"plugins/schemas/binding.schema.yaml",
		"plugins/schemas/grant.schema.yaml",
		"plugins/schemas/lock.schema.yaml",
		"plugins/schemas/transaction.schema.yaml",
		"plugins/schemas/certification.schema.yaml",
	}

	for _, schemaPath := range schemas {
		schemaPath := schemaPath
		t.Run(schemaPath, func(t *testing.T) {
			t.Parallel()
			data, err := embedpkg.Extractor{}.ReadFile(schemaPath)
			require.NoError(t, err)

			var doc map[string]any
			require.NoError(t, yaml.Unmarshal(data, &doc))
			assert.Equal(t, "strategist-plugin-resource-schema/v1", doc["schema"])
			assert.NotEmpty(t, doc["resource"])
			assert.NotEmpty(t, doc["authority"])
			assert.Contains(t, string(data), "additional_properties: false")
			assert.Contains(t, string(data), "max_document_bytes: 262144")
		})
	}
}
