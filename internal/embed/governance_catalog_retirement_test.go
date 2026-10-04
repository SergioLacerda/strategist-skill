package embed_test

import (
	"testing"

	embedpkg "github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestEmbeddedGovernanceCatalogHasNoRetiredSDDProviders(t *testing.T) {
	raw, err := (embedpkg.Extractor{}).ReadFile("plugins/catalog.yaml")
	require.NoError(t, err)
	var catalog struct {
		Providers []struct {
			ID string `yaml:"id"`
		} `yaml:"providers"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &catalog))
	for _, provider := range catalog.Providers {
		require.NotContains(t, provider.ID, "sdd-", "retired SDD provider %q remains in the embedded catalog", provider.ID)
	}
}
