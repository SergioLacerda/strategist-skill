package calibration

import "strings"

func rangerOperators() []Operator {
	ranger := func(name string, band Band, criteria []string, applies func(Fields) bool, apply func(Fields, Context) Fields) Operator {
		return Operator{Name: name, Stage: 1, Boundary: BoundaryRanger, Band: band, Expect: ExpectReject, Criteria: criteria, Applies: applies, Apply: apply}
	}
	return []Operator{
		ranger("vague_objective", BandMajor, []string{"objective_clear"}, func(f Fields) bool { return hasField(f, "objective") },
			func(f Fields, _ Context) Fields {
				return with(f, "objective", "Look into the topic and improve things.")
			}),
		ranger("facts_without_sources", BandMajor, []string{"facts_evidenced"}, func(f Fields) bool { return pathToken.MatchString(f["known_facts"]) },
			func(f Fields, _ Context) Fields {
				return with(f, "known_facts", strings.TrimSpace(pathToken.ReplaceAllString(f["known_facts"], "")))
			}),
		ranger("uncertainties_as_facts", BandMajor, []string{"uncertainties_explicit"}, func(f Fields) bool { return hasField(f, "uncertainties") && hasField(f, "known_facts") },
			func(f Fields, _ Context) Fields {
				out := with(f, "known_facts", f["known_facts"]+"\n"+f["uncertainties"])
				return with(out, "uncertainties", "")
			}),
		ranger("empty_refinement_focus", BandMajor, []string{"refinement_focus_actionable"}, func(f Fields) bool { return hasField(f, "recommended_refinement_focus") },
			func(f Fields, _ Context) Fields { return with(f, "recommended_refinement_focus", "") }),
	}
}
