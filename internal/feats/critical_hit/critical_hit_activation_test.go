package criticalhit_test

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/domain"
	criticalhit "github.com/SergioLacerda/strategist-skill/internal/feats/critical_hit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func eligiblePlainEvidence() criticalhit.Evidence {
	return criticalhit.Evidence{
		Mode: criticalhit.ModePlain, TaskType: "analysis_move",
		SourcePath: "/workspace/.analysis/pending/card.md",
		TargetPath: "/workspace/.analysis/archived/card.md",
		BasePath:   "/workspace/.analysis", FileTypes: []string{".md"},
		RiskLevel: "low", FileCount: 1,
	}
}

func eligibleClosureEvidence() criticalhit.Evidence {
	return criticalhit.Evidence{
		Mode: criticalhit.ModeClosure, TaskType: "analysis_move",
		SourcePath: "/workspace/.analysis/refined/card",
		TargetPath: "/workspace/.analysis/done/card",
		BasePath:   "/workspace/.analysis", ExplicitCompletionClaim: true,
		EvidenceSummaryPresent: true,
	}
}

func TestActivateReturnsPassiveFeatContextAndShortRequest(t *testing.T) {
	for _, tc := range []struct {
		name     string
		evidence criticalhit.Evidence
	}{
		{name: "plain", evidence: eligiblePlainEvidence()},
		{name: "closure", evidence: eligibleClosureEvidence()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := criticalhit.Activate(criticalhit.ActivationRequest{
				MissionID: "mission-1", Role: "scout", Evidence: tc.evidence,
			})
			require.NoError(t, err)
			assert.True(t, result.Allowed)
			assert.Equal(t, criticalhit.ActivationSchemaVersion, result.SchemaVersion)
			assert.Equal(t, criticalhit.FeatID, result.Feat)
			assert.Equal(t, domain.TaxonomyFeat, result.Identity.Family)
			assert.Equal(t, criticalhit.FeatID, result.Identity.ID)
			assert.Equal(t, "mission-1:critical_hit", result.CorrelationKey)
			assert.True(t, result.GateRequired)
			assert.Equal(t, criticalhit.FeatID, result.StageRequest.Route)
			assert.Equal(t, domain.StageShort, mustResolveStage(t, result.StageRequest).Stage)
			assert.Equal(t, "scout", result.StageRequest.Role)
			assert.True(t, result.StageRequest.GateRequired)
			assert.Equal(t, "mission-1:critical_hit", result.StageRequest.CorrelationKey)
			assert.False(t, result.StageRequest.MissionExecution)
			require.NoError(t, result.Validate())
		})
	}
}

func TestActivateFailsClosedToFullStage(t *testing.T) {
	evidence := eligiblePlainEvidence()
	evidence.FileTypes = []string{".go"}
	result, err := criticalhit.Activate(criticalhit.ActivationRequest{
		MissionID: "mission-2", Role: "scout", Evidence: evidence,
	})
	require.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, "full_pipeline", result.FallbackRoute)
	assert.Equal(t, "full_pipeline", result.StageRequest.Route)
	assert.Equal(t, domain.StageFull, mustResolveStage(t, result.StageRequest).Stage)
	assert.True(t, result.GateRequired)
	require.NoError(t, result.Validate())
}

func TestActivationResultRejectsDirectExecutionMutation(t *testing.T) {
	result, err := criticalhit.Activate(criticalhit.ActivationRequest{
		MissionID: "mission-4", Role: "scout", Evidence: eligiblePlainEvidence(),
	})
	require.NoError(t, err)
	result.StageRequest.Route = domain.MissionRouteDirectExecute
	result.StageRequest.MissionExecution = true
	assert.ErrorContains(t, result.Validate(), "cannot authorize mission execution")
}

func TestActivateRequiresRoleAndMission(t *testing.T) {
	_, err := criticalhit.Activate(criticalhit.ActivationRequest{Role: "scout", Evidence: eligiblePlainEvidence()})
	require.ErrorContains(t, err, "mission_id")

	_, err = criticalhit.Activate(criticalhit.ActivationRequest{MissionID: "mission-3", Evidence: eligiblePlainEvidence()})
	assert.ErrorContains(t, err, "role")
}

func TestEvaluateEligibilityRemainsPureAndCompatible(t *testing.T) {
	evidence := eligiblePlainEvidence()
	assert.Equal(t, criticalhit.EvaluateEligibility(evidence), criticalhit.EvaluateCriticalHit(evidence))
	evidence.Mode = criticalhit.Mode("unknown")
	decision := criticalhit.EvaluateEligibility(evidence)
	assert.False(t, decision.Allowed)
	assert.Equal(t, "full_pipeline", decision.FallbackRoute)
}

func TestActivationResultValidateRejectsMalformedIdentityContextAndStage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*criticalhit.ActivationResult)
	}{
		{name: "schema", mutate: func(result *criticalhit.ActivationResult) { result.SchemaVersion = "" }},
		{name: "invalid identity family", mutate: func(result *criticalhit.ActivationResult) { result.Identity.Family = "unknown" }},
		{name: "wrong identity family", mutate: func(result *criticalhit.ActivationResult) { result.Identity.Family = domain.TaxonomyTool }},
		{name: "empty identity id", mutate: func(result *criticalhit.ActivationResult) { result.Identity.ID = "" }},
		{name: "wrong feat field", mutate: func(result *criticalhit.ActivationResult) { result.Feat = "other" }},
		{name: "missing mission", mutate: func(result *criticalhit.ActivationResult) { result.MissionID = "" }},
		{name: "missing role", mutate: func(result *criticalhit.ActivationResult) { result.Role = "" }},
		{name: "missing correlation", mutate: func(result *criticalhit.ActivationResult) { result.CorrelationKey = "" }},
		{name: "missing result gate", mutate: func(result *criticalhit.ActivationResult) { result.GateRequired = false }},
		{name: "missing stage gate", mutate: func(result *criticalhit.ActivationResult) { result.StageRequest.GateRequired = false }},
		{name: "mission execution", mutate: func(result *criticalhit.ActivationResult) { result.StageRequest.MissionExecution = true }},
		{name: "mission mismatch", mutate: func(result *criticalhit.ActivationResult) { result.StageRequest.MissionID = "other" }},
		{name: "role mismatch", mutate: func(result *criticalhit.ActivationResult) { result.StageRequest.Role = "archivist" }},
		{name: "feat mismatch", mutate: func(result *criticalhit.ActivationResult) { result.StageRequest.Feat = "other" }},
		{name: "correlation mismatch", mutate: func(result *criticalhit.ActivationResult) { result.StageRequest.CorrelationKey = "other" }},
		{name: "unresolvable route", mutate: func(result *criticalhit.ActivationResult) { result.StageRequest.Route = "unknown" }},
		{name: "eligible route mismatch", mutate: func(result *criticalhit.ActivationResult) {
			result.StageRequest.Route = domain.MissionRouteFullPipeline
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := criticalhit.Activate(criticalhit.ActivationRequest{
				MissionID: "mission-validation", Role: "scout", Evidence: eligiblePlainEvidence(),
			})
			require.NoError(t, err)
			tc.mutate(&result)
			require.Error(t, result.Validate())
		})
	}

	result, err := criticalhit.Activate(criticalhit.ActivationRequest{
		MissionID: "mission-validation", Role: "scout", Evidence: eligiblePlainEvidence(),
	})
	require.NoError(t, err)
	result.Allowed = false
	result.StageRequest.Route = criticalhit.FeatID
	require.Error(t, result.Validate())
}

func mustResolveStage(t *testing.T, request domain.StageRequest) domain.StageResolution {
	t.Helper()
	resolution, err := domain.ResolveStage(request)
	require.NoError(t, err)
	return resolution
}
