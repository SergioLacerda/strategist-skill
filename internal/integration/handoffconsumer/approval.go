package handoffconsumer

import (
	"sort"

	"github.com/SergioLacerda/strategist-skill/internal/integration"
)

// Hint names a criterion the provider judged unsatisfied. It is a repair hint
// for the agent, never evidence.
type Hint struct {
	Criterion string
	Value     float64
}

// Verdict is the local decision on a provider answer.
type Verdict struct {
	Approved   bool
	Confidence float64
	Hints      []Hint
}

// satisfied is the noul value at or above which a criterion counts as met.
const satisfied = 0.5

// Evaluate applies the fixed local rule: the aggregate answer must be a choice
// of accept whose confidence in acceptance is strictly above the threshold. That
// confidence is the provider's probability of accept when it reports one (the
// live provider's `confidence` is a separate, more conservative measure), and the
// documented confidence otherwise. A noul answer has no confidence and can never
// approve; a missing aggregate never approves. Hints are returned only when the
// answer does not approve.
func Evaluate(transition string, result integration.Result, threshold float64) Verdict {
	aggregate, ok := result.Answers[AggregateKey]
	verdict := Verdict{}
	acceptance, known := acceptanceConfidence(aggregate)
	if ok && known {
		verdict.Confidence = acceptance
	}
	verdict.Approved = ok && known && aggregate.Kind == integration.KindChoice && aggregate.Choice == OptionAccept && acceptance > threshold
	if !verdict.Approved {
		verdict.Hints = hints(transition, result)
	}
	return verdict
}

func hints(transition string, result integration.Result) []Hint {
	var found []Hint
	for _, criterion := range Criteria(transition) {
		if answer, ok := result.Answers[criterion.Key]; ok && answer.Kind == integration.KindNoul && answer.Noul < satisfied {
			found = append(found, Hint{Criterion: criterion.Key, Value: answer.Noul})
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].Criterion < found[j].Criterion })
	return found
}

// acceptanceConfidence is the confidence that the handoff conforms, and whether
// the answer carries one. The probability of accept wins when reported; otherwise
// the confidence of the chosen option is converted: a reject at 99% is 1%.
func acceptanceConfidence(aggregate integration.Answer) (float64, bool) {
	if probability, reported := aggregate.Probabilities[OptionAccept]; reported {
		return probability, true
	}
	switch {
	case !aggregate.HasConfidence:
		return 0, false
	case aggregate.Choice == OptionAccept:
		return aggregate.Confidence, true
	default:
		return 1 - aggregate.Confidence, true
	}
}
