package initiative

import "strings"

// ConfidenceAssessment is the deterministic PRECISE-SHOT projection of a
// ResultAssessment. The legacy Challenge and ConfidenceCeiling fields remain
// first-class compatibility fields; the assessed/effective tiers are the new
// explicit values.
type ConfidenceAssessment = ResultAssessment

func assessValidated(advice Advice, result Result) ResultAssessment {
	assessment := newAssessment(advice, result)
	applyCheckStatuses(&assessment, result.Checks)
	applyEvidenceRefs(&assessment, result.EvidenceRefs)
	applyDeviations(&assessment, result.Deviations)
	applyOutcomeStatuses(&assessment, result.Outcomes)
	applyOutcomeConflicts(&assessment, result.Outcomes)
	if advice.Alignment == AlignmentBelowRecommendation {
		assessment.Assessed = lowerConfidence(assessment.Assessed, ConfidenceMedium)
		assessment.Reasons = append(assessment.Reasons, PreciseReasonEffortBelowRecommendation)
	}
	applyTriggerReasons(&assessment, advice.Trigger)
	finalizeAssessment(&assessment, advice, result)
	return assessment
}

func newAssessment(advice Advice, result Result) ResultAssessment {
	return ResultAssessment{
		SchemaVersion:     PreciseShotSchemaVersion,
		Mechanism:         PreciseShotMechanism,
		AlgorithmVersion:  PreciseShotAlgorithmVersion,
		AdviceID:          advice.AdviceID,
		PolicyVersion:     advice.PolicyVersion,
		PolicyDigest:      advice.PolicyDigest,
		ResultID:          result.ResultID,
		ResultSequence:    result.Sequence,
		CalibrationStatus: PreciseShotCalibrationNoSample,
		ConfidenceCeiling: advice.Diligence.ConfidenceCeiling,
		Ceiling:           advice.Diligence.ConfidenceCeiling,
		Assessed:          ConfidenceHigh,
		EvidenceSummary:   EvidenceSummary{Expected: len(advice.Diligence.Checks)},
	}
}

func applyCheckStatuses(assessment *ResultAssessment, checks []ObligationCheck) {
	for _, check := range checks {
		switch check.Status {
		case CheckSatisfied, CheckNotApplicable:
			assessment.EvidenceSummary.Satisfied++
		case CheckPartial:
			assessment.EvidenceSummary.Partial++
			assessment.Assessed = lowerConfidence(assessment.Assessed, ConfidenceMedium)
			assessment.Reasons = append(assessment.Reasons, PreciseReasonPartialObligation+":"+check.ID)
		case CheckBlocked:
			assessment.EvidenceSummary.Blocked++
			assessment.Assessed = ConfidenceLow
			assessment.Reasons = append(assessment.Reasons, PreciseReasonBlockedObligation+":"+check.ID)
		}
		if len(check.EvidenceRefs) == 0 && check.Status != CheckNotApplicable {
			assessment.EvidenceSummary.Missing++
		}
	}
}

func applyEvidenceRefs(assessment *ResultAssessment, refs []EvidenceRef) {
	unknownEvidence := false
	for _, ref := range refs {
		if ref.Class == "unknown" {
			unknownEvidence = true
			assessment.EvidenceSummary.Missing++
			continue
		}
		assessment.EvidenceSummary.Verified++
	}
	if unknownEvidence {
		assessment.Assessed = lowerConfidence(assessment.Assessed, ConfidenceMedium)
		assessment.Reasons = append(assessment.Reasons, PreciseReasonUnverifiedEvidence)
	}
	if len(refs) == 0 && assessment.EvidenceSummary.Missing > 0 {
		assessment.Assessed = ConfidenceLow
		// Keep the historical field's conservative projection for existing
		// consumers; Ceiling remains the authoritative Advice ceiling.
		assessment.ConfidenceCeiling = ConfidenceLow
		assessment.Reasons = append(assessment.Reasons, PreciseReasonMissingEvidence)
	}
}

func applyOutcomeConflicts(assessment *ResultAssessment, outcomes []OutcomeCorrelation) {
	if conflicts := conflictingOutcomeEvidence(outcomes); conflicts > 0 {
		assessment.EvidenceSummary.Conflicting = conflicts
		assessment.Assessed = ConfidenceLow
		assessment.Reasons = append(assessment.Reasons, PreciseReasonConflictingEvidence)
	}
}

func applyDeviations(assessment *ResultAssessment, deviations []Deviation) {
	for _, deviation := range deviations {
		assessment.Reasons = append(assessment.Reasons, PreciseReasonDeviation+":"+deviation.ObligationID)
		if isCriticalDeviation(deviation) {
			assessment.Assessed = ConfidenceLow
			continue
		}
		assessment.Assessed = lowerConfidence(assessment.Assessed, ConfidenceMedium)
	}
}

func isCriticalDeviation(deviation Deviation) bool {
	impact := strings.ToLower(strings.TrimSpace(deviation.Impact))
	reason := strings.ToLower(strings.TrimSpace(deviation.Reason))
	return impact == "critical" || impact == "high" || strings.Contains(reason, "security")
}

func applyOutcomeStatuses(assessment *ResultAssessment, outcomes []OutcomeCorrelation) {
	for _, outcome := range outcomes {
		status := strings.ToLower(strings.TrimSpace(outcome.Status))
		switch status {
		case "failed", "failure", "blocked", "error", "rejected":
			assessment.Assessed = ConfidenceLow
			assessment.Reasons = append(assessment.Reasons, PreciseReasonNegativeOutcome+":"+outcome.ID)
		}
	}
}

func finalizeAssessment(assessment *ResultAssessment, advice Advice, result Result) {
	assessment.Reasons = canonicalReasons(assessment.Reasons)
	assessment.Effective = minConfidence(assessment.Assessed, assessment.Ceiling)
	if assessment.Effective != assessment.Assessed {
		assessment.Reasons = canonicalReasons(append(assessment.Reasons, PreciseReasonConfidenceCeilingApplied))
	}
	assessment.Challenge = hasChallengeReason(assessment.Reasons)
	assessment.InputDigest, assessment.AssessmentID = assessmentIdentity(advice, result)
	assessment.Escalation = escalationFor(advice, result, *assessment)
}

func hasChallengeReason(reasons []string) bool {
	for _, reason := range reasons {
		if reason != PreciseReasonConfidenceCeilingApplied {
			return true
		}
	}
	return false
}

// AssessConfidence validates the input boundary and computes PRECISE-SHOT.
func AssessConfidence(advice Advice, result Result) (ConfidenceAssessment, error) {
	if err := result.ValidateAgainst(advice); err != nil {
		return ConfidenceAssessment{}, err
	}
	return assessValidated(advice, result), nil
}

func applyTriggerReasons(assessment *ResultAssessment, trigger Trigger) {
	switch trigger {
	case TriggerScopeChanged:
		assessment.Assessed = lowerConfidence(assessment.Assessed, ConfidenceMedium)
		assessment.Reasons = append(assessment.Reasons, PreciseReasonMaterialScopeChange)
	case TriggerSecurityRiskDiscovered:
		assessment.Assessed = ConfidenceLow
		assessment.Reasons = append(assessment.Reasons, PreciseReasonCriticalSecurityRisk)
	case TriggerConflictingEvidence:
		assessment.Assessed = ConfidenceLow
		assessment.Reasons = append(assessment.Reasons, PreciseReasonConflictingEvidence)
	case TriggerRepeatedFailure:
		assessment.Assessed = ConfidenceLow
		assessment.Reasons = append(assessment.Reasons, PreciseReasonRepeatedFailure)
	case TriggerMandatoryObligationBlocked:
		assessment.Assessed = ConfidenceLow
		assessment.Reasons = append(assessment.Reasons, PreciseReasonBlockedObligation)
	case TriggerInitial, TriggerHandoffChallenged, TriggerUserRevisionRequested:
		// These triggers carry no PRECISE-SHOT confidence reason of their own.
	}
}
