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
// of accept whose confidence is strictly above the threshold. A noul answer has
// no confidence and can never approve; a missing aggregate never approves. Hints
// are returned only when the answer does not approve.
func Evaluate(transition string, result integration.Result, threshold float64) Verdict {
	aggregate, ok := result.Answers[AggregateKey]
	verdict := Verdict{}
	if ok && aggregate.HasConfidence {
		verdict.Confidence = aggregate.Confidence
	}
	verdict.Approved = ok && aggregate.Kind == integration.KindChoice && aggregate.Choice == OptionAccept &&
		aggregate.HasConfidence && aggregate.Confidence > threshold
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
