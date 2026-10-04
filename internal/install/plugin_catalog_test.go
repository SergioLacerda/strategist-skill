package install

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/tools/resolver"
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

func TestLoadKnownProvidersPrefersPluginCatalog(t *testing.T) {
	t.Parallel()

	got := loadKnownProviders(catalogOnlyExtractor{catalog: []byte(`
schema_version: strategist-plugin-catalog/v2
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

	got, err := resolveInstallableDefaultProviders(catalogOnlyExtractor{catalog: []byte(`
schema_version: strategist-plugin-catalog/v2
providers:
  - id: alpha
    risk_score: write_analysis
    installable: true
    legacy_manifest_path: skills/alpha/skill.yaml
  - id: zeta
    risk_score: controlled
`)})

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"alpha": "alpha"}, got)
}

// Installable detection keys on `installable: true`, not on legacy_manifest_path: an
// entry that no longer carries the legacy path is still installable.
func TestInstallableDetectionDoesNotDependOnLegacyManifestPath(t *testing.T) {
	t.Parallel()

	got, err := resolveInstallableDefaultProviders(catalogOnlyExtractor{catalog: []byte(`
schema_version: strategist-plugin-catalog/v2
providers:
  - id: no-legacy-path
    risk_score: write_analysis
    installable: true
  - id: not-installable
    risk_score: write_analysis
    legacy_manifest_path: skills/not-installable/skill.yaml
`)})

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"no-legacy-path": "no-legacy-path"}, got)
}

func TestPluginCatalogRejectsUnverifiedRoleAffinity(t *testing.T) {
	_, err := parseCatalogBytes([]byte(`
schema_version: strategist-plugin-catalog/v2
providers:
  - id: draft-role-weapon
    risk_score: write_analysis
    canonical_role: pathfinder
`))
	require.ErrorContains(t, err, "not approved for activation")
}

// TestResolveInstallableDefaultProvidersPropagatesCatalogError asserts SQ-2's
// hardening (ADR-0035 Decision 2, no fallback substitution): a
// loadPluginCatalog failure must return an error, not silently substitute
// installableDefaultProviders.
func TestResolveInstallableDefaultProvidersPropagatesCatalogError(t *testing.T) {
	t.Parallel()

	got, err := resolveInstallableDefaultProviders(partialExtractor{failPath: pluginCatalogPath})

	require.Error(t, err)
	require.ErrorContains(t, err, "resolve installable default providers")
	assert.Nil(t, got)
}

func TestPluginCatalogFeedsDeterministicResolverLock(t *testing.T) {
	t.Parallel()

	ext := defaultsExtractor{}
	catalog, err := loadPluginCatalog(ext)
	require.NoError(t, err)
	candidates := catalogResolverCandidates(catalog)

	first, err := resolver.Resolve([]resolver.Requirement{
		{ID: "brainstorming", Kind: "adapter_contract", Constraint: ">=1 <2"},
		{ID: "openspec-explore", Kind: "adapter_contract", Constraint: ">=1 <2"},
	}, candidates)
	require.NoError(t, err)
	second, err := resolver.Resolve([]resolver.Requirement{
		{ID: "openspec-explore", Kind: "adapter_contract", Constraint: ">=1 <2"},
		{ID: "brainstorming", Kind: "adapter_contract", Constraint: ">=1 <2"},
	}, reverseCatalogCandidates(candidates))
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.Equal(t, "strategist-plugin-lock/v1", first.SchemaVersion)
	assert.Len(t, first.Nodes, 2)
	assert.Regexp(t, `^sha256:[a-f0-9]{64}$`, first.GraphDigest)
	require.NoError(t, resolver.VerifyLock(first, candidates))
}

func TestPluginCatalogCandidateDigestTracksNormalizedManifest(t *testing.T) {
	t.Parallel()

	ext := defaultsExtractor{}
	catalog, err := loadPluginCatalog(ext)
	require.NoError(t, err)

	provider, ok := findCatalogProvider(catalog, "brainstorming")
	require.True(t, ok)
	generated, err := normalizedDigestManifest(catalog, "brainstorming")
	require.NoError(t, err)

	sum := sha256.Sum256(generated)
	assert.Equal(t, fmt.Sprintf("sha256:%x", sum), catalogProviderDigest(provider))
}

func reverseCatalogCandidates(in []resolver.Candidate) []resolver.Candidate {
	out := append([]resolver.Candidate(nil), in...)
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
