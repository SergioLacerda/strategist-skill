package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func versionedSkill(t *testing.T, id, version string) IngestedSkill {
	t.Helper()
	dir := filepath.Join(t.TempDir(), id+"-"+version)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("payload "+version+"\n"), 0o644))
	return IngestedSkill{ID: id, Dir: dir, Package: domain.PluginPackage{Version: version}}
}

func versionedCatalog(skills ...IngestedSkill) pluginCatalog {
	catalog := pluginCatalog{SchemaVersion: "v1"}
	for _, skill := range skills {
		catalog.Providers = append(catalog.Providers, pluginCatalogProvider{ID: skill.ID, Version: skill.Package.Version, RiskScore: "write_analysis"})
	}
	return catalog
}

func TestWriteSkillMirrorsKeepsTwoVersionsSideBySide(t *testing.T) {
	t.Parallel()
	defaults := t.TempDir()
	older, newer := versionedSkill(t, "demo", "1.4.0"), versionedSkill(t, "demo", "2.0.0")

	require.NoError(t, writeSkillMirrors(versionedCatalog(older, newer), []IngestedSkill{older, newer}, defaults))

	for version, want := range map[string]string{"1.4.0": "payload 1.4.0\n", "2.0.0": "payload 2.0.0\n"} {
		got, err := os.ReadFile(filepath.Join(defaults, "skills", "demo@"+version, "SKILL.md"))
		require.NoError(t, err)
		assert.Equal(t, want, string(got), "each version keeps its own payload")
		assert.NoFileExists(t, filepath.Join(defaults, "skills", "demo@"+version, "skill.yaml"))
	}
	assert.NoDirExists(t, filepath.Join(defaults, "skills", "demo"), "no id-only directory is created")
}

func TestWriteSkillMirrorsRemovesTheLegacyIDOnlyDirectory(t *testing.T) {
	t.Parallel()
	defaults := t.TempDir()
	skill := versionedSkill(t, "demo", "1.0.0")
	legacy := filepath.Join(defaults, "skills", "demo")
	require.NoError(t, os.MkdirAll(legacy, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "SKILL.md"), []byte("old layout\n"), 0o644))
	other := filepath.Join(defaults, "skills", "not-ingested")
	require.NoError(t, os.MkdirAll(other, 0o755))

	require.NoError(t, writeSkillMirrors(versionedCatalog(skill), []IngestedSkill{skill}, defaults))

	assert.NoDirExists(t, legacy, "the pre-ADR-0061 directory of an ingested id is migrated away")
	assert.DirExists(t, other, "a directory of an id this run did not ingest is never touched")
	assert.FileExists(t, filepath.Join(defaults, "skills", "demo@1.0.0", "SKILL.md"))
}

func TestMirrorsHaveDriftedWhenALegacyIDOnlyDirectoryRemains(t *testing.T) {
	t.Parallel()
	defaults, tmp := t.TempDir(), t.TempDir()
	skill := versionedSkill(t, "demo", "1.0.0")
	require.NoError(t, writeSkillMirrors(versionedCatalog(skill), []IngestedSkill{skill}, tmp))
	require.NoError(t, writeSkillMirrors(versionedCatalog(skill), []IngestedSkill{skill}, defaults))

	drifted, err := mirrorsHaveDrifted([]IngestedSkill{skill}, defaults, tmp)
	require.NoError(t, err)
	assert.False(t, drifted, "identical versioned mirrors do not drift")

	require.NoError(t, os.MkdirAll(filepath.Join(defaults, "skills", "demo"), 0o755))
	drifted, err = mirrorsHaveDrifted([]IngestedSkill{skill}, defaults, tmp)
	require.NoError(t, err)
	assert.True(t, drifted, "a leftover id-only directory is stale drift")
}

func TestBuildCompiledWeaponsReadsEachVersionsOwnPayload(t *testing.T) {
	t.Parallel()
	defaults := t.TempDir()
	older, newer := versionedSkill(t, "demo", "1.4.0"), versionedSkill(t, "demo", "2.0.0")
	require.NoError(t, writeSkillMirrors(versionedCatalog(older, newer), []IngestedSkill{older, newer}, defaults))
	catalog := pluginCatalog{Providers: []pluginCatalogProvider{
		{ID: "demo", Version: "1.4.0", CompatibilitySource: "embedded"},
		{ID: "demo", Version: "2.0.0", CompatibilitySource: "embedded"},
	}}

	weapons, err := buildCompiledWeapons(catalog, defaults)

	require.NoError(t, err)
	require.Len(t, weapons, 2)
	assert.Equal(t, "1.4.0", weapons[0].Version)
	assert.Equal(t, "2.0.0", weapons[1].Version)
	assert.NotEqual(t, weapons[0].SourceDigest, weapons[1].SourceDigest, "two versions are two distinct payload digests")
}

func TestBuildCompiledRegistryHoldsTwoCertifiedVersionsDeterministically(t *testing.T) {
	t.Parallel()
	defaults := t.TempDir()
	older, newer := versionedSkill(t, "demo", "1.4.0"), versionedSkill(t, "demo", "2.0.0")
	require.NoError(t, writeSkillMirrors(versionedCatalog(older, newer), []IngestedSkill{older, newer}, defaults))
	copyEmbeddedRoles(t, defaults)
	certified := func(version string) pluginCatalogProvider {
		return pluginCatalogProvider{
			ID: "demo", Version: version, CompatibilitySource: "embedded", RiskScore: "write_analysis", CanonicalRole: "ranger",
			Roles: []string{"ranger"}, SupportedSlots: []string{"discovery"}, Ranked: true, CertificationDigest: "sha256:cert-" + version,
			RankedBindingGeneration: 1, RankedBindingStatus: "active",
			Runtime: domain.WeaponRuntime{Kind: domain.RankedRuntimeEmbedded, ExecutionMode: domain.WeaponExecutionModePromptBridge},
		}
	}
	forward := pluginCatalog{Providers: []pluginCatalogProvider{certified("1.4.0"), certified("2.0.0")}}
	backward := pluginCatalog{Providers: []pluginCatalogProvider{certified("2.0.0"), certified("1.4.0")}}

	first, err := buildCompiledRegistry(forward, defaults)
	require.NoError(t, err)
	second, err := buildCompiledRegistry(backward, defaults)
	require.NoError(t, err)

	assert.Equal(t, first, second, "catalog order never changes the compiled registry")
	offers := first.RankedBindingsFor("ranger", "discovery")
	require.Len(t, offers, 2)
	assert.Equal(t, "1.4.0", offers[0].WeaponVersion)
	assert.Equal(t, "2.0.0", offers[1].WeaponVersion)
	assert.NotEqual(t, offers[0].BindingDigest, offers[1].BindingDigest, "the version is part of the binding digest")
	require.NoError(t, first.Validate())
}

// copyEmbeddedRoles copies the real embedded role files, which the compiled
// Role registry is built from.
func copyEmbeddedRoles(t *testing.T, defaults string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(defaults, "roles"), 0o755))
	for _, name := range []string{"default", "ranger", "archivist", "sniper"} {
		raw, err := os.ReadFile(filepath.Join("..", "embed", "defaults", "roles", name+".yaml"))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(defaults, "roles", name+".yaml"), raw, 0o644))
	}
}
