package leveling

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestLevelProjectsToVersionedEffortResolutionArtifact(t *testing.T) {
	budget := &domain.EffortBudget{
		SchemaVersion: domain.EffortBudgetSchemaVersion, Role: "ranger",
		MaxDelegations: 1, MaxDepth: 1, Parallelism: 1,
		ContextBudget: "medium", EvidenceDepth: "standard",
		EscalationAllowed: true, MaxEscalations: 1,
	}
	artifact, err := (Level{
		Role: "ranger", Provider: "CODEX", Model: "reasoning", Effort: "high",
		Source: SourcePolicy, Capability: "reasoning", PolicyVersion: 1, PolicyDigest: "sha256:policy",
		Budget: budget,
	}).EffortResolutionArtifact()
	require.NoError(t, err)
	require.Equal(t, "strategist-effort-resolution/v1", artifact.SchemaVersion)
	require.Equal(t, "ranger", artifact.Role)
	require.Equal(t, "high", artifact.Effort)
	require.NotSame(t, budget, artifact.Budget)
	require.Equal(t, budget, artifact.Budget)
}

func TestResolveLevelArtifactPreservesHostOnlyUnresolvedState(t *testing.T) {
	artifact, err := ResolveLevelArtifact(Policy{}, "", "ranger", Signals{}, Host{})
	require.NoError(t, err)
	require.Equal(t, "ranger", artifact.Role)
	require.Empty(t, artifact.Effort)
	require.Empty(t, artifact.Provider)
}
