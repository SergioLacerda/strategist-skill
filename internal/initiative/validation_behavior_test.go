package initiative

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// blockedAssessment returns a persisted-style assessment that carries an
// escalation request, so every validator branch has something to inspect.
func blockedAssessment(t *testing.T) (Advice, Result, ResultAssessment) {
	t.Helper()
	advice := validAdvice()
	result := preciseResult(advice, CheckBlocked, "e-1")
	assessment, err := AssessConfidence(advice, result)
	require.NoError(t, err)
	require.NotNil(t, assessment.Escalation)
	return advice, result, assessment
}

func TestValidateAssessmentRejectsEachInconsistentField(t *testing.T) {
	_, _, base := blockedAssessment(t)
	require.NoError(t, validateAssessment(base))
	for _, tc := range []struct {
		name   string
		mutate func(*ResultAssessment)
		want   string
	}{
		{"mechanism", func(a *ResultAssessment) { a.Mechanism = "other" }, "schema or version"},
		{"tampered algorithm", func(a *ResultAssessment) { a.AlgorithmVersion = "v0" }, "schema or version"},
		{"calibration", func(a *ResultAssessment) { a.CalibrationStatus = "bogus" }, "calibration status"},
		{"tier", func(a *ResultAssessment) { a.Assessed = "bogus" }, "assessed"},
		{"ceiling", func(a *ResultAssessment) { a.Effective, a.Ceiling = ConfidenceHigh, ConfidenceLow }, "exceeds ceiling"},
		{"reason count", func(a *ResultAssessment) { a.Reasons = make([]string, 65) }, "too many reason codes"},
		{"reason code", func(a *ResultAssessment) { a.Reasons = []string{"nope"} }, "unknown reason"},
		{"negative count", func(a *ResultAssessment) { a.EvidenceSummary.Missing = -1 }, "must not be negative"},
		{"escalation identity", func(a *ResultAssessment) { e := *a.Escalation; e.AssessmentID = "other"; a.Escalation = &e }, "identity mismatch"},
		{"escalation result identity", func(a *ResultAssessment) { e := *a.Escalation; e.ResultID = "other"; a.Escalation = &e }, "identity mismatch"},
		{"escalation missing request identity", func(a *ResultAssessment) { e := *a.Escalation; e.RequestID = ""; a.Escalation = &e }, "identity or trigger"},
		{"escalation status", func(a *ResultAssessment) { e := *a.Escalation; e.Status = "weird"; a.Escalation = &e }, "unknown escalation status"},
	} {
		assessment := base
		tc.mutate(&assessment)
		require.ErrorContains(t, validateAssessment(assessment), tc.want, tc.name)
	}
}

func TestValidateAssessmentAgainstRejectsStaleAndInvalidInputs(t *testing.T) {
	advice, result, persisted := blockedAssessment(t)
	require.NoError(t, ValidateAssessmentAgainst(advice, result, persisted))
	stale := persisted
	stale.Assessed = ConfidenceHigh
	require.ErrorContains(t, ValidateAssessmentAgainst(advice, result, stale), "initiative_assessment_stale")
	broken := result
	broken.GateIndependent = false
	require.ErrorContains(t, ValidateAssessmentAgainst(advice, broken, persisted), "gate_independent")
}

func TestSameEscalationComparesIdentityReasonsAndStatus(t *testing.T) {
	base := &EscalationRequest{RequestID: "r", IdempotencyKey: "k", AssessmentID: "a", Effective: ConfidenceLow, Reasons: []string{"x"}, Status: EscalationRequested}
	other := func(mutate func(*EscalationRequest)) *EscalationRequest {
		copied := *base
		mutate(&copied)
		return &copied
	}
	require.True(t, sameEscalation(nil, nil))
	require.False(t, sameEscalation(base, nil))
	require.True(t, sameEscalation(base, other(func(*EscalationRequest) {})))
	require.False(t, sameEscalation(base, other(func(e *EscalationRequest) { e.RequestID = "z" })))
	require.False(t, sameEscalation(base, other(func(e *EscalationRequest) { e.Reasons = []string{"y"} })))
	require.False(t, sameEscalation(base, other(func(e *EscalationRequest) { e.Status = "weird" })))
	require.False(t, sameStrings([]string{"a"}, []string{"a", "b"}))
	require.False(t, sameStrings([]string{"a"}, []string{"b"}))
	require.True(t, validPreciseReason(PreciseReasonPartialObligation+":inspect"))
}

func TestResultValidationRejectsEachMalformedPart(t *testing.T) {
	advice := validAdvice()
	long := strings.Repeat("x", maxInitiativeFieldBytes+1)
	dupRef := EvidenceRef{ID: "e-1", Class: "explicit"}
	for _, tc := range []struct {
		name   string
		mutate func(*Result)
		want   string
	}{
		{"duplicate evidence", func(r *Result) { r.EvidenceRefs = append(r.EvidenceRefs, dupRef) }, "duplicate evidence"},
		{"invalid class", func(r *Result) { r.EvidenceRefs[0].Class = "bogus" }, "invalid class"},
		{"oversized ref", func(r *Result) { r.EvidenceRefs[0].Fingerprint = long }, "field size limit"},
		{"short fingerprint", func(r *Result) { r.EvidenceRefs[0].Fingerprint = "abc" }, "invalid fingerprint"},
		{"duplicate check evidence", func(r *Result) { r.Checks[0].EvidenceRefs = []EvidenceRef{dupRef, dupRef} }, "duplicate check evidence"},
		{"unlinked check evidence", func(r *Result) { r.Checks[0].EvidenceRefs = []EvidenceRef{{ID: "e-9", Class: "explicit"}} }, "check \"inspect\" references unlinked"},
		{"unlinked outcome evidence", func(r *Result) {
			r.Outcomes = []OutcomeCorrelation{{ID: "o", Status: "ok", EvidenceRefs: []EvidenceRef{{ID: "e-9", Class: "explicit"}}}}
		}, "outcome \"o\" references unlinked"},
		{"invalid outcome evidence", func(r *Result) {
			r.Outcomes = []OutcomeCorrelation{{ID: "o", Status: "ok", EvidenceRefs: []EvidenceRef{{Class: "explicit"}}}}
		}, "evidence id is required"},
		{"outcome without status", func(r *Result) { r.Outcomes = []OutcomeCorrelation{{ID: "o"}} }, "outcome id and status"},
		{"oversized outcome", func(r *Result) { r.Outcomes = []OutcomeCorrelation{{ID: "o", Status: long}} }, "exceeds the field size limit"},
		{"non-positive sequence", func(r *Result) { r.Sequence = 0 }, "sequence must be positive"},
		{"orphan supersession", func(r *Result) { r.ResultID, r.Sequence, r.Supersedes = "", 0, "prior" }, "requires result_id"},
		{"satisfied without evidence", func(r *Result) { r.Checks[0].EvidenceRefs = nil }, "requires evidence"},
		{"not applicable without reason", func(r *Result) { r.Checks[0].Status, r.Checks[0].EvidenceRefs = CheckNotApplicable, nil }, "requires a reason"},
		{"blank deviation", func(r *Result) { r.Deviations = []Deviation{{ObligationID: "inspect", Impact: "", Reason: "reason"}} }, "deviation obligation_id, impact, and reason"},
		{"oversized check", func(r *Result) { r.Checks[0].Reason = long }, "check \"inspect\" exceeds"},
		{"blank check id", func(r *Result) { r.Checks[0].ID = " " }, "check id and valid status"},
	} {
		result := preciseResult(advice, CheckSatisfied, "e-1")
		tc.mutate(&result)
		require.ErrorContains(t, result.ValidateAgainst(advice), tc.want, tc.name)
	}
}

func TestResultValidationRequiresEveryExpectedCheck(t *testing.T) {
	advice := validAdvice()
	advice.Diligence.Checks = []string{"inspect", "verify"}
	result := preciseResult(advice, CheckSatisfied, "e-1")
	require.ErrorContains(t, result.ValidateAgainst(advice), "missing check \"verify\"")
}
