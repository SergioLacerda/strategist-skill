package domain_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const manifestCatalog = `schema_version: strategist-plugin-catalog/v2
providers:
  - id: cat-weapon
    risk_score: controlled
    canonical_role: sniper
    roles: [sniper]
    scratch_root: runtime
    weapon_contract:
      role_owner: sniper
      participation: required
  - id: shared-weapon
    risk_score: write_analysis
    canonical_role: ranger
`

func manifestRoot(t *testing.T, catalog string, compat map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if catalog != "" {
		require.NoError(t, os.MkdirAll(filepath.Join(root, "plugins"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, "plugins", "catalog.yaml"), []byte(catalog), 0o644))
	}
	for id, body := range compat {
		require.NoError(t, os.MkdirAll(filepath.Join(root, "skills", id), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, "skills", id, "skill.yaml"), []byte(body), 0o644))
	}
	return root
}

// The catalog is the authority: with the compat view absent, every field a
// Weapon manifest carries is still resolved.
func TestResolveWeaponFactsReadsTheCatalogWithoutTheCompatView(t *testing.T) {
	root := manifestRoot(t, manifestCatalog, nil)

	manifest, err := domain.ResolveWeaponFacts(root, "cat-weapon")

	require.NoError(t, err)
	assert.Equal(t, domain.WeaponFactsSourceCatalog, manifest.Source)
	assert.Equal(t, "controlled", manifest.RiskScore)
	assert.Equal(t, []string{"sniper"}, manifest.Roles)
	assert.Equal(t, "runtime", manifest.ScratchRoot)
	assert.Equal(t, "sniper", manifest.WeaponContract.RoleOwner)
}

func TestResolveWeaponFactsPrefersTheCatalogOverAStaleCompatView(t *testing.T) {
	root := manifestRoot(t, manifestCatalog, map[string]string{"cat-weapon": "risk_score: write_analysis\ncanonical_role: ranger\n"})

	manifest, err := domain.ResolveWeaponFacts(root, "cat-weapon")

	require.NoError(t, err)
	assert.Equal(t, "controlled", manifest.RiskScore, "the compat view never overrides the catalog")
	assert.Equal(t, domain.WeaponFactsSourceCatalog, manifest.Source)
}

func TestResolveWeaponFactsRejectsAnUnlistedProviderEvenWhenACompatViewRemains(t *testing.T) {
	root := manifestRoot(t, manifestCatalog, map[string]string{"custom": "risk_score: write_analysis\ncanonical_role: archivist\nscratch_root: runtime\n"})

	_, err := domain.ResolveWeaponFacts(root, "custom")

	require.ErrorIs(t, err, domain.ErrWeaponFactsNotFound)
}

func TestResolveWeaponFactsRequiresCatalogOrBoundCustomPackage(t *testing.T) {
	root := manifestRoot(t, "", map[string]string{"legacy": "risk_score: controlled\nroles: [sniper]\n"})

	_, err := domain.ResolveWeaponFacts(root, "legacy")

	require.ErrorIs(t, err, domain.ErrWeaponFactsNotFound)
}

func TestResolveWeaponFactsReportsAnUnknownProvider(t *testing.T) {
	root := manifestRoot(t, manifestCatalog, nil)

	_, err := domain.ResolveWeaponFacts(root, "nobody")

	require.ErrorIs(t, err, domain.ErrWeaponFactsNotFound)
}

func TestResolveWeaponFactsRejectsAMalformedCatalog(t *testing.T) {
	root := manifestRoot(t, "providers: [unclosed", map[string]string{"legacy": "risk_score: controlled\n"})

	_, err := domain.ResolveWeaponFacts(root, "legacy")

	require.Error(t, err, "a broken authority is an error, not a silent fallback")
}

func TestResolveWeaponFactsNestedCanonicalRoleShape(t *testing.T) {
	root := manifestRoot(t, "schema_version: strategist-plugin-catalog/v2\nproviders:\n  - id: nested\n    risk_score: write_analysis\n    specialization_taxonomy:\n      canonical_role: ranger\n", nil)

	manifest, err := domain.ResolveWeaponFacts(root, "nested")

	require.NoError(t, err)
	assert.Equal(t, []string{"ranger"}, manifest.Roles)
}

func TestResolveWeaponFactsCarriesTheCatalogClassification(t *testing.T) {
	catalog := `schema_version: strategist-plugin-catalog/v2
providers:
  - id: nat
    risk_score: controlled
    compatibility_source: native_role
  - id: emb
    risk_score: write_analysis
    compatibility_source: embedded
    installable: true
    canonical_role: ranger
    supported_slots: [discovery]
    runtime:
      kind: host
      host_api: strategist-host-skill/v1
  - id: oroot
    risk_score: write_analysis
    compatibility_source: embedded
    runtime:
      kind: openspec_root
      root: .strategist/openspec
`
	root := manifestRoot(t, catalog, nil)

	native, err := domain.ResolveWeaponFacts(root, "nat")
	require.NoError(t, err)
	assert.Equal(t, "native_role", native.CompatibilitySource)
	assert.False(t, native.Installable)

	embedded, err := domain.ResolveWeaponFacts(root, "emb")
	require.NoError(t, err)
	assert.Equal(t, "embedded", embedded.CompatibilitySource)
	assert.True(t, embedded.Installable)
	assert.Equal(t, "host", embedded.RuntimeKind)
	assert.Equal(t, []string{"discovery"}, embedded.SupportedSlots)

	rooted, err := domain.ResolveWeaponFacts(root, "oroot")
	require.NoError(t, err)
	assert.Equal(t, "openspec_root", rooted.RuntimeKind)
	assert.Equal(t, ".strategist/openspec", rooted.RuntimeRoot)
}

func writeCustomPackage(t *testing.T, root, instance, adapter string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "providers", instance), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "providers", instance, "adapter.yaml"), []byte(adapter), 0o644))
}

const customAdapter = "schema_version: strategist-plugin-adapter/v1\nid: fixture-provider\nadapter_revision: 1.0.0\nplugin_api_range: \"=1\"\nsupported_slots: [refinement]\nsupported_roles: [archivist]\nentrypoints: [host.prompt]\npackage_constraint: fixture-provider@1\nrequested_permissions: [workspace.read]\nrisk_score: write_analysis\nscratch_root: runtime\n"

func TestResolveWeaponFactsReadsTheAdapterOfACustomBinding(t *testing.T) {
	root := manifestRoot(t, manifestCatalog, nil)
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("schema_version: strategist-plugin-lock-file/v1\nbindings:\n  - slot: refinement\n    installed_instance_id: fixture-provider@1.0.0\n    mode: custom\n    status: active\n"), 0o644))
	writeCustomPackage(t, root, "fixture-provider@1.0.0", customAdapter)

	facts, err := domain.ResolveWeaponFacts(root, "fixture-provider")

	require.NoError(t, err)
	assert.Equal(t, domain.WeaponFactsSourceAdapter, facts.Source)
	assert.Equal(t, "write_analysis", facts.RiskScore)
	assert.Equal(t, []string{"archivist"}, facts.Roles)
	assert.Equal(t, "runtime", facts.ScratchRoot)
	assert.Equal(t, []string{"refinement"}, facts.SupportedSlots)
	assert.Equal(t, []domain.PluginPermission{"workspace.read"}, facts.RequestedPermissions)
}

func TestResolveWeaponFactsIgnoresAnAdapterWithoutACustomBinding(t *testing.T) {
	root := manifestRoot(t, manifestCatalog, nil)
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("schema_version: strategist-plugin-lock-file/v1\nbindings:\n  - slot: refinement\n    installed_instance_id: fixture-provider@1.0.0\n    mode: ranked\n    status: active\n"), 0o644))
	writeCustomPackage(t, root, "fixture-provider@1.0.0", customAdapter)

	_, err := domain.ResolveWeaponFacts(root, "fixture-provider")

	require.ErrorIs(t, err, domain.ErrWeaponFactsNotFound, "a package that is staged but not bound as custom is not a resolved Weapon")
}

func TestResolveWeaponFactsCatalogStillBeatsTheAdapter(t *testing.T) {
	root := manifestRoot(t, manifestCatalog, nil)
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("schema_version: strategist-plugin-lock-file/v1\nbindings:\n  - slot: execution\n    installed_instance_id: cat-weapon@1.0.0\n    mode: custom\n    status: active\n"), 0o644))
	writeCustomPackage(t, root, "cat-weapon@1.0.0", customAdapter)

	facts, err := domain.ResolveWeaponFacts(root, "cat-weapon")

	require.NoError(t, err)
	assert.Equal(t, domain.WeaponFactsSourceCatalog, facts.Source)
	assert.Equal(t, "controlled", facts.RiskScore)
}

func TestResolveCustomPackageFactsReadsTheAdapterOfACustomBinding(t *testing.T) {
	root := manifestRoot(t, manifestCatalog, nil)
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("schema_version: strategist-plugin-lock-file/v1\nbindings:\n  - slot: refinement\n    installed_instance_id: fixture-provider@1.0.0\n    mode: custom\n    status: active\n"), 0o644))
	writeCustomPackage(t, root, "fixture-provider@1.0.0", customAdapter)

	facts, found, err := domain.ResolveCustomPackageFacts(root, "fixture-provider@1.0.0")

	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, domain.WeaponFactsSourceAdapter, facts.Source)
	assert.Equal(t, "write_analysis", facts.RiskScore)
	assert.Equal(t, []string{"archivist"}, facts.Roles)
	assert.Equal(t, []string{"host.prompt"}, facts.Entrypoints)
}

func TestResolveCustomPackageFactsIgnoresARankedBinding(t *testing.T) {
	root := manifestRoot(t, manifestCatalog, nil)
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("schema_version: strategist-plugin-lock-file/v1\nbindings:\n  - slot: refinement\n    installed_instance_id: fixture-provider@1.0.0\n    mode: ranked\n    status: active\n"), 0o644))
	writeCustomPackage(t, root, "fixture-provider@1.0.0", customAdapter)

	_, found, err := domain.ResolveCustomPackageFacts(root, "fixture-provider@1.0.0")

	require.NoError(t, err)
	assert.False(t, found, "a ranked binding is never a custom package")
}

func TestResolveCustomPackageFactsWithoutAStagedAdapterIsNotFound(t *testing.T) {
	root := manifestRoot(t, manifestCatalog, nil)
	require.NoError(t, os.WriteFile(filepath.Join(root, "plugins.lock"), []byte("schema_version: strategist-plugin-lock-file/v1\nbindings:\n  - slot: refinement\n    installed_instance_id: hand-made\n    status: active\n"), 0o644))

	_, found, err := domain.ResolveCustomPackageFacts(root, "hand-made")

	require.NoError(t, err)
	assert.False(t, found, "a legacy binding with no providers/ package falls through to the other resolvers")
}

const versionedManifestCatalog = `schema_version: strategist-plugin-catalog/v2
providers:
  - id: demo
    version: "1.4.0"
    risk_score: write_analysis
    canonical_role: ranger
    scratch_root: none
  - id: demo
    version: "2.0.0"
    risk_score: write_analysis
    canonical_role: ranger
    scratch_root: runtime
  - id: solo
    version: "1.0.0"
    risk_score: write_analysis
    canonical_role: ranger
`

func TestResolveWeaponFactsResolvesAnIDAtVersionReference(t *testing.T) {
	root := manifestRoot(t, versionedManifestCatalog, nil)

	older, err := domain.ResolveWeaponFacts(root, "demo@1.4.0")
	require.NoError(t, err)
	assert.Equal(t, "1.4.0", older.Version)
	assert.Equal(t, "none", older.ScratchRoot)

	newer, err := domain.ResolveWeaponFacts(root, "demo@2.0.0")
	require.NoError(t, err)
	assert.Equal(t, "2.0.0", newer.Version)
	assert.Equal(t, "runtime", newer.ScratchRoot, "each version keeps its own manifest")
}

func TestResolveWeaponFactsRefusesAPlainIDWithSeveralVersions(t *testing.T) {
	root := manifestRoot(t, versionedManifestCatalog, nil)

	_, err := domain.ResolveWeaponFacts(root, "demo")

	require.ErrorIs(t, err, domain.ErrWeaponFactsAmbiguous)
	require.ErrorContains(t, err, "demo@1.4.0")
	require.ErrorContains(t, err, "demo@2.0.0")

	solo, err := domain.ResolveWeaponFacts(root, "solo")
	require.NoError(t, err, "a plain id resolves while exactly one version exists")
	assert.Equal(t, "1.0.0", solo.Version)
}

func TestResolveWeaponFactsReportsAnUncataloguedVersion(t *testing.T) {
	root := manifestRoot(t, versionedManifestCatalog, nil)

	_, err := domain.ResolveWeaponFacts(root, "demo@9.9.9")

	require.ErrorIs(t, err, domain.ErrWeaponFactsNotFound)
}

func TestResolveWeaponFactsRejectsAnUnboundCustomNameAtVersion(t *testing.T) {
	root := manifestRoot(t, versionedManifestCatalog, map[string]string{"team-skill@1.2.0": "id: team-skill\nrisk_score: write_analysis\ncanonical_role: ranger\n"})

	_, err := domain.ResolveWeaponFacts(root, "team-skill@1.2.0")

	require.ErrorIs(t, err, domain.ErrWeaponFactsNotFound)
}

func TestWeaponRefMatchesBinding(t *testing.T) {
	ranked := domain.SlotBinding{InstalledInstanceID: "demo", WeaponVersion: "2.0.0"}
	custom := domain.SlotBinding{InstalledInstanceID: "team-skill@1.2.0"}

	assert.True(t, domain.WeaponRefMatchesBinding("demo@2.0.0", ranked))
	assert.True(t, domain.WeaponRefMatchesBinding("demo", ranked), "a plain id matches the locked id")
	assert.False(t, domain.WeaponRefMatchesBinding("demo@1.4.0", ranked), "another version never matches")
	assert.False(t, domain.WeaponRefMatchesBinding("other", ranked))
	assert.False(t, domain.WeaponRefMatchesBinding("", ranked))
	assert.True(t, domain.WeaponRefMatchesBinding("team-skill@1.2.0", custom), "a custom package id already carries its version")
	assert.False(t, domain.WeaponRefMatchesBinding("team-skill@9.9.9", custom))
}
