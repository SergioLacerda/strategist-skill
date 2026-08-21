package install

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/plugins"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPluginCatalogGeneratesKnownProvidersView(t *testing.T) {
	t.Parallel()

	ext := defaultsExtractor{}
	catalog, err := loadPluginCatalog(ext)
	require.NoError(t, err)

	generated := generateKnownProvidersYAML(catalog)
	assert.Equal(t, string(generated), string(generateKnownProvidersYAML(catalog)))

	var generatedDoc, legacyDoc struct {
		Providers map[string]string `yaml:"providers"`
	}
	require.NoError(t, yaml.Unmarshal(generated, &generatedDoc))
	legacyBytes, err := ext.ReadFile(knownProvidersTemplatePath)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(legacyBytes, &legacyDoc))

	assert.Equal(t, legacyDoc.Providers, generatedDoc.Providers)
}

func TestPluginCatalogGeneratesLegacyProviderManifests(t *testing.T) {
	t.Parallel()

	ext := defaultsExtractor{}
	catalog, err := loadPluginCatalog(ext)
	require.NoError(t, err)

	for provider, legacyPath := range installableDefaultProviders {
		provider := provider
		legacyPath := legacyPath
		t.Run(provider, func(t *testing.T) {
			t.Parallel()
			generated, err := generateLegacyProviderManifest(catalog, provider)
			require.NoError(t, err)
			assert.Equal(t, string(generated), string(mustGenerateLegacyProviderManifest(t, catalog, provider)))

			legacyBytes, err := ext.ReadFile(legacyPath)
			require.NoError(t, err)

			var generatedDoc, legacyDoc map[string]any
			require.NoError(t, yaml.Unmarshal(generated, &generatedDoc))
			require.NoError(t, yaml.Unmarshal(legacyBytes, &legacyDoc))
			assert.Equal(t, legacyDoc, generatedDoc)
		})
	}
}

func TestLoadKnownProvidersPrefersPluginCatalog(t *testing.T) {
	t.Parallel()

	got := loadKnownProviders(catalogOnlyExtractor{catalog: []byte(`
schema_version: strategist-plugin-catalog/v1
providers:
  - id: zeta
    risk_score: controlled
  - id: alpha
    risk_score: write_analysis
`)})

	assert.Equal(t, map[string]string{"alpha": "write_analysis", "zeta": "controlled"}, got)
}

func TestResolveInstallableDefaultProvidersPrefersPluginCatalog(t *testing.T) {
	t.Parallel()

	got := resolveInstallableDefaultProviders(catalogOnlyExtractor{catalog: []byte(`
schema_version: strategist-plugin-catalog/v1
providers:
  - id: alpha
    risk_score: write_analysis
    installable: true
    legacy_manifest_path: skills/alpha/skill.yaml
  - id: zeta
    risk_score: controlled
`)})

	assert.Equal(t, map[string]string{"alpha": "skills/alpha/skill.yaml"}, got)
}

func TestLegacyProviderManifestBytesPrefersPluginCatalog(t *testing.T) {
	t.Parallel()

	data, err := legacyProviderManifestBytes(catalogOnlyExtractor{catalog: []byte(`
schema_version: strategist-plugin-catalog/v1
providers:
  - id: alpha
    version: "1.0.0"
    provider_schema_version: "1"
    status: active
    risk_score: write_analysis
    category: discovery
    provider_class: rankeado
    canonical_role: ranger
    description: Generated from catalog.
    installable: true
    legacy_manifest_path: skills/alpha/skill.yaml
`)}, "alpha", "skills/alpha/skill.yaml")
	require.NoError(t, err)

	var manifest map[string]any
	require.NoError(t, yaml.Unmarshal(data, &manifest))
	assert.Equal(t, "alpha", manifest["id"])
	assert.Equal(t, "write_analysis", manifest["risk_score"])
	assert.Equal(t, map[string]any{"canonical_role": "ranger", "provider_class": "rankeado"}, manifest["specialization_taxonomy"])
}

func TestPluginCatalogFeedsDeterministicResolverLock(t *testing.T) {
	t.Parallel()

	ext := defaultsExtractor{}
	catalog, err := loadPluginCatalog(ext)
	require.NoError(t, err)
	candidates := catalogResolverCandidates(catalog)

	first, err := plugins.Resolve([]plugins.Requirement{
		{ID: "brainstorming", Kind: "adapter_contract", Constraint: ">=1 <2"},
		{ID: "openspec-explore", Kind: "adapter_contract", Constraint: ">=1 <2"},
	}, candidates)
	require.NoError(t, err)
	second, err := plugins.Resolve([]plugins.Requirement{
		{ID: "openspec-explore", Kind: "adapter_contract", Constraint: ">=1 <2"},
		{ID: "brainstorming", Kind: "adapter_contract", Constraint: ">=1 <2"},
	}, reverseCatalogCandidates(candidates))
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.Equal(t, "strategist-plugin-lock/v1", first.SchemaVersion)
	assert.Len(t, first.Nodes, 2)
	assert.Regexp(t, `^sha256:[a-f0-9]{64}$`, first.GraphDigest)
	require.NoError(t, plugins.VerifyLock(first, candidates))
}

func TestPluginCatalogCandidateDigestTracksGeneratedLegacyManifest(t *testing.T) {
	t.Parallel()

	ext := defaultsExtractor{}
	catalog, err := loadPluginCatalog(ext)
	require.NoError(t, err)

	provider, ok := findCatalogProvider(catalog, "brainstorming")
	require.True(t, ok)
	generated, err := generateLegacyProviderManifest(catalog, "brainstorming")
	require.NoError(t, err)

	sum := sha256.Sum256(generated)
	assert.Equal(t, fmt.Sprintf("sha256:%x", sum), catalogProviderDigest(provider))
}

func mustGenerateLegacyProviderManifest(t *testing.T, catalog pluginCatalog, provider string) []byte {
	t.Helper()
	generated, err := generateLegacyProviderManifest(catalog, provider)
	require.NoError(t, err)
	return generated
}

func reverseCatalogCandidates(in []plugins.Candidate) []plugins.Candidate {
	out := append([]plugins.Candidate(nil), in...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

type catalogOnlyExtractor struct {
	catalog []byte
}

func (c catalogOnlyExtractor) Extract(_ string, _ bool) error { return nil }
func (c catalogOnlyExtractor) ReadFile(relPath string) ([]byte, error) {
	if relPath == pluginCatalogPath {
		return c.catalog, nil
	}
	return nil, assert.AnError
}

type defaultsExtractor struct{}

func (defaultsExtractor) Extract(_ string, _ bool) error { return nil }

func (defaultsExtractor) ReadFile(relPath string) ([]byte, error) {
	path := filepath.Join("..", "embed", "defaults", filepath.FromSlash(relPath))
	data, err := os.ReadFile(path) //nolint:gosec // test fixture path is repo-local
	if err != nil {
		return nil, fmt.Errorf("read defaults fixture %s: %w", relPath, err)
	}
	return data, nil
}
