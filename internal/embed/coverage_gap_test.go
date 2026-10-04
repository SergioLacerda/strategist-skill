package embed

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestRoleSourceArtifactRejectsInvalidAndUnavailableSources(t *testing.T) {
	extractor := Extractor{}
	for _, roleID := range []string{"", "../ranger", "ranger@1.0.0"} {
		_, err := extractor.RoleSourceArtifact(roleID)
		require.Error(t, err, roleID)
	}

	_, err := extractor.RoleSourceArtifact("missing")
	require.Error(t, err)
	_, err = extractor.RoleSourceArtifact("default")
	require.Error(t, err, "a role without a native skill manifest must fail closed")
}

func TestRoleSkillManifestParserRequiresStableIdentityHeader(t *testing.T) {
	for _, raw := range []string{
		"version: 1.0.0\nschema_version: skill/v1\n",
		"id: ranger\nschema_version: skill/v1\n",
		"id: ranger\nversion: 1.0.0\n",
	} {
		_, err := parseRoleSkillManifest([]byte(raw))
		require.Error(t, err)
	}

	manifest, err := parseRoleSkillManifest([]byte("id: \"ranger\"\nversion: \"1.0.0\"\nschema_version: skill/v1\nignored: value\n"))
	require.NoError(t, err)
	require.Equal(t, roleSkillManifest{ID: "ranger", Version: "1.0.0", SchemaVersion: "skill/v1"}, manifest)
}

func TestUpdateRoleSkillManifestIgnoresMalformedAndUnknownLines(t *testing.T) {
	manifest := roleSkillManifest{}
	for _, line := range []string{"not a mapping", " id: ignored", "unknown: value"} {
		updateRoleSkillManifest(&manifest, line)
	}
	require.Equal(t, roleSkillManifest{}, manifest)

	for _, line := range []string{"id: ranger", "version: 1.0.0", "schema_version: skill/v1"} {
		updateRoleSkillManifest(&manifest, line)
	}
	require.Equal(t, roleSkillManifest{ID: "ranger", Version: "1.0.0", SchemaVersion: "skill/v1"}, manifest)
}

func TestExtractorRequiresLevelingPolicy(t *testing.T) {
	require.True(t, (Extractor{}).LevelingPolicyRequired())
}

func TestEmbeddedRootErrorPaths(t *testing.T) {
	original := defaultsFS
	defaultsFS = fstest.MapFS{}
	t.Cleanup(func() { defaultsFS = original })

	_, err := (Extractor{}).AllPaths()
	require.Error(t, err)
	_, err = fs.ReadFile(DefaultsFS(), "missing")
	require.Error(t, err)
}

func TestReadRoleSkillManifestRejectsMissingFile(t *testing.T) {
	_, _, err := (Extractor{}).readRoleSkillManifest("missing")
	require.Error(t, err)
}

func TestRoleSourceArtifactValidatesInjectedSourceContent(t *testing.T) {
	original := defaultsFS
	t.Cleanup(func() { defaultsFS = original })

	defaultsFS = fstest.MapFS{
		"defaults/roles/ranger.yaml":                    {Data: []byte("role: ranger\nslot: discovery\n")},
		"defaults/internal_skills/ranger/skill.yaml":    {Data: []byte("id: archivist\nversion: 1.0.0\nschema_version: '1'\n")},
		"defaults/roles/archivist.yaml":                 {Data: []byte("role: archivist\nslot: refinement\n")},
		"defaults/internal_skills/archivist/skill.yaml": {Data: []byte("id: archivist\nversion: 1.0.0\n")},
	}
	_, err := (Extractor{}).RoleSourceArtifact("ranger")
	require.ErrorContains(t, err, "declares id")
	_, err = (Extractor{}).RoleSourceArtifact("archivist")
	require.ErrorContains(t, err, "parse internal_skills/archivist/skill.yaml")
}
