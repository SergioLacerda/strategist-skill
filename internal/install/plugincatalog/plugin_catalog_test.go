package plugincatalog_test

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/install/installtest"
	"github.com/SergioLacerda/strategist-skill/internal/install/names"
	"github.com/SergioLacerda/strategist-skill/internal/install/plugincatalog"
	"github.com/SergioLacerda/strategist-skill/internal/tools/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPluginCatalogGeneratesKnownProvidersView(t *testing.T) {
	t.Parallel()

	ext := installtest.DefaultsExtractor{}
	catalog, err := plugincatalog.LoadPluginCatalog(ext)
	require.NoError(t, err)

	generated := plugincatalog.GenerateKnownProvidersYAML(catalog)
	assert.Equal(t, string(generated), string(plugincatalog.GenerateKnownProvidersYAML(catalog)))

	var generatedDoc, legacyDoc struct {
		Providers map[string]string `yaml:"providers"`
	}
	require.NoError(t, yaml.Unmarshal(generated, &generatedDoc))
	legacyBytes, err := ext.ReadFile(names.KnownProvidersTemplatePath)
	require.NoError(t, err)
	require.NoError(t, yaml.Unmarshal(legacyBytes, &legacyDoc))

	assert.Equal(t, legacyDoc.Providers, generatedDoc.Providers)
}

func TestLoadKnownProvidersPrefersPluginCatalog(t *testing.T) {
	t.Parallel()

	got := plugincatalog.LoadKnownProviders(catalogOnlyExtractor{catalog: []byte(`
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

	got, err := plugincatalog.ResolveInstallableDefaultProviders(catalogOnlyExtractor{catalog: []byte(`
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

	got, err := plugincatalog.ResolveInstallableDefaultProviders(catalogOnlyExtractor{catalog: []byte(`
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
	_, err := plugincatalog.ParseCatalogBytes([]byte(`
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

	got, err := plugincatalog.ResolveInstallableDefaultProviders(installtest.PartialExtractor{FailPath: names.PluginCatalogPath})

	require.Error(t, err)
	require.ErrorContains(t, err, "resolve installable default providers")
	assert.Nil(t, got)
}

func TestPluginCatalogFeedsDeterministicResolverLock(t *testing.T) {
	t.Parallel()

	ext := installtest.DefaultsExtractor{}
	catalog, err := plugincatalog.LoadPluginCatalog(ext)
	require.NoError(t, err)
	candidates := plugincatalog.CatalogResolverCandidates(catalog)

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

	ext := installtest.DefaultsExtractor{}
	catalog, err := plugincatalog.LoadPluginCatalog(ext)
	require.NoError(t, err)

	provider, ok := plugincatalog.FindCatalogProvider(catalog, "brainstorming")
	require.True(t, ok)
	generated, err := plugincatalog.NormalizedDigestManifest(catalog, "brainstorming")
	require.NoError(t, err)

	sum := sha256.Sum256(generated)
	assert.Equal(t, fmt.Sprintf("sha256:%x", sum), plugincatalog.CatalogProviderDigest(provider))
}

func TestNormalizedDigestManifestIncludesLegacyContractAndAuxiliaryTools(t *testing.T) {
	t.Parallel()
	manifest := plugincatalog.NormalizedDigestManifestFor(plugincatalog.Provider{
		ID: "fixture", Version: "1.0.0", SchemaVersion: "v1", Status: "active",
		RiskScore: "write_analysis", Category: "test", CanonicalRole: "ranger",
		Roles: []string{"ranger"}, ScratchRoot: "runtime", Description: "first line\n\nlast line",
		AuxiliaryTools: []string{"helper-a", "helper-b"},
		WeaponContract: plugincatalog.WeaponContract{
			RoleOwner: "ranger", Participation: "required", InvocationEvidence: "required",
			UnavailableBehavior: "role_invocation_failed", NativeSubstitution: "forbidden",
		},
	})
	text := string(manifest)
	assert.Contains(t, text, "weapon_contract:")
	assert.Contains(t, text, "\nauxiliary_tools_allowed:\n  - helper-a\n  - helper-b\n")
	assert.Contains(t, text, "description: >\n  first line")
}

func TestNormalizedDigestManifestRejectsAnUnknownProvider(t *testing.T) {
	t.Parallel()
	_, err := plugincatalog.NormalizedDigestManifest(plugincatalog.PluginCatalog{}, "missing")
	require.ErrorContains(t, err, `plugin catalog: provider "missing" not found`)
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
	if relPath == names.PluginCatalogPath {
		return c.catalog, nil
	}
	return nil, assert.AnError
}
