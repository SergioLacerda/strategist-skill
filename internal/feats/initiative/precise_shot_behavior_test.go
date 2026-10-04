package initiative

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyTriggerReasonsMapsEachTrigger(t *testing.T) {
	for _, tc := range []struct {
		trigger Trigger
		tier    ConfidenceTier
		reason  string
	}{
		{TriggerScopeChanged, ConfidenceMedium, PreciseReasonMaterialScopeChange},
		{TriggerSecurityRiskDiscovered, ConfidenceLow, PreciseReasonCriticalSecurityRisk},
		{TriggerConflictingEvidence, ConfidenceLow, PreciseReasonConflictingEvidence},
		{TriggerRepeatedFailure, ConfidenceLow, PreciseReasonRepeatedFailure},
		{TriggerMandatoryObligationBlocked, ConfidenceLow, PreciseReasonBlockedObligation},
		{TriggerInitial, ConfidenceHigh, ""},
		{TriggerHandoffChallenged, ConfidenceHigh, ""},
		{TriggerUserRevisionRequested, ConfidenceHigh, ""},
	} {
		assessment := ResultAssessment{Assessed: ConfidenceHigh}
		applyTriggerReasons(&assessment, tc.trigger)
		require.Equal(t, tc.tier, assessment.Assessed, tc.trigger)
		if tc.reason == "" {
			require.Empty(t, assessment.Reasons, tc.trigger)
			continue
		}
		require.Equal(t, []string{tc.reason}, assessment.Reasons, tc.trigger)
	}
}

func TestAssessmentLowersConfidenceForUnknownEvidenceConflictsAndLowEffort(t *testing.T) {
	advice := validAdvice()
	advice.Alignment = AlignmentBelowRecommendation
	result := preciseResult(advice, CheckSatisfied, "e-1")
	result.EvidenceRefs = append(result.EvidenceRefs, EvidenceRef{ID: "e-unknown", Class: "unknown"})
	result.Outcomes = []OutcomeCorrelation{
		{ID: "o-1", Status: "passed", EvidenceRefs: []EvidenceRef{{ID: "e-1", Class: "explicit"}}},
		{ID: "o-2", Status: "failed", EvidenceRefs: []EvidenceRef{{ID: "e-1", Class: "explicit"}}},
	}
	assessment, err := AssessConfidence(advice, result)
	require.NoError(t, err)
	require.Equal(t, ConfidenceLow, assessment.Assessed)
	require.Equal(t, 1, assessment.EvidenceSummary.Conflicting)
	require.Equal(t, 1, assessment.EvidenceSummary.Missing)
	for _, reason := range []string{PreciseReasonUnverifiedEvidence, PreciseReasonConflictingEvidence, PreciseReasonEffortBelowRecommendation} {
		require.Contains(t, assessment.Reasons, reason)
	}
	require.True(t, assessment.Challenge)
	require.NotNil(t, assessment.Escalation)
	require.Equal(t, "conflicting", assessment.Escalation.Signals.Evidence)
}

func TestAssessmentTreatsNotApplicableAsSatisfiedWithoutEvidence(t *testing.T) {
	advice := validAdvice()
	result := preciseResult(advice, CheckNotApplicable, "unused")
	result.Checks[0].EvidenceRefs = nil
	result.EvidenceRefs = nil
	result.Checks[0].Reason = "not in scope"
	assessment, err := AssessConfidence(advice, result)
	require.NoError(t, err)
	require.Equal(t, ConfidenceHigh, assessment.Assessed)
	require.Equal(t, 1, assessment.EvidenceSummary.Satisfied)
	require.Zero(t, assessment.EvidenceSummary.Missing)
}

func TestAssessmentAccountsForDeviationsAndNegativeOutcomes(t *testing.T) {
	advice := validAdvice()
	result := preciseResult(advice, CheckSatisfied, "e-deviation")
	result.Deviations = []Deviation{{ObligationID: "inspect", Impact: "minor", Reason: "time-boxed"}}
	result.Outcomes = []OutcomeCorrelation{{ID: "out-failed", Status: "failed", EvidenceRefs: []EvidenceRef{{ID: "e-deviation", Class: "explicit"}}}}
	assessment, err := AssessConfidence(advice, result)
	require.NoError(t, err)
	require.Equal(t, ConfidenceLow, assessment.Assessed)
	require.Contains(t, assessment.Reasons, PreciseReasonDeviation+":inspect")
	require.Contains(t, assessment.Reasons, PreciseReasonNegativeOutcome+":out-failed")
}

func TestAssessmentPersistsPolicyIdentity(t *testing.T) {
	advice := validAdvice()
	assessment, err := AssessConfidence(advice, preciseResult(advice, CheckSatisfied, "e-policy"))
	require.NoError(t, err)
	require.Equal(t, advice.PolicyVersion, assessment.PolicyVersion)
	require.Equal(t, advice.PolicyDigest, assessment.PolicyDigest)
	mutated := assessment
	mutated.PolicyDigest = "other"
	require.ErrorContains(t, ValidateAssessmentAgainst(advice, preciseResult(advice, CheckSatisfied, "e-policy"), mutated), "initiative_assessment_stale")
}

func TestConflictingOutcomeEvidenceCountsEachDisagreeingReference(t *testing.T) {
	ref := func(id string) []EvidenceRef { return []EvidenceRef{{ID: id}} }
	require.Zero(t, conflictingOutcomeEvidence(nil))
	require.Zero(t, conflictingOutcomeEvidence([]OutcomeCorrelation{{Status: "ok", EvidenceRefs: ref("a")}, {Status: "ok", EvidenceRefs: ref("a")}}))
	require.Equal(t, 1, conflictingOutcomeEvidence([]OutcomeCorrelation{
		{Status: "ok", EvidenceRefs: ref("a")}, {Status: "bad", EvidenceRefs: ref("a")}, {Status: "ok", EvidenceRefs: ref("b")},
	}))
}

func TestEscalationForProjectsEveryReasonIntoSignals(t *testing.T) {
	advice := validAdvice()
	advice.Leveling = &LevelingResolution{EventID: "lev-1"}
	result := preciseResult(advice, CheckSatisfied, "e-1")
	require.Nil(t, escalationFor(advice, result, ResultAssessment{Reasons: []string{PreciseReasonConfidenceCeilingApplied}}))
	request := escalationFor(advice, result, ResultAssessment{
		AssessmentID: "psa-1", InputDigest: "digest", Effective: ConfidenceLow,
		Reasons: []string{
			PreciseReasonConflictingEvidence, PreciseReasonCriticalSecurityRisk,
			PreciseReasonMaterialScopeChange, PreciseReasonRepeatedFailure,
		},
	})
	require.NotNil(t, request)
	require.Equal(t, EscalationRequested, request.Status)
	require.Equal(t, EscalationSignals{
		Evidence: "conflicting", ConflictingEvidence: true, Risk: "high", SecuritySensitive: true,
		Scope: "cross_module", ArchitecturalChange: true, RepeatedFailures: 2,
	}, request.Signals)
	require.Equal(t, "lev-1", levelingEventID(advice))
	require.Empty(t, levelingEventID(Advice{}))
}

func TestHasCriticalReasonRecognizesOnlyCriticalCodes(t *testing.T) {
	for _, reason := range []string{
		PreciseReasonCriticalSecurityRisk, PreciseReasonConflictingEvidence,
		PreciseReasonRepeatedFailure, PreciseReasonBlockedObligation + ":inspect",
	} {
		require.True(t, hasCriticalReason([]string{reason}), reason)
	}
	require.False(t, hasCriticalReason([]string{PreciseReasonPartialObligation + ":inspect", PreciseReasonMissingEvidence}))
	require.False(t, hasCriticalReason(nil))
}

func TestEscalationPolicyAndBoundEscalationRejectInvalidInput(t *testing.T) {
	require.ErrorContains(t, EscalationPolicy{MaxRequests: 0}.Validate(), "max_requests")
	require.ErrorContains(t, EscalationPolicy{MaxRequests: 1, CooldownSequences: -1}.Validate(), "cooldown_sequences")
	valid := EscalationRequest{Mechanism: PreciseShotMechanism, IdempotencyKey: "k", Effective: ConfidenceLow}
	_, err := BoundEscalation(valid, EscalationState{}, EscalationPolicy{})
	require.ErrorContains(t, err, "max_requests")
	_, err = BoundEscalation(EscalationRequest{Mechanism: PreciseShotMechanism}, EscalationState{}, DefaultEscalationPolicy())
	require.ErrorContains(t, err, "idempotency key")
}

func TestEscalationStatusAppliesControlsInOrder(t *testing.T) {
	policy := EscalationPolicy{MaxRequests: 2, CooldownSequences: 1}
	request := EscalationRequest{
		IdempotencyKey: "k", ResultSequence: 2, Effective: ConfidenceLow,
		Reasons: []string{PreciseReasonCriticalSecurityRisk},
	}
	for _, tc := range []struct {
		name  string
		state EscalationState
		want  string
	}{
		{"replay", EscalationState{LastIdempotencyKey: "k"}, EscalationSuppressed},
		{"budget", EscalationState{RequestCount: 2, LastIdempotencyKey: "x"}, EscalationHumanReview},
		{"maximum effort", EscalationState{MaximumEffort: true, LastIdempotencyKey: "x"}, EscalationHumanReview},
		{"cooldown", EscalationState{LastResultSequence: 1, LastEffective: ConfidenceLow, LastIdempotencyKey: "x"}, EscalationSuppressed},
		{"fresh", EscalationState{LastResultSequence: 1, LastEffective: ConfidenceHigh, LastIdempotencyKey: "x"}, EscalationRequested},
	} {
		require.Equal(t, tc.want, escalationStatus(request, tc.state, policy), tc.name)
	}
}

func TestEscalationCooldownDoesNotSuppressFreshAdvice(t *testing.T) {
	request := EscalationRequest{AdviceID: "new-advice", IdempotencyKey: "new-key", ResultSequence: 1, Effective: ConfidenceLow}
	state := EscalationState{LastAdviceID: "old-advice", LastResultSequence: 1, LastEffective: ConfidenceLow, LastIdempotencyKey: "old-key"}
	require.Equal(t, EscalationRequested, escalationStatus(request, state, EscalationPolicy{MaxRequests: 3}))
}

func TestPreciseShotUtilitiesAreDeterministic(t *testing.T) {
	require.Equal(t, []string{"a", "b"}, canonicalReasons([]string{"b", " ", "a", "b", ""}))
	require.Zero(t, confidenceRank("bogus"))
	require.Equal(t, ConfidenceLow, minConfidence(ConfidenceHigh, ConfidenceLow))
	require.Empty(t, digest(make(chan int)))
	require.Len(t, digest("x"), 64)
	require.Equal(t, "TIRO PRECISO", PreciseShotLabel(" pt_br "))
	require.True(t, strings.HasPrefix(assessmentIDFor(t), "psa-"))
}

func assessmentIDFor(t *testing.T) string {
	t.Helper()
	advice := validAdvice()
	assessment, err := AssessConfidence(advice, preciseResult(advice, CheckSatisfied, "e-1"))
	require.NoError(t, err)
	return assessment.AssessmentID
}
