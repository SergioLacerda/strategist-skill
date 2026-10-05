package calibration

import (
	"math"
	"sort"
)

type familyAgg struct {
	dim        string
	pairs, hit int
	delta      []float64
}

func (a *familyAgg) add(s, base *pairStat) {
	d, hit := pairDelta(s, base, a.dim)
	a.pairs++
	a.hit += hit
	a.delta = append(a.delta, d)
}

// faithfulByParent indexes the faithful pair of each parent.
func faithfulByParent(stats map[string]*pairStat) map[string]*pairStat {
	out := map[string]*pairStat{}
	for _, s := range stats {
		if s.obs.Family == FamilyFaithful {
			out[s.obs.Parent] = s
		}
	}
	return out
}

// familyReports compares each pair with the faithful pair of its parent.
func familyReports(stats map[string]*pairStat) []FamilyReport {
	faithful := faithfulByParent(stats)
	byFamily := map[string]*familyAgg{}
	for _, s := range stats {
		base, ok := faithful[s.obs.Parent]
		if !ok || s.obs.Family == FamilyFaithful {
			continue
		}
		if byFamily[s.obs.Family] == nil {
			byFamily[s.obs.Family] = &familyAgg{dim: brokenDimension(s.obs.Oracle)}
		}
		byFamily[s.obs.Family].add(s, base)
	}
	return finishFamilies(byFamily)
}

func finishFamilies(byFamily map[string]*familyAgg) []FamilyReport {
	out := make([]FamilyReport, 0, len(byFamily))
	for family, a := range byFamily {
		out = append(out, FamilyReport{Family: family, Dimension: a.dim, Pairs: a.pairs, Responded: NewRate(a.hit, a.pairs), MeanDelta: Mean(a.delta)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Family < out[j].Family })
	return out
}

// brokenDimension is the single dimension an oracle says does not hold, "" for a control.
func brokenDimension(oracle map[string]bool) string {
	for dim, holds := range oracle {
		if !holds {
			return dim
		}
	}
	return ""
}

// pairDelta is the score change against the faithful sibling and whether it counts as
// a response: a drop beyond PairEpsilon on the broken dimension, or, for a control, a
// change within PairEpsilon on every dimension.
func pairDelta(s, base *pairStat, dim string) (float64, int) {
	if dim != "" {
		delta := base.mean(dim) - s.mean(dim)
		return delta, b2i(delta > PairEpsilon)
	}
	worst := 0.0
	for d := range s.scores {
		worst = math.Max(worst, math.Abs(s.mean(d)-base.mean(d)))
	}
	return worst, b2i(worst <= PairEpsilon)
}
