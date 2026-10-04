// Package handoffconsumer is the first consumer of the Provider Integration
// Mechanism: it asks a provider whether a handoff artifact conforms to its
// contract's criteria. The provider only suggests; a fixed local rule decides,
// and a miss or any failure leaves the handoff on the main path.
package handoffconsumer

import (
	"github.com/SergioLacerda/strategist-skill/internal/handoff"
	"github.com/SergioLacerda/strategist-skill/internal/integration"
)

// Aggregate question vocabulary.
const (
	AggregateKey = "handoff_acceptable"
	OptionAccept = "accept"
	OptionReject = "reject"
)

// Criterion is one property of a handoff contract, phrased as a yes/no
// question. Field names the contract field the criterion is about.
type Criterion struct {
	Key          string
	Field        string
	Instructions string
}

var rangerCriteria = []Criterion{
	{"objective_clear", "objective", "The objective states a concrete outcome rather than a topic."},
	{"facts_evidenced", "known_facts", "Each known fact states what it was verified against."},
	{"uncertainties_explicit", "uncertainties", "Unresolved points are listed as uncertainties and not asserted as facts."},
	{"refinement_focus_actionable", "recommended_refinement_focus", "The recommended refinement focus tells the next role what to do."},
}

var sniperCriteria = []Criterion{
	{"scope_bounded", "approved_scope", "The approved scope names concrete allowed and forbidden paths."},
	{"tasks_classified", "implementation_plan", "Every task carries exactly one task type from the contract's closed list."},
	{"validation_stated", "implementation_plan", "Every task states how its completion is verified."},
	{"implementation_not_materialized", "implementation_plan", "No implementation_handoff task is presented as executable documentation work."},
	{"acceptance_checkable", "acceptance_checks", "The acceptance checks can be confirmed against a command result or file content."},
}

// Criteria returns the criteria catalog of a lifecycle transition, empty for any
// other transition: only the two lifecycle-owned transitions are consumers.
func Criteria(transition string) []Criterion {
	switch transition {
	case handoff.TransitionRangerToArchivist:
		return rangerCriteria
	case handoff.TransitionArchivistToSniper:
		return sniperCriteria
	default:
		return nil
	}
}

// Fields lists the distinct contract fields the catalog reads, in order. They
// are the only fields a projection may ever carry.
func Fields(transition string) []string {
	var fields []string
	seen := map[string]bool{}
	for _, criterion := range Criteria(transition) {
		if !seen[criterion.Field] {
			seen[criterion.Field] = true
			fields = append(fields, criterion.Field)
		}
	}
	return fields
}

// Questions builds the typed questions of one pre-check: the aggregate choice
// first, whose answer carries the confidence criterion, then one noul per
// criterion, used only to explain a miss.
func Questions(transition string) []integration.Question {
	criteria := Criteria(transition)
	if len(criteria) == 0 {
		return nil
	}
	description := "Every criterion holds:"
	questions := []integration.Question{{}}
	for _, criterion := range criteria {
		description += " " + criterion.Instructions
		questions = append(questions, integration.Question{Key: criterion.Key, Kind: integration.KindNoul, Instructions: criterion.Instructions})
	}
	questions[0] = integration.Question{
		Key: AggregateKey, Kind: integration.KindChoice,
		Instructions: "Does this handoff conform to its contract and give the receiving role enough to proceed?",
		Options: []integration.Option{
			{Key: OptionAccept, Description: description},
			{Key: OptionReject, Description: "At least one criterion does not hold."},
		},
	}
	return questions
}
