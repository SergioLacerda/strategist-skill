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

func hintKeys(hints []Hint) []string {
	keys := make([]string, 0, len(hints))
	for _, hint := range hints {
		keys = append(keys, hint.Criterion)
	}
	return keys
}
