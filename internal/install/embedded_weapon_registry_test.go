package install

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildCompiledWeaponsRejectsMissingEmbeddedPayload(t *testing.T) {
	root := t.TempDir()

	_, err := buildCompiledWeapons(pluginCatalog{Providers: []pluginCatalogProvider{{
		ID: "brainstorming", CompatibilitySource: "embedded",
	}}}, root)
	require.Error(t, err)
	require.ErrorContains(t, err, "embedded_payload_missing")
}

func TestBuildCompiledWeaponsAllowsAdaptedUpstreamPayload(t *testing.T) {
	root := t.TempDir()
	weaponDir := filepath.Join(root, "skills", "openspec-propose@0.0.0")
	require.NoError(t, os.MkdirAll(weaponDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(weaponDir, "SKILL.md"), []byte("adapted payload\n"), 0o644))

	weapons, err := buildCompiledWeapons(pluginCatalog{Providers: []pluginCatalogProvider{{
		ID: "openspec-propose", CompatibilitySource: "embedded", UpstreamContentDigest: "sha256:the-upstream-file-is-different",
	}}}, root)
	require.NoError(t, err)
	require.Len(t, weapons, 1)
	require.NotEqual(t, "sha256:the-upstream-file-is-different", weapons[0].SourceDigest)
}

func TestBuildCompiledWeaponsAllowsUnpinnedSourceForLocalFixture(t *testing.T) {
	root := t.TempDir()
	weaponDir := filepath.Join(root, "skills", "fixture@0.0.0")
	require.NoError(t, os.MkdirAll(weaponDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(weaponDir, "SKILL.md"), []byte("fixture payload\n"), 0o644))

	weapons, err := buildCompiledWeapons(pluginCatalog{Providers: []pluginCatalogProvider{{
		ID: "fixture", CompatibilitySource: "embedded",
	}}}, root)
	require.NoError(t, err)
	require.Len(t, weapons, 1)
	require.Equal(t, "fixture", weapons[0].ID)
	require.NotEmpty(t, weapons[0].SourceDigest)
}

func TestBuildCompiledWeaponsSkipsExternalSource(t *testing.T) {
	weapons, err := buildCompiledWeapons(pluginCatalog{Providers: []pluginCatalogProvider{{
		ID: "custom", CompatibilitySource: "external", UpstreamContentDigest: "sha256:ignored",
	}}}, t.TempDir())
	require.NoError(t, err)
	require.Empty(t, weapons)
}
