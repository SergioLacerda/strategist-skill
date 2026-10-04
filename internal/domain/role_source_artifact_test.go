package domain

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoleSourceArtifactKeepsRoleAndProjectionIdentityTogether(t *testing.T) {
	artifact, err := NewRoleSourceArtifact(
		"Ranger", "1", "1.0.0", "schemas/handoff-ranger-to-archivist.schema.yaml",
		strings.Repeat("a", 64), strings.Repeat("b", 64),
	)
	require.NoError(t, err)
	require.Equal(t, RoleSourceArtifactSchemaVersion, artifact.SchemaVersion)
	require.Equal(t, "ranger", artifact.Role)
	require.Equal(t, "roles/ranger.yaml", artifact.RoleConfigPath)
	require.Equal(t, "internal_skills/ranger/skill.yaml", artifact.SkillManifestPath)
}

func TestRoleSourceArtifactRejectsMismatchedPathsAndDigests(t *testing.T) {
	artifact, err := NewRoleSourceArtifact("ranger", "1", "1.0.0", "", strings.Repeat("a", 64), strings.Repeat("b", 64))
	require.NoError(t, err)

	artifact.RoleConfigPath = "roles/archivist.yaml"
	require.ErrorContains(t, artifact.Validate(), "role_config_path")

	artifact.RoleConfigPath = "roles/ranger.yaml"
	artifact.RoleConfigDigest = "not-a-digest"
	require.ErrorContains(t, artifact.Validate(), "role_config_digest")
}
