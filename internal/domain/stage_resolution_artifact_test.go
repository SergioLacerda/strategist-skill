package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStageResolutionArtifactRoundTripsAndReconstructsResolution(t *testing.T) {
	t.Parallel()

	resolution, err := domain.ResolveStage(domain.StageResolutionRequest{
		Route: domain.MissionRouteDirectExecute, Role: "sniper", Feat: "critical_hit",
		PolicyVersion: "route-resolution/v1", MissionID: "mission-1",
		CorrelationKey: "mission-1:critical_hit", GateRequired: true, MissionExecution: true,
	})
	require.NoError(t, err)
	artifact, err := domain.NewStageResolutionArtifact(resolution, "analysis_move")
	require.NoError(t, err)
	assert.Equal(t, domain.CanonicalTaxonomyVersion, artifact.TaxonomyVersion)
	assert.Equal(t, "mission-1:critical_hit", artifact.CorrelationKey)
	assert.True(t, artifact.GateRequired)

	raw, err := json.Marshal(artifact)
	require.NoError(t, err)
	var decoded domain.StageResolutionArtifact
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, artifact, decoded)

	reconstructed, err := decoded.Resolution()
	require.NoError(t, err)
	assert.Equal(t, resolution, reconstructed)
}

func TestStageResolutionArtifactRejectsUnknownTaxonomyVersion(t *testing.T) {
	t.Parallel()

	resolution, err := domain.ResolveStage(domain.StageResolutionRequest{Route: domain.MissionRouteFullPipeline})
	require.NoError(t, err)
	artifact, err := domain.NewStageResolutionArtifact(resolution, "general")
	require.NoError(t, err)
	artifact.TaxonomyVersion = "strategist-taxonomy/v0"
	assert.ErrorContains(t, artifact.Validate(), "unsupported taxonomy version")
}

func TestStageResolutionArtifactRequiresTriggerAndLegacyRoute(t *testing.T) {
	t.Parallel()

	resolution, err := domain.ResolveStage(domain.StageResolutionRequest{Route: domain.MissionRouteFullPipeline})
	require.NoError(t, err)
	_, err = domain.NewStageResolutionArtifact(resolution, " ")
	require.ErrorContains(t, err, "trigger")

	artifact, err := domain.NewStageResolutionArtifact(resolution, "general")
	require.NoError(t, err)
	artifact.LegacyRoute = ""
	require.ErrorContains(t, artifact.Validate(), "legacy_route")
}
