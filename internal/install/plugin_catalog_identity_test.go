package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeVersionedExternalSkill writes a package whose directory name differs from
// its SKILL.md name, so identity must come from name+version (ADR-0061 D12).
func writeVersionedExternalSkill(t *testing.T, root, dirName, name, version string) {
	t.Helper()
	dir := filepath.Join(root, dirName)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	skillMD := "---\nname: " + name + "\ndescription: Test skill " + name + ".\nmetadata:\n  version: \"" + version + "\"\n  author: test\n---\nbody\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillMD), 0o644))
	adapter := "canonical_role: ranger\nroles:\n  - ranger\nkind: atomic\nsupported_slots:\n  - discovery\nruntime:\n  kind: host\n  host_api: strategist-host-skill/v1\nrisk_score: write_analysis\ncategory: test\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "strategist.yaml"), []byte(adapter), 0o644))
}

func catalogIdentities(catalog pluginCatalog) []string {
	identities := make([]string, 0, len(catalog.Providers))
	for _, provider := range catalog.Providers {
		identities = append(identities, providerIdentity(provider))
	}
	return identities
}

func TestIngestExternalSkillsKeepsTwoVersionsOfOneWeapon(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeVersionedExternalSkill(t, root, "demo-2.0.0", "demo", "2.0.0")
	writeVersionedExternalSkill(t, root, "demo-1.4.0", "demo", "1.4.0")

	result, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})

	require.NoError(t, err)
	assert.Empty(t, result.Rejected)
	assert.Equal(t, []string{"demo@1.4.0", "demo@2.0.0"}, catalogIdentities(result.Catalog), "sorted by id then version")
}

func TestIngestExternalSkillsRejectsTheSameIdentityTwice(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeVersionedExternalSkill(t, root, "a-copy", "demo", "1.0.0")
	writeVersionedExternalSkill(t, root, "b-copy", "demo", "1.0.0")

	result, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})

	require.NoError(t, err)
	assert.Equal(t, []string{"demo@1.0.0"}, catalogIdentities(result.Catalog))
	require.Len(t, result.Rejected, 1)
	assert.Contains(t, result.Rejected[0].Reason, "duplicate_identity")
	assert.Contains(t, result.Rejected[0].Reason, "demo@1.0.0")
}

func TestIngestExternalSkillsRefreshesOnlyTheSameIdentity(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeVersionedExternalSkill(t, root, "demo-2.0.0", "demo", "2.0.0")
	existing := pluginCatalog{SchemaVersion: "v1", Providers: []pluginCatalogProvider{
		{ID: "demo", Version: "1.0.0", RiskScore: "write_analysis", CompatibilitySource: "embedded"},
		{ID: "demo", Version: "2.0.0", RiskScore: "write_analysis", CompatibilitySource: "embedded", Description: "stale"},
	}}

	result, err := IngestExternalSkills(root, existing, domain.TrustPolicy{})

	require.NoError(t, err)
	assert.Empty(t, result.Rejected)
	assert.Equal(t, []string{"demo@1.0.0", "demo@2.0.0"}, catalogIdentities(result.Catalog), "the other version is kept, the same one is refreshed")
	for _, provider := range result.Catalog.Providers {
		assert.NotEqual(t, "stale", provider.Description)
	}
}

func TestIngestExternalSkillsShadowingIgnoresTheVersion(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeVersionedExternalSkill(t, root, "demo-3.0.0", "demo", "3.0.0")
	existing := pluginCatalog{SchemaVersion: "v1", Providers: []pluginCatalogProvider{
		{ID: "demo", Version: "1.0.0", RiskScore: "controlled", CompatibilitySource: "native_role"},
	}}

	result, err := IngestExternalSkills(root, existing, domain.TrustPolicy{})

	require.NoError(t, err)
	require.Len(t, result.Rejected, 1)
	assert.Contains(t, result.Rejected[0].Reason, "id_shadowing")
}

func TestParseCatalogBytesAcceptsVersionsButRejectsDuplicateIdentity(t *testing.T) {
	t.Parallel()
	base := "schema_version: " + pluginCatalogSchemaVersion + "\nproviders:\n"
	entry := func(version string) string {
		return "  - id: demo\n    version: \"" + version + "\"\n    risk_score: write_analysis\n"
	}

	catalog, err := parseCatalogBytes([]byte(base + entry("1.0.0") + entry("2.0.0")))
	require.NoError(t, err)
	assert.Equal(t, []string{"demo@1.0.0", "demo@2.0.0"}, catalogIdentities(catalog))

	_, err = parseCatalogBytes([]byte(base + entry("1.0.0") + entry("1.0.0")))
	require.ErrorContains(t, err, "duplicate provider \"demo@1.0.0\"")
}

func TestFindCatalogProviderRefusesAnAmbiguousID(t *testing.T) {
	t.Parallel()
	single := pluginCatalog{Providers: []pluginCatalogProvider{{ID: "demo", Version: "1.0.0"}}}
	multi := pluginCatalog{Providers: []pluginCatalogProvider{{ID: "demo", Version: "1.0.0"}, {ID: "demo", Version: "2.0.0"}}}

	_, ok := findCatalogProvider(single, "demo")
	assert.True(t, ok, "a single version resolves by id")

	_, ok = findCatalogProvider(multi, "demo")
	assert.False(t, ok, "several versions never resolve by id: no first-match, no latest")

	provider, ok := findCatalogProviderVersion(multi, "demo", "2.0.0")
	require.True(t, ok)
	assert.Equal(t, "2.0.0", provider.Version)
	_, ok = findCatalogProviderVersion(multi, "demo", "")
	assert.False(t, ok)
	_, ok = findCatalogProviderVersion(multi, "demo", "3.0.0")
	assert.False(t, ok)
	assert.Equal(t, []string{"1.0.0", "2.0.0"}, catalogProviderVersions(multi, "demo"))
}

func TestFindCatalogProviderRefResolvesIDsRefsAndCustomPackageIDs(t *testing.T) {
	t.Parallel()
	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "demo", Version: "1.4.0"}, {ID: "demo", Version: "2.0.0"},
		{ID: "solo", Version: "1.0.0"},
		{ID: "team-skill@1.2.0", Version: "1.2.0"}, // a Custom package id already carries its version
	}}

	got, ok := findCatalogProviderRef(catalog, "demo@2.0.0")
	require.True(t, ok)
	assert.Equal(t, "2.0.0", got.Version)

	_, ok = findCatalogProviderRef(catalog, "demo")
	assert.False(t, ok, "a plain id with several versions never resolves")
	_, ok = findCatalogProviderRef(catalog, "demo@9.9.9")
	assert.False(t, ok)

	got, ok = findCatalogProviderRef(catalog, "solo")
	require.True(t, ok, "a plain id resolves while one version exists")
	assert.Equal(t, "1.0.0", got.Version)

	got, ok = findCatalogProviderRef(catalog, "team-skill@1.2.0")
	require.True(t, ok, "the whole string is tried as an id first")
	assert.Equal(t, "team-skill@1.2.0", got.ID)

	assert.True(t, catalogKnowsRef(catalog, "demo"), "an ambiguous id is still a known catalog id")
	assert.True(t, catalogKnowsRef(catalog, "demo@1.4.0"))
	assert.False(t, catalogKnowsRef(catalog, "demo@9.9.9"))
	assert.False(t, catalogKnowsRef(catalog, "unknown"))
}
