package initiative

import "strings"

func escalationFor(advice Advice, result Result, assessment ResultAssessment) *EscalationRequest {
	if !hasChallengeReason(assessment.Reasons) {
		return nil
	}
	signals := EscalationSignals{Evidence: "insufficient"}
	for _, reason := range assessment.Reasons {
		switch reason {
		case PreciseReasonConflictingEvidence:
			signals.Evidence, signals.ConflictingEvidence = "conflicting", true
		case PreciseReasonCriticalSecurityRisk:
			signals.Risk, signals.SecuritySensitive = "high", true
		case PreciseReasonMaterialScopeChange:
			signals.Scope, signals.ArchitecturalChange = "cross_module", true
		case PreciseReasonRepeatedFailure:
			signals.RepeatedFailures = 2
		}
	}
	material := struct {
		AdviceID, ResultID, InputDigest, LevelingEventID string
		Reasons                                          []string
	}{AdviceID: advice.AdviceID, ResultID: result.ResultID, InputDigest: assessment.InputDigest, LevelingEventID: levelingEventID(advice)}
	material.Reasons = append([]string(nil), assessment.Reasons...)
	idempotency := digest(material)
	return &EscalationRequest{
		SchemaVersion: PreciseShotSchemaVersion, Mechanism: PreciseShotMechanism,
		RequestID: "psr-" + idempotency[:24], IdempotencyKey: idempotency,
		MissionID: advice.MissionID, Role: advice.Role, RunID: advice.RunID,
		AdviceID: advice.AdviceID, ResultID: result.ResultID, ResultSequence: result.Sequence,
		AssessmentID: assessment.AssessmentID, Trigger: advice.Trigger,
		Effective: assessment.Effective, Reasons: append([]string(nil), assessment.Reasons...),
		Signals: signals, Status: EscalationRequested,
	}
}

func levelingEventID(advice Advice) string {
	if advice.Leveling == nil {
		return ""
	}
	return advice.Leveling.EventID
}

func hasCriticalReason(reasons []string) bool {
	for _, reason := range reasons {
		if reason == PreciseReasonCriticalSecurityRisk || reason == PreciseReasonConflictingEvidence || reason == PreciseReasonRepeatedFailure || strings.HasPrefix(reason, PreciseReasonBlockedObligation) {
			return true
		}
	}
	return false
}
