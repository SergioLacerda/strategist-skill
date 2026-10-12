package calibration

import (
	"fmt"
	"sort"
)

// PairEpsilon is the score change that counts as a response to a mutation, and
// the tolerance within which a neutral perturbation counts as unchanged.
const PairEpsilon = 0.05

// OperatorResult is the paired-mutation measure of one operator: sensitivity is
// the share of pairs whose child scored lower than its golden parent by more than
// PairEpsilon; for a neutral operator it is the share that stayed unchanged.
type OperatorResult struct {
	Operator    string `json:"operator"`
	Band        Band   `json:"band"`
	Stage       int    `json:"stage"`
	Cases       int    `json:"cases"`
	Sensitivity Rate   `json:"sensitivity"`
	Neutral     bool   `json:"neutral,omitempty"`
}

// Insensitive reports whether the judge largely ignores the operator: enough
// pairs and fewer than half responded.
func (o OperatorResult) Insensitive() bool {
	return !o.Neutral && o.Sensitivity.N >= 3 && o.Sensitivity.Value < 0.5
}

// PairedResults measures every operator over all counted cases whose parent is a
// golden case that was answered.
func PairedResults(stats []CaseStat) []OperatorResult {
	byID := map[string]CaseStat{}
	for _, stat := range stats {
		byID[stat.ID] = stat
	}
	groups := map[string]*OperatorResult{}
	for _, stat := range stats {
		if parent, ok := pairParent(stat, byID); ok {
			tally(groups, stat, parent)
		}
	}
	return finishOperators(groups)
}

// pairParent returns the golden, answered parent of an answered mutation.
func pairParent(stat CaseStat, byID map[string]CaseStat) (CaseStat, bool) {
	parent, ok := byID[stat.ParentID]
	if stat.Operator == "" || !ok || !parent.Golden {
		return CaseStat{}, false
	}
	return parent, stat.Answered() > 0 && parent.Answered() > 0
}

func tally(groups map[string]*OperatorResult, stat, parent CaseStat) {
	res := groups[stat.Operator]
	if res == nil {
		res = &OperatorResult{Operator: stat.Operator, Band: stat.Band, Stage: stat.Stage, Neutral: stat.Kind == KindNeutral}
		groups[stat.Operator] = res
	}
	res.Cases++
	res.Sensitivity.N++
	if responded(stat, parent) {
		res.Sensitivity.K++
	}
}

func responded(child, parent CaseStat) bool {
	delta := parent.Score() - child.Score()
	if child.Kind == KindNeutral {
		return delta <= PairEpsilon && delta >= -PairEpsilon
	}
	return delta > PairEpsilon
}

func finishOperators(groups map[string]*OperatorResult) []OperatorResult {
	out := make([]OperatorResult, 0, len(groups))
	for _, res := range groups {
		res.Sensitivity = NewRate(res.Sensitivity.K, res.Sensitivity.N)
		out = append(out, *res)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Operator < out[j].Operator })
	return out
}

func sampleNote(what string, have, need int) string {
	return fmt.Sprintf("%s: have %d, need at least %d to certify the limit", what, have, need)
}
