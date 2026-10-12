package plugincatalog

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// knownProvidersExtractor serves a synthetic templates/known-providers.yaml.
type knownProvidersExtractor struct {
	yaml string
}

func (k knownProvidersExtractor) Extract(_ string, _ bool) error { return nil }
func (k knownProvidersExtractor) ReadFile(relPath string) ([]byte, error) {
	if relPath == "templates/known-providers.yaml" {
		return []byte(k.yaml), nil
	}
	return nil, fmt.Errorf("not found: %s", relPath)
}

func TestLoadKnownProviders(t *testing.T) {
	t.Parallel()

	t.Run("reads valid providers yaml", func(t *testing.T) {
		t.Parallel()
		ext := knownProvidersExtractor{yaml: "providers:\n  brainstorming: write_analysis\n  fixture-provider: controlled\n"}
		got := LoadKnownProviders(ext)
		assert.Equal(t, "write_analysis", got["brainstorming"])
		assert.Equal(t, "controlled", got["fixture-provider"])
	})

	t.Run("falls back to static map when providers map is empty", func(t *testing.T) {
		t.Parallel()
		ext := knownProvidersExtractor{yaml: "providers: {}\n"}
		got := LoadKnownProviders(ext)
		assert.Equal(t, KnownProviderRisk, got)
	})

	t.Run("falls back to static map on invalid yaml", func(t *testing.T) {
		t.Parallel()
		ext := knownProvidersExtractor{yaml: ": invalid: yaml:\n"}
		got := LoadKnownProviders(ext)
		assert.Equal(t, KnownProviderRisk, got)
	})
}
