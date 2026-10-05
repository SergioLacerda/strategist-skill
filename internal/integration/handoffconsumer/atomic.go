package handoffconsumer

import (
	"time"

	"github.com/SergioLacerda/strategist-skill/internal/calibration"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
	"github.com/SergioLacerda/strategist-skill/internal/integration/mechanism"
)

// combineRule is how profile handoff-atomic-v1 turns its criterion answers into
// one probability of accept, kept in the profile version: every criterion must
// hold, so the lowest answer decides.
const combineRule = "the probability of accept is the lowest criterion answer"

// atomicWording replaces the catalog wording of a criterion for the atomic profile where
// that wording is a negative statement. The first atomic run answered "no" to it on every
// intact package (a mean of 0.08), reading "no implementation_handoff task..." as "there is
// no such task", so the question is asked positively and with both readings defined, as the
// pair layer asks the same relation (which it answered with an AUC of 1.0).
var atomicWording = map[string]string{
	"implementation_not_materialized": "Every task's declared type matches the work its action describes: `implementation_handoff` is building or changing executable code or behavior, " +
		"`analysis_artifact` is analysis or documentation only. A task that builds or changes code but is typed `analysis_artifact` does not match.",
}

func atomicWordingOf(criterion Criterion) string {
	if wording, ok := atomicWording[criterion.Key]; ok {
		return wording
	}
	return criterion.Instructions
}

// atomicQuestions asks one noul per criterion, each about one named field, with the
// yes and no readings stated, and no aggregate choice.
func atomicQuestions(criteria []Criterion) []integration.Question {
	questions := make([]integration.Question, 0, len(criteria))
	for _, criterion := range criteria {
		questions = append(questions, integration.Question{
			Key: criterion.Key, Kind: integration.KindNoul,
			Instructions: "About the field `" + criterion.Field + "`: does this hold? " + atomicWordingOf(criterion) +
				" Yes only when it holds for every item; no when it fails for any item." + absentNote,
		})
	}
	return questions
}

// atomicScores reads every criterion answer of a transition, in catalog order, and
// the lowest of them; ok is false when one is missing or is not a noul.
func atomicScores(transition string, result integration.Result) (map[string]float64, float64, bool) {
	scores, lowest := map[string]float64{}, 1.0
	for _, criterion := range Criteria(transition) {
		answer, found := result.Answers[criterion.Key]
		if !found || answer.Kind != integration.KindNoul {
			return nil, 0, false
		}
		scores[criterion.Key], lowest = answer.Noul, min(lowest, answer.Noul)
	}
	return scores, lowest, len(scores) > 0
}

// atomicVerdictOf reports the lowest criterion answer as the probability of accept.
func atomicVerdictOf(transition string, outcome mechanism.Outcome, latency time.Duration) calibration.Verdict {
	if outcome.Result == nil {
		return unanswered(outcome)
	}
	scores, lowest, ok := atomicScores(transition, *outcome.Result)
	if !ok {
		return calibration.Verdict{State: string(integration.StateInvalidResponse), Model: outcome.Result.Model}
	}
	decision := OptionReject
	if lowest >= satisfied {
		decision = OptionAccept
	}
	return calibration.Verdict{
		Decision: decision, Probabilities: calibration.Distribution{OptionAccept: lowest, OptionReject: 1 - lowest}, PAccept: lowest, Scores: scores,
		Model: outcome.Result.Model, Latency: latency, InputTokens: outcome.Result.Usage.InputTokens, OutputTokens: outcome.Result.Usage.OutputTokens,
	}
}

// withAggregate gives an atomic answer the aggregate answer the local rule reads: the lowest
// criterion answer as the probability of accept, accept from the half line up. The criterion
// answers stay as they are (they are the repair hints). Any other profile, and an atomic
// answer that lacks a criterion, is returned unchanged, so the rule finds no aggregate and
// never approves.
func withAggregate(profile, transition string, result integration.Result) integration.Result {
	if !calibration.IsAtomic(profile) {
		return result
	}
	_, lowest, ok := atomicScores(transition, result)
	if !ok {
		return result
	}
	choice := OptionReject
	if lowest >= satisfied {
		choice = OptionAccept
	}
	answers := make(map[string]integration.Answer, len(result.Answers)+1)
	for key, answer := range result.Answers {
		answers[key] = answer
	}
	answers[AggregateKey] = integration.Answer{
		Kind: integration.KindChoice, Choice: choice, Confidence: lowest, HasConfidence: true,
		Probabilities: map[string]float64{OptionAccept: lowest, OptionReject: 1 - lowest},
	}
	result.Answers = answers
	return result
}
