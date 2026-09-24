package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeExternalSkill(t *testing.T, root, id, canonicalRole, riskScore string, auxTools []string) string {
	t.Helper()
	dir := filepath.Join(root, id)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	skillMD := "---\nname: " + id + "\ndescription: Test skill " + id + ".\nmetadata:\n  version: \"1.0.0\"\n  author: test\n---\nbody\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillMD), 0o644))

	adapter := "canonical_role: " + canonicalRole + "\nroles:\n  - " + canonicalRole + "\nkind: atomic\nsupported_slots:\n  - " + map[string]string{"ranger": "discovery", "archivist": "refinement", "sniper": "execution"}[canonicalRole] + "\nruntime:\n  kind: host\n  host_api: strategist-host-skill/v1\nrisk_score: " + riskScore + "\ncategory: test\n"
	if len(auxTools) > 0 {
		adapter += "auxiliary_tools_allowed:\n"
		for _, tool := range auxTools {
			adapter += "  - " + tool + "\n"
		}
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "strategist.yaml"), []byte(adapter), 0o644))
	return dir
}

func TestIngestExternalSkillsAcceptsValidPackage(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeExternalSkill(t, root, "sample-skill", "ranger", "write_analysis", nil)

	result, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})
	require.NoError(t, err)
	require.Len(t, result.Ingested, 1)
	assert.Empty(t, result.Rejected)
	assert.Equal(t, "sample-skill", result.Ingested[0].ID)

	require.Len(t, result.Catalog.Providers, 1)
	assert.Equal(t, "ranger", result.Catalog.Providers[0].CanonicalRole)
	assert.Equal(t, "embedded", result.Catalog.Providers[0].CompatibilitySource)
}

func TestIngestExternalSkillsCarriesSupportedHandoffSchemas(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, "sample-skill")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	skillMD := "---\nname: sample-skill\ndescription: Test skill.\nmetadata:\n  version: \"1.0.0\"\n  author: test\n---\nbody\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillMD), 0o644))
	adapter := "canonical_role: archivist\nroles:\n  - archivist\nkind: atomic\nsupported_slots:\n  - refinement\nruntime:\n  kind: host\n  host_api: strategist-host-skill/v1\nrisk_score: write_analysis\ncategory: test\nsupported_handoff_schemas:\n  - example-schema.yaml\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "strategist.yaml"), []byte(adapter), 0o644))

	result, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})
	require.NoError(t, err)
	require.Len(t, result.Ingested, 1)
	assert.Equal(t, []string{"example-schema.yaml"}, result.Ingested[0].Adapter.SupportedHandoffSchemas)

	require.Len(t, result.Catalog.Providers, 1)
	assert.Equal(t, []string{"example-schema.yaml"}, result.Catalog.Providers[0].SupportedHandoffSchemas)
}

func TestIngestExternalSkillsCarriesScratchRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, "sample-skill")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	skillMD := "---\nname: sample-skill\ndescription: Test skill.\nmetadata:\n  version: \"1.0.0\"\n  author: test\n---\nbody\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillMD), 0o644))
	adapter := "canonical_role: archivist\nroles:\n  - archivist\nkind: atomic\nsupported_slots:\n  - refinement\nruntime:\n  kind: host\n  host_api: strategist-host-skill/v1\nrisk_score: write_analysis\ncategory: test\nscratch_root: runtime\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "strategist.yaml"), []byte(adapter), 0o644))

	result, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})
	require.NoError(t, err)
	require.Len(t, result.Ingested, 1)
	assert.Equal(t, "runtime", result.Ingested[0].Adapter.ScratchRoot)

	require.Len(t, result.Catalog.Providers, 1)
	assert.Equal(t, "runtime", result.Catalog.Providers[0].ScratchRoot)
}

func TestIngestExternalSkillsCataloguesFormerAuxiliarySkillAsWeapon(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, "writing-plans")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	skillMD := "---\nname: writing-plans\ndescription: Auxiliary planning tool.\nmetadata:\n  version: \"1.0.0\"\n  author: test\n---\nbody\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillMD), 0o644))
	adapter := "canonical_role: archivist\nroles:\n  - archivist\nkind: atomic\nsupported_slots:\n  - refinement\nruntime:\n  kind: host\n  host_api: strategist-host-skill/v1\nrisk_score: write_analysis\ncategory: auxiliary\nscratch_root: runtime\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "strategist.yaml"), []byte(adapter), 0o644))

	result, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})
	require.NoError(t, err)
	require.Len(t, result.Ingested, 1)
	assert.Equal(t, "archivist", result.Catalog.Providers[0].CanonicalRole)
	assert.Equal(t, []string{"archivist"}, result.Ingested[0].Adapter.Roles)
	assert.Equal(t, "runtime", result.Catalog.Providers[0].ScratchRoot)
}

func TestIngestExternalSkillsWithoutScratchRootIsUnaffected(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeExternalSkill(t, root, "sample-skill", "ranger", "write_analysis", nil)

	result, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})
	require.NoError(t, err)
	require.Len(t, result.Ingested, 1)
	assert.Empty(t, result.Ingested[0].Adapter.ScratchRoot)

	require.Len(t, result.Catalog.Providers, 1)
	assert.Empty(t, result.Catalog.Providers[0].ScratchRoot)
}

func TestIngestExternalSkillsRejectsInvalidScratchRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, "sample-skill")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	skillMD := "---\nname: sample-skill\ndescription: Test skill.\nmetadata:\n  version: \"1.0.0\"\n  author: test\n---\nbody\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillMD), 0o644))
	adapter := "canonical_role: archivist\nroles:\n  - archivist\nkind: atomic\nsupported_slots:\n  - refinement\nruntime:\n  kind: host\n  host_api: strategist-host-skill/v1\nrisk_score: write_analysis\ncategory: test\nscratch_root: workspace\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "strategist.yaml"), []byte(adapter), 0o644))

	result, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})
	require.NoError(t, err)
	assert.Empty(t, result.Ingested)
	require.Len(t, result.Rejected, 1)
	assert.Contains(t, result.Rejected[0].Reason, "scratch_root")
}

func TestIngestExternalSkillsRejectsIDShadowingAgainstNonEmbeddedEntry(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeExternalSkill(t, root, "sniper", "sniper", "controlled", nil)

	existing := pluginCatalog{SchemaVersion: "v1", Providers: []pluginCatalogProvider{
		{ID: "sniper", RiskScore: "controlled", CompatibilitySource: "native_role"},
	}}

	result, err := IngestExternalSkills(root, existing, domain.TrustPolicy{})
	require.NoError(t, err)
	assert.Empty(t, result.Ingested)
	require.Len(t, result.Rejected, 1)
	assert.Contains(t, result.Rejected[0].Reason, "id_shadowing")
}

func TestIngestExternalSkillsIsIdempotentAcrossReRuns(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeExternalSkill(t, root, "sample-skill", "ranger", "write_analysis", nil)

	first, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})
	require.NoError(t, err)
	require.Len(t, first.Ingested, 1)

	// Re-run against the catalog the first run produced — must not reject
	// the same id as a collision (that would break re-runnability), and must
	// reproduce the same catalog content.
	second, err := IngestExternalSkills(root, first.Catalog, domain.TrustPolicy{})
	require.NoError(t, err)
	assert.Empty(t, second.Rejected)
	require.Len(t, second.Ingested, 1)
	assert.Equal(t, first.Catalog, second.Catalog)
}

func TestIngestExternalSkillsRejectsLegacyAuxiliaryDependency(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeExternalSkill(t, root, "master-skill", "ranger", "write_analysis", []string{"helper-skill"})

	result, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})
	require.NoError(t, err)
	assert.Empty(t, result.Ingested)
	require.Len(t, result.Rejected, 1)
	assert.Equal(t, "master-skill", result.Rejected[0].ID)
	assert.Contains(t, result.Rejected[0].Reason, "auxiliary_tools_allowed is legacy")
}

func TestIngestExternalSkillsDoesNotUseLegacyAuxiliaryDependency(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeExternalSkill(t, root, "master-skill", "ranger", "write_analysis", []string{"helper-skill"})
	writeExternalSkill(t, root, "helper-skill", "ranger", "write_analysis", nil)

	result, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})
	require.NoError(t, err)
	assert.Len(t, result.Rejected, 1)
	assert.Contains(t, result.Rejected[0].Reason, "auxiliary_tools_allowed is legacy")
	assert.Len(t, result.Ingested, 1)
}

func TestIngestExternalSkillsResolvesCompositeWeaponComponents(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeExternalSkill(t, root, "brainstorming", "ranger", "write_analysis", nil)
	writeExternalSkill(t, root, "openspec-explore", "ranger", "write_analysis", nil)
	compositeDir := writeExternalSkill(t, root, "discovery-suite", "ranger", "write_analysis", nil)
	composite := "roles:\n  - ranger\nkind: composite\nsupported_slots:\n  - discovery\nruntime:\n  kind: host\n  host_api: strategist-host-skill/v1\nrisk_score: write_analysis\ncategory: discovery\ncomposition:\n  strategy: ordered_dag\n  components:\n    - id: brainstorming\n      required: true\n    - id: openspec-explore\n      required: false\n      depends_on:\n        - brainstorming\n"
	require.NoError(t, os.WriteFile(filepath.Join(compositeDir, "strategist.yaml"), []byte(composite), 0o644))

	result, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})
	require.NoError(t, err)
	assert.Empty(t, result.Rejected)
	assert.Len(t, result.Ingested, 3)
	compositeProvider, ok := findCatalogProvider(result.Catalog, "discovery-suite")
	require.True(t, ok)
	assert.Equal(t, domain.WeaponKindComposite, compositeProvider.Kind)
	require.NotNil(t, compositeProvider.Composition)
	assert.Equal(t, []string{"brainstorming", "openspec-explore"}, []string{compositeProvider.Composition.Components[0].ID, compositeProvider.Composition.Components[1].ID})
}

func TestIngestExternalSkillsRejectsUntrustedPublisher(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeExternalSkill(t, root, "sample-skill", "ranger", "write_analysis", nil)

	policy := domain.TrustPolicy{TrustedPublishers: []string{"someone-else"}}
	result, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, policy)
	require.NoError(t, err)
	assert.Empty(t, result.Ingested)
	require.Len(t, result.Rejected, 1)
	assert.Contains(t, result.Rejected[0].Reason, "trust_verification_failed")
}

func TestIngestExternalSkillsRejectsMissingAdapterFields(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dir := filepath.Join(root, "sample-skill")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: sample-skill\nmetadata:\n  version: \"1.0.0\"\n---\nbody\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "strategist.yaml"), []byte("category: test\n"), 0o644))

	result, err := IngestExternalSkills(root, pluginCatalog{SchemaVersion: "v1"}, domain.TrustPolicy{})
	require.NoError(t, err)
	assert.Empty(t, result.Ingested)
	require.Len(t, result.Rejected, 1)
	assert.Contains(t, result.Rejected[0].Reason, "roles")
}

func TestScanExternalSkillsSourceDirsMissingDirIsNotAnError(t *testing.T) {
	t.Parallel()

	dirs, err := ScanExternalSkillsSourceDirs(filepath.Join(t.TempDir(), "does-not-exist"))
	require.NoError(t, err)
	assert.Empty(t, dirs)
}
