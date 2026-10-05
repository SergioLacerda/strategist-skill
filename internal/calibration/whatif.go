package calibration

import (
	"fmt"
	"slices"
)

// AtomicAcceptAt is the lowest criterion answer at which an atomic profile reads accept.
const AtomicAcceptAt = 0.5

// WithoutCriteria re-reads atomic observations as if the named criteria had not
// been asked: the probability of accept becomes the lowest of the remaining answers.
// It is a what-if over stored answers, never a measurement of a profile without
// those questions. A name no observation answered, or an exclusion that leaves no
// criterion, is an error, so a typo does not read as a result.
func WithoutCriteria(observations []Observation, skip []string) ([]Observation, error) {
	if len(skip) == 0 {
		return observations, nil
	}
	known := map[string]bool{}
	out := make([]Observation, len(observations))
	for i, obs := range observations {
		for name := range obs.Scores {
			known[name] = true
		}
		out[i] = withoutCriteria(obs, skip)
	}
	for _, name := range skip {
		if !known[name] {
			return nil, fmt.Errorf("no observation answered a criterion named %q (this needs the atomic profile)", name)
		}
	}
	return out, remainingError(out)
}

func remainingError(observations []Observation) error {
	for _, obs := range observations {
		if obs.Answered() && len(obs.Scores) == 0 {
			return fmt.Errorf("excluding those criteria leaves none for case %s", obs.CaseID)
		}
	}
	return nil
}

func withoutCriteria(obs Observation, skip []string) Observation {
	if len(obs.Scores) == 0 {
		return obs
	}
	kept, lowest := map[string]float64{}, 1.0
	for name, score := range obs.Scores {
		if !slices.Contains(skip, name) {
			kept[name], lowest = score, min(lowest, score)
		}
	}
	obs.Scores = kept
	if len(kept) == 0 {
		return obs
	}
	obs.PAccept, obs.Decision = lowest, "reject"
	if lowest >= AtomicAcceptAt {
		obs.Decision = "accept"
	}
	obs.Probabilities = Distribution{"accept": lowest, "reject": 1 - lowest}
	return obs
}
