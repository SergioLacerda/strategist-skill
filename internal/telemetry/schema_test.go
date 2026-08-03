package telemetry

import "testing"

func TestSchema_constants(t *testing.T) {
	constants := []string{
		AttrPhase, AttrStatus, AttrComponent, AttrSkill, AttrSelectedSkill,
		AttrArtifact, AttrArtifactPath, AttrReason, AttrCacheHit, AttrTarget,
		AttrMandates, AttrMission, AttrMissionID, AttrCorrelationID,
		AttrRuntimeMode, AttrOutputProfile, AttrGateType, AttrGateStatus,
		AttrGateResponse, AttrApprovalPolicy, AttrTransitionGroup,
		AttrCheckpointPath, AttrStartToIntakeMS, AttrIntakeToRangerMS,
		AttrIntakeToScoutMS, AttrScoutToRangerMS, AttrRangerToArchivistMS,
		AttrArchivistToGateMS, AttrGateWaitMS, AttrGateToSniperMS,
		AttrSniperToDoneMS, AttrDocumentationScope, AttrTotalWallTimeMS,
		AttrHandoffChallengeStatus, AttrHandoffChallengeCriticalFailures,
		AttrHandoffChallengeTypes, AttrTokensIn, AttrTokensOut, AttrLinesEmitted,
	}
	for _, c := range constants {
		if c == "" {
			t.Errorf("schema constant must not be empty string")
		}
	}
}

func TestSchema_values(t *testing.T) {
	if documentationScopeApprovedTargets == "" {
		t.Fatal("documentation scope value must not be empty")
	}
}
