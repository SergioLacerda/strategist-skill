package handoffconsumer

import (
	"testing"

	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/stretchr/testify/require"
)

func result(aggregate integration.Answer, unsatisfied ...string) integration.Result {
	answers := map[string]integration.Answer{AggregateKey: aggregate}
	for _, criterion := range Criteria(handoff.TransitionRangerToArchivist) {
		answers[criterion.Key] = integration.Answer{Kind: integration.KindNoul, Noul: 1}
	}
	for _, key := range unsatisfied {
		answers[key] = integration.Answer{Kind: integration.KindNoul, Noul: 0}
	}
	return integration.Result{Answers: answers, Model: "jev-1.13.0"}
}

func accept(confidence float64) integration.Answer {
	return integration.Answer{Kind: integration.KindChoice, Choice: OptionAccept, Confidence: confidence, HasConfidence: true}
}

func TestApprovalRequiresConfidenceStrictlyAboveTheThreshold(t *testing.T) {
	for name, tc := range map[string]struct {
		answer integration.Answer
		want   bool
	}{
		"above":                     {accept(0.95), true},
		"just above":                {accept(0.9001), true},
		"exactly the threshold":     {accept(0.90), false},
		"below":                     {accept(0.60), false},
		"reject with certainty":     {integration.Answer{Kind: integration.KindChoice, Choice: OptionReject, Confidence: 0.99, HasConfidence: true}, false},
		"noul cannot approve":       {integration.Answer{Kind: integration.KindNoul, Noul: 1}, false},
		"choice without confidence": {integration.Answer{Kind: integration.KindChoice, Choice: OptionAccept}, false},
	} {
		t.Run(name, func(t *testing.T) {
			verdict := Evaluate(handoff.TransitionRangerToArchivist, result(tc.answer), 0.90)
			require.Equal(t, tc.want, verdict.Approved)
		})
	}
}

func TestMissingAggregateAnswerNeverApproves(t *testing.T) {
	res := result(accept(0.99))
	delete(res.Answers, AggregateKey)
	require.False(t, Evaluate(handoff.TransitionRangerToArchivist, res, 0.90).Approved)
}

func TestHintsAppearOnlyWhenNotApprovedAndListUnsatisfiedCriteria(t *testing.T) {
	approved := Evaluate(handoff.TransitionRangerToArchivist, result(accept(0.95), "objective_clear"), 0.90)
	require.True(t, approved.Approved)
	require.Empty(t, approved.Hints, "per-criterion reasons are for repair, returned only in the signal case")

	signal := Evaluate(handoff.TransitionRangerToArchivist, result(accept(0.50), "uncertainties_explicit", "objective_clear"), 0.90)
	require.False(t, signal.Approved)
	require.Equal(t, []string{"objective_clear", "uncertainties_explicit"}, hintKeys(signal.Hints), "sorted and only the unsatisfied ones")
}

func TestConfidenceIsTheConfidenceInAcceptanceNotInTheChosenOption(t *testing.T) {
	reject := integration.Answer{Kind: integration.KindChoice, Choice: OptionReject, Confidence: 0.99, HasConfidence: true}
	verdict := Evaluate(handoff.TransitionRangerToArchivist, result(reject), 0.90)
	require.False(t, verdict.Approved)
	require.InDelta(t, 0.01, verdict.Confidence, 1e-9, "a 99% reject is a 1% confidence that the handoff conforms")

	verdict = Evaluate(handoff.TransitionRangerToArchivist, result(accept(0.95)), 0.90)
	require.InDelta(t, 0.95, verdict.Confidence, 1e-9)
}

func choiceWithProbabilities(choice string, confidence, accepted float64) integration.Answer {
	return integration.Answer{
		Kind: integration.KindChoice, Choice: choice, Confidence: confidence, HasConfidence: true,
		Probabilities: map[string]float64{OptionAccept: accepted, OptionReject: 1 - accepted},
	}
}

func TestAcceptanceConfidenceIsTheProviderProbabilityOfAcceptWhenItIsReported(t *testing.T) {
	// The live provider reported confidence 0.71 with P(accept) 0.85 for a well-formed handoff:
	// `confidence` is a separate measure, and P(accept) is the belief that the handoff conforms.
	for name, tc := range map[string]struct {
		answer    integration.Answer
		threshold float64
		approved  bool
		want      float64
	}{
		"P(accept) above the threshold approves":     {choiceWithProbabilities(OptionAccept, 0.71, 0.85), 0.80, true, 0.85},
		"P(accept) exactly the threshold does not":   {choiceWithProbabilities(OptionAccept, 0.99, 0.80), 0.80, false, 0.80},
		"P(accept) below the threshold does not":     {choiceWithProbabilities(OptionAccept, 0.99, 0.60), 0.80, false, 0.60},
		"a reject choice never approves":             {choiceWithProbabilities(OptionReject, 0.99, 0.01), 0.80, false, 0.01},
		"an inconsistent reject with high P(accept)": {choiceWithProbabilities(OptionReject, 0.60, 0.95), 0.80, false, 0.95},
	} {
		t.Run(name, func(t *testing.T) {
			verdict := Evaluate(handoff.TransitionRangerToArchivist, result(tc.answer), tc.threshold)
			require.Equal(t, tc.approved, verdict.Approved)
			require.InDelta(t, tc.want, verdict.Confidence, 1e-9)
		})
	}
}

func TestWithoutProbabilitiesTheDocumentedConfidenceStillDecides(t *testing.T) {
	verdict := Evaluate(handoff.TransitionRangerToArchivist, result(accept(0.95)), 0.80)
	require.True(t, verdict.Approved, "an answer shape without probabilities keeps the confidence rule")
	require.InDelta(t, 0.95, verdict.Confidence, 1e-9)
}
