package telemetry

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStageResolutionEventProjectsCanonicalAndLegacyFields(t *testing.T) {
	t.Parallel()

	resolution, err := domain.ResolveStage(domain.StageResolutionRequest{
		Route: domain.MissionRouteDirectExecute, Role: "sniper", Feat: "critical_hit",
		PolicyVersion: "route-resolution/v1", MissionExecution: true,
	})
	require.NoError(t, err)
	artifact, err := domain.NewStageResolutionArtifact(resolution, "analysis_move")
	require.NoError(t, err)

	event := NewStageResolutionEvent("mission-1", artifact)
	require.NoError(t, event.Validate())
	assert.Equal(t, StageResolutionEventName, event.Name)
	assert.Equal(t, StageResolutionContract, event.Attributes[AttrEventContractID])
	assert.Equal(t, domain.StageResolutionArtifactSchemaVersion, event.Attributes[AttrSchemaVersion])
	assert.Equal(t, domain.CanonicalTaxonomyVersion, event.Attributes[AttrTaxonomyVersion])
	assert.Equal(t, "SHORT", event.Attributes[AttrStage])
	assert.Equal(t, "analysis_move", event.Attributes[AttrStageTrigger])
	assert.Equal(t, "direct_execute", event.Attributes[AttrRoute])
	assert.Equal(t, "sniper", event.Attributes[AttrRole])
	assert.Equal(t, "critical_hit", event.Attributes[AttrFeat])
	assert.Equal(t, "compatibility_only", event.Attributes[AttrDeprecationState])
	assert.Equal(t, "direct_execute", event.Attributes[AttrDeprecatedValue])
	assert.Equal(t, "SHORT", event.Attributes[AttrCanonicalValue])
}
