package install

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		ext := knownProvidersExtractor{yaml: "providers:\n  brainstorming: write_analysis\n  sdd-ask: controlled\n"}
		got := loadKnownProviders(ext)
		assert.Equal(t, "write_analysis", got["brainstorming"])
		assert.Equal(t, "controlled", got["sdd-ask"])
	})

	t.Run("falls back to static map when providers map is empty", func(t *testing.T) {
		t.Parallel()
		ext := knownProvidersExtractor{yaml: "providers: {}\n"}
		got := loadKnownProviders(ext)
		assert.Equal(t, knownProviderRisk, got)
	})

	t.Run("falls back to static map on invalid yaml", func(t *testing.T) {
		t.Parallel()
		ext := knownProvidersExtractor{yaml: ": invalid: yaml:\n"}
		got := loadKnownProviders(ext)
		assert.Equal(t, knownProviderRisk, got)
	})
}

// partialExtractor serves minimalExtractor files but fails for the given
// path(s). failPaths supersedes the legacy single failPath field when set.
type partialExtractor struct {
	failPath  string
	failPaths map[string]bool
}

func (p partialExtractor) Extract(targetDir string, overwrite bool) error {
	return minimalExtractor{}.Extract(targetDir, overwrite)
}
func (p partialExtractor) ReadFile(relPath string) ([]byte, error) {
	if relPath == p.failPath || p.failPaths[relPath] {
		return nil, fmt.Errorf("partialExtractor: injected failure for %s", relPath)
	}
	return minimalExtractor{}.ReadFile(relPath)
}

// TestWriteSelectedProviderManifests_ReadFileFails exercises
// writeSelectedProviderManifest's legacy fallback branch directly
// (legacyProviderManifestBytes' extractor.ReadFile(fallbackPath) call),
// bypassing runWizard: since docs/adr/0035-embedded-weapon-fallback-policy.md,
// runWizard itself hard-blocks whenever plugins/catalog.yaml is unreadable,
// so going through the full wizard can no longer reach this manifest-write
// fallback with a failing catalog — the catalog load that gates runWizard
// and the one legacyProviderManifestBytes performs share the same
// extractor and would both fail together, never one after the other.
func TestWriteSelectedProviderManifests_ReadFileFails(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ext := partialExtractor{failPaths: map[string]bool{
		pluginCatalogPath:                 true, // forces the legacy-fallback branch
		"skills/brainstorming/skill.yaml": true, // then fails the fallback read itself
	}}
	svc := Service{Extractor: ext, Compiler: nopCompiler{}, ShimHomeDir: t.TempDir()}
	err := svc.writeSelectedProviderManifest(dir, "brainstorming")
	require.Error(t, err)
	assert.ErrorContains(t, err, "brainstorming")
}
