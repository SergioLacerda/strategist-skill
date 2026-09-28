package initiative

import (
	"fmt"
	"reflect"
	"strings"
)

// ValidateAssessmentAgainst recomputes the current assessment and rejects a
// stale or tampered persisted projection.
func ValidateAssessmentAgainst(advice Advice, result Result, persisted ResultAssessment) error {
	current, err := AssessConfidence(advice, result)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(scalarAssessment(current), scalarAssessment(persisted)) ||
		!sameStrings(current.Reasons, persisted.Reasons) ||
		!sameEscalation(current.Escalation, persisted.Escalation) {
		return fmt.Errorf("initiative_assessment_stale: persisted PRECISE-SHOT assessment does not match current inputs")
	}
	return nil
}

func validateAssessment(assessment ResultAssessment) error {
	for _, check := range []func(ResultAssessment) error{
		validateAssessmentVersion, validateAssessmentTiers, validateAssessmentReasons,
		validateEvidenceSummary, validateAssessmentEscalation,
	} {
		if err := check(assessment); err != nil {
			return err
		}
	}
	return nil
}

func validateAssessmentVersion(assessment ResultAssessment) error {
	if assessment.Mechanism != PreciseShotMechanism || assessment.SchemaVersion != PreciseShotSchemaVersion ||
		assessment.AlgorithmVersion != PreciseShotAlgorithmVersion {
		return fmt.Errorf("initiative_assessment_invalid: PRECISE-SHOT schema or version is unsupported")
	}
	if strings.TrimSpace(assessment.PolicyVersion) == "" || strings.TrimSpace(assessment.PolicyDigest) == "" {
		return fmt.Errorf("initiative_assessment_invalid: policy version and digest are required")
	}
	if assessment.CalibrationStatus != PreciseShotCalibrationNoSample && assessment.CalibrationStatus != PreciseShotCalibrationUncalibrated {
		return fmt.Errorf("initiative_assessment_invalid: unsupported calibration status %q", assessment.CalibrationStatus)
	}
	return nil
}

func validateAssessmentTiers(assessment ResultAssessment) error {
	for name, tier := range map[string]ConfidenceTier{
		"assessed": assessment.Assessed, "ceiling": assessment.Ceiling,
		"effective": assessment.Effective, "confidence_ceiling": assessment.ConfidenceCeiling,
	} {
		if err := validateConfidenceTier(tier); err != nil {
			return fmt.Errorf("initiative_assessment_invalid: %s: %w", name, err)
		}
	}
	if confidenceRank(assessment.Effective) > confidenceRank(assessment.Ceiling) {
		return fmt.Errorf("initiative_assessment_invalid: effective confidence exceeds ceiling")
	}
	return nil
}

func validateAssessmentReasons(assessment ResultAssessment) error {
	if len(assessment.Reasons) > 64 {
		return fmt.Errorf("initiative_assessment_invalid: too many reason codes")
	}
	for _, reason := range assessment.Reasons {
		if !validPreciseReason(reason) {
			return fmt.Errorf("initiative_assessment_invalid: unknown reason %q", reason)
		}
	}
	return nil
}

func validateEvidenceSummary(assessment ResultAssessment) error {
	summary := assessment.EvidenceSummary
	for _, count := range []int{
		summary.Expected, summary.Satisfied, summary.Partial, summary.Blocked,
		summary.Verified, summary.Missing, summary.Invalid, summary.Conflicting,
	} {
		if count < 0 {
			return fmt.Errorf("initiative_assessment_invalid: evidence summary counts must not be negative")
		}
	}
	return nil
}

func validateAssessmentEscalation(assessment ResultAssessment) error {
	escalation := assessment.Escalation
	if escalation == nil {
		return nil
	}
	if escalation.SchemaVersion != PreciseShotSchemaVersion || escalation.Mechanism != PreciseShotMechanism ||
		escalation.AssessmentID != assessment.AssessmentID || escalation.AdviceID != assessment.AdviceID ||
		escalation.ResultID != assessment.ResultID || escalation.Effective != assessment.Effective {
		return fmt.Errorf("initiative_assessment_invalid: escalation identity mismatch")
	}
	if strings.TrimSpace(escalation.RequestID) == "" || strings.TrimSpace(escalation.IdempotencyKey) == "" ||
		strings.TrimSpace(escalation.MissionID) == "" || strings.TrimSpace(escalation.Role) == "" ||
		strings.TrimSpace(escalation.RunID) == "" || escalation.ResultSequence < 1 || !validTriggers[escalation.Trigger] {
		return fmt.Errorf("initiative_assessment_invalid: escalation identity or trigger is incomplete")
	}
	if !validEscalationStatus(escalation.Status) {
		return fmt.Errorf("initiative_assessment_invalid: unknown escalation status %q", escalation.Status)
	}
	return nil
}

func validEscalationStatus(status string) bool {
	switch status {
	case EscalationRequested, EscalationSuppressed, EscalationHumanReview, EscalationBlocked:
		return true
	default:
		return false
	}
}

func validPreciseReason(reason string) bool {
	for _, exact := range []string{
		PreciseReasonMissingEvidence, PreciseReasonInvalidEvidence, PreciseReasonStaleEvidence,
		PreciseReasonUnverifiedEvidence, PreciseReasonConflictingEvidence,
		PreciseReasonEffortBelowRecommendation, PreciseReasonMaterialScopeChange, PreciseReasonRepeatedFailure,
		PreciseReasonCriticalSecurityRisk, PreciseReasonConfidenceCeilingApplied,
	} {
		if reason == exact {
			return true
		}
	}
	return strings.HasPrefix(reason, PreciseReasonBlockedObligation+":") ||
		strings.HasPrefix(reason, PreciseReasonPartialObligation+":") ||
		strings.HasPrefix(reason, PreciseReasonDeviation+":") ||
		strings.HasPrefix(reason, PreciseReasonNegativeOutcome+":")
}

func sameEscalation(current, persisted *EscalationRequest) bool {
	if current == nil || persisted == nil {
		return current == nil && persisted == nil
	}
	if !reflect.DeepEqual(scalarEscalation(*current), scalarEscalation(*persisted)) || !sameStrings(current.Reasons, persisted.Reasons) {
		return false
	}
	return validEscalationStatus(persisted.Status)
}

// scalarAssessment drops the slice and pointer fields so the remaining
// fields can be
// compared with reflect.DeepEqual.
func scalarAssessment(a ResultAssessment) ResultAssessment {
	a.Reasons, a.Escalation = nil, nil
	return a
}

// scalarEscalation drops the slice field so the remaining fields can be
// compared with reflect.DeepEqual.
func scalarEscalation(e EscalationRequest) EscalationRequest {
	e.Reasons = nil
	return e
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
