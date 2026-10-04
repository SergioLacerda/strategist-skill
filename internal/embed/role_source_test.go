package embed_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/SergioLacerda/strategist-skill/internal/embed"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedRoleSourceArtifactsCoverEveryBuiltInRole(t *testing.T) {
	extractor := embed.Extractor{}
	for _, roleID := range []string{"scout", "ranger", "archivist", "sniper"} {
		artifact, err := extractor.RoleSourceArtifact(roleID)
		require.NoError(t, err, roleID)
		require.NoError(t, artifact.Validate(), roleID)
		require.Equal(t, roleID, artifact.Role)
		require.NotEmpty(t, artifact.SkillVersion)
	}
}

func TestExtractedRoleSourcesPreserveCanonicalDigests(t *testing.T) {
	extractor := embed.Extractor{}
	target := t.TempDir()
	require.NoError(t, extractor.Extract(target, false))

	for _, roleID := range []string{"scout", "ranger", "archivist", "sniper"} {
		artifact, err := extractor.RoleSourceArtifact(roleID)
		require.NoError(t, err, roleID)
		roleRaw, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(artifact.RoleConfigPath)))
		require.NoError(t, err, roleID)
		skillRaw, err := os.ReadFile(filepath.Join(target, filepath.FromSlash(artifact.SkillManifestPath)))
		require.NoError(t, err, roleID)
		require.Equal(t, artifact.RoleConfigDigest, domain.SHA256Hex(roleRaw), roleID)
		require.Equal(t, artifact.SkillManifestDigest, domain.SHA256Hex(skillRaw), roleID)
	}
}
