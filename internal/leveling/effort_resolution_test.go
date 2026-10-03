package leveling

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLevelProjectsToVersionedEffortResolutionArtifact(t *testing.T) {
	artifact, err := (Level{
		Role: "ranger", Provider: "CODEX", Model: "reasoning", Effort: "high",
		Source: SourcePolicy, Capability: "reasoning", PolicyVersion: 1, PolicyDigest: "sha256:policy",
	}).EffortResolutionArtifact()
	require.NoError(t, err)
	require.Equal(t, "strategist-effort-resolution/v1", artifact.SchemaVersion)
	require.Equal(t, "ranger", artifact.Role)
	require.Equal(t, "high", artifact.Effort)
}

func TestResolveLevelArtifactPreservesHostOnlyUnresolvedState(t *testing.T) {
	artifact, err := ResolveLevelArtifact(Policy{}, "", "ranger", Signals{}, Host{})
	require.NoError(t, err)
	require.Equal(t, "ranger", artifact.Role)
	require.Empty(t, artifact.Effort)
	require.Empty(t, artifact.Provider)
}
