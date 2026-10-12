package calibration

import "math"

// BoundaryCases returns the ids of the cases whose mean score lies within margin
// of the threshold, where the decision is least certain, so a run can repeat them.
func BoundaryCases(observations []Observation, threshold, margin float64) []string {
	var ids []string
	for _, stat := range Stats(observations, Filter{Audited: allStage2(observations)}) {
		if stat.Answered() > 0 && math.Abs(stat.Score()-threshold) <= margin {
			ids = append(ids, stat.ID)
		}
	}
	return ids
}

func allStage2(observations []Observation) map[string]bool {
	out := map[string]bool{}
	for _, obs := range observations {
		out[obs.CaseID] = true
	}
	return out
}
