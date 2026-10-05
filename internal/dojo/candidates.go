package dojo

import "time"

// Candidate kinds: where the provider and the later outcome disagreed.
const (
	CandidateRejectedAccepted = "provider_rejected_accepted_package"
	CandidateApprovedRevised  = "provider_approved_revised_package"
)

// MissionSignal is what one provider reading and the later gate outcome of a
// mission recorded. It carries no content.
type MissionSignal struct {
	Transition  string
	Answered    bool
	Approved    bool
	Confidence  float64
	Threshold   float64
	GateOutcome string
}

// DetectCandidates returns a regression candidate for each reading that
// disagreed with the gate outcome: the provider did not approve a package the
// gate accepted (a suspected false rejection), or approved one a human then sent
// back for revision (a suspected false approval). Unanswered readings and
// unknown outcomes are never candidates.
func DetectCandidates(mission string, signals []MissionSignal, now time.Time) []Candidate {
	var out []Candidate
	for _, s := range signals {
		kind := candidateKind(s)
		if kind == "" {
			continue
		}
		out = append(out, Candidate{
			Mission: mission, Package: mission, Transition: s.Transition, Kind: kind, Confidence: s.Confidence,
			Threshold: s.Threshold, Outcome: s.GateOutcome, At: now.UTC().Format(time.RFC3339),
		})
	}
	return out
}

func candidateKind(s MissionSignal) string {
	switch {
	case !s.Answered:
		return ""
	case s.GateOutcome == "accepted" && !s.Approved:
		return CandidateRejectedAccepted
	case s.GateOutcome == "revision_requested" && s.Approved:
		return CandidateApprovedRevised
	}
	return ""
}
